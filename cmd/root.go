package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/thomaswinkler/c8y-session-1password/pkg/core"
	"github.com/thomaswinkler/c8y-session-1password/pkg/core/picker"
	"github.com/thomaswinkler/c8y-session-1password/pkg/onepassword"
)

// Version information - set by build process
var (
	Version = "1.0.0"
	Commit  = "none"
	Date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "c8y-session-1password [filter]",
	Short: "go-c8y-cli 1Password session selector",
	Long: `Smart go-c8y-cli session picker from 1Password vaults

This command provides smart filtering and selection of sessions:
- Shows interactive picker for multiple sessions matching the filter
- Automatically returns the session when filter matches exactly one item
- Filter matches against session names, item names, and URLs
- Support multiple urls per 1Password item showing one session per URL

By default, sensitive information (passwords, TOTP secrets) is obfuscated in the output.
Use --reveal to show the actual values.

Direct item access:
- Use --item flag for direct item retrieval by ID or name
- Use --uri flag for direct item retrieval using op://vault/item format
- Use --vault flag to limit searches to specific vault(s)

Pre-requisites:

 * 1Password CLI (op) - https://developer.1password.com/docs/cli/
 * Enable "Integrate with 1Password CLI" in 1Password app settings (Developer/Advanced section)
 * Use with go-c8y-cli - https://goc8ycli.netlify.app

Authentication options:
 * Interactive: Sign in to your 1Password account: op signin
 * Service Account: Set OP_SERVICE_ACCOUNT_TOKEN environment variable
 * 1Password Connect: Set OP_CONNECT_HOST and OP_CONNECT_TOKEN environment variables

Environment Variables:

 * C8YOP_VAULT - Default vault to search in (can be vault name or ID)
 * C8YOP_TAGS - Default tags to filter by (comma-separated, defaults to "c8y" if not set)
 * C8YOP_ITEM - Default item to retrieve (item ID or name)
 * C8YOP_LOG_LEVEL - Logging level (debug, info, warn, error; defaults to info)`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		vault, err := cmd.Flags().GetString("vault")
		if err != nil {
			return err
		}

		item, err := cmd.Flags().GetString("item")
		if err != nil {
			return err
		}

		// Check for op:// URI
		opURI, err := cmd.Flags().GetString("uri")
		if err != nil {
			return err
		}

		tagsFlag, err := cmd.Flags().GetString("tags")
		if err != nil {
			return err
		}

		reveal, err := cmd.Flags().GetBool("reveal")
		if err != nil {
			return err
		}

		noColor, err := cmd.Flags().GetBool("no-color")
		if err != nil {
			return err
		}

		noColorCompat, err := cmd.Flags().GetBool("noColor")
		if err != nil {
			return err
		}

		// Combine no-color flags (either --no-color or --noColor disables colors)
		noColorFinal := noColor || noColorCompat

		// Get filter argument if provided
		var filter string
		if len(args) > 0 {
			filter = args[0]
		}

		// Get default values from environment variables
		if vault == "" {
			vault = getEnvWithFallback("C8YOP_VAULT", "CYOP_VAULT")
		}

		if item == "" {
			item = getEnvWithFallback("C8YOP_ITEM", "CYOP_ITEM")
		}

		// Parse op:// URI if provided
		if opURI != "" {
			vaultFromURI, itemFromURI, err := onepassword.ParseOPURI(opURI)
			if err != nil {
				return fmt.Errorf("invalid URI: %w", err)
			}
			// URI takes precedence over individual flags
			if vault == "" || vault == vaultFromURI {
				vault = vaultFromURI
			}
			if item == "" || item == itemFromURI {
				item = itemFromURI
			}
		}

		// Get tags using helper function
		tags := parseTags(tagsFlag)

		// If we have a specific item, get it directly (vault is optional)
		if item != "" {
			client := onepassword.NewClient(vault, tags...)
			session, err := client.GetItem(vault, item)
			if err != nil {
				return err
			}

			// Get TOTP if available
			populateTOTP(session)

			return outputSession(session, reveal)
		}

		// Interactive/filtered selection mode
		client := onepassword.NewClient(vault, tags...)
		sessions, err := client.List()
		if err != nil {
			return err
		}

		if len(sessions) == 0 {
			return fmt.Errorf("no sessions found matching tags: %v", tags)
		}

		// Apply filter if provided
		filteredSessions := sessions
		if filter != "" {
			filteredSessions = core.FilterSessions(sessions, filter)
		}

		// Smart selection behavior
		if len(filteredSessions) == 0 {
			return fmt.Errorf("no sessions found matching filter: %s", filter)
		} else if len(filteredSessions) == 1 {
			// Auto-select the single matching session
			session := filteredSessions[0]
			// Populate session details and TOTP from the full session list
			populateSessionFromList(session, sessions)
			return outputSession(session, reveal)
		} else {
			// Multiple sessions found, use interactive picker
			vaultList := splitAndTrimString(vault)
			metadata := picker.PickerMetadata{
				Vaults:  vaultList,
				Tags:    tags,
				Filter:  filter,
				NoColor: noColorFinal,
			}
			session, err := picker.Pick(filteredSessions, metadata)
			if err != nil {
				return err
			}
			// Populate session details and TOTP from the full session list
			populateSessionFromList(session, sessions)
			return outputSession(session, reveal)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// setupLogging configures slog based on C8YOP_LOG_LEVEL or LOG_LEVEL environment variable
func setupLogging() {
	// Check C8YOP_LOG_LEVEL first for consistency, fallback to LOG_LEVEL
	logLevel := os.Getenv("C8YOP_LOG_LEVEL")
	if logLevel == "" {
		logLevel = os.Getenv("LOG_LEVEL")
	}
	var level slog.Level

	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo // Default to info level
	}

	// Create a new logger with the specified level
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))

	// Set as the default logger
	slog.SetDefault(logger)
}

func init() {
	setupLogging()
	rootCmd.PersistentFlags().String("vault", "", "Vault name or ID (optional - if not provided, use C8YOP_VAULT env var or use all vaults)")
	rootCmd.PersistentFlags().String("item", "", "Specific item ID or name to retrieve (defaults to C8YOP_ITEM env var)")
	rootCmd.PersistentFlags().String("uri", "", "Specific item with op://vault/item URI")
	rootCmd.PersistentFlags().String("tags", "", "Comma-separated tags to filter by (defaults to C8YOP_TAGS env var, then 'c8y')")
	rootCmd.PersistentFlags().Bool("reveal", false, "Show sensitive information like passwords and TOTP secrets in output")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output in picker")
	rootCmd.PersistentFlags().Bool("noColor", false, "Disable colored output in picker (go-c8y-cli compatibility)")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(debugColorsCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("c8y-session-1password version %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Built: %s\n", Date)
	},
}

var debugColorsCmd = &cobra.Command{
	Use:    "debug-colors",
	Hidden: true,
	Short:  "Test color compatibility across terminals",
	Run: func(cmd *cobra.Command, args []string) {
		// Use improved color detection from picker package
		profile := picker.GetTerminalColorProfile()
		fmt.Printf("Terminal: %s\n", os.Getenv("TERM"))
		fmt.Printf("Color profile: %v\n", profile)
		fmt.Printf("Color support: %s\n", profile.String())

		// Set the detected color profile
		lipgloss.SetColorProfile(profile)

		// Test title style
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
				Light: "#119D11", // Green text for light terminals
				Dark:  "#FFBE00", // Yellow text for dark terminals
			}).
			Padding(0, 1)

		// Test selected item style (blue text, no background)
		selectedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#056AD6")). // Blue text for both modes
			Bold(true)

		// Test selected description style
		selectedDescStyle := lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
				Light: "#2970B0", // Darker blue for light terminals
				Dark:  "#3A8BDB", // Lighter blue for dark terminals
			})

		// Test status message style
		statusStyle := lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
				Light: "#056AD6", // Blue for light terminals
				Dark:  "#56C8FF", // Lighter blue for dark terminals
			})

		fmt.Println("\nColor Test Results:")
		fmt.Println(titleStyle.Render("Sessions (3) • Vault: Employee • Tag: c8y"))
		fmt.Println(selectedStyle.Render("→ Selected Session Item"))
		fmt.Println("  " + selectedDescStyle.Render("│ Description line with border"))
		fmt.Println("  Normal session item")
		fmt.Println(statusStyle.Render("Status message text"))

		fmt.Println("\nIf colors appear correctly:")
		fmt.Println("✓ Title should have green text (light mode) or yellow text (dark mode)")
		fmt.Println("✓ Selected item should have blue text (#056AD6) with no background")
		fmt.Println("✓ Description border (│) should be darker blue (light mode) or light blue (dark mode)")
		fmt.Println("✓ Status message should be blue/light-blue text")
	},
}

// Helper function to populate TOTP for a session
func populateTOTP(session *core.CumulocitySession) {
	if session.TOTPSecret != "" {
		totp, err := onepassword.GetTOTPCodeFromSecret(session.TOTPSecret)
		if err == nil {
			session.TOTP = totp
		}
	}
}

// Helper function to find and populate session details from list
func populateSessionFromList(targetSession *core.CumulocitySession, allSessions []*core.CumulocitySession) {
	for _, s := range allSessions {
		if targetSession.ItemID == s.ItemID {
			targetSession.Password = s.Password
			populateTOTP(targetSession)
			break
		}
	}
}

// Helper function to get environment variable with fallback compatibility
func getEnvWithFallback(primary, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return os.Getenv(fallback)
}

// Helper function to split and trim strings from comma-separated list
func splitAndTrimString(input string) []string {
	if input == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// Filter out empty strings
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

// Helper function to parse tags from environment variables or command line
func parseTags(flagValue string) []string {
	var tags []string

	if flagValue != "" {
		tags = splitAndTrimString(flagValue)
	} else if envTags := getEnvWithFallback("C8YOP_TAGS", "CYOP_TAGS"); envTags != "" {
		tags = splitAndTrimString(envTags)
	}

	// Default to "c8y" tag if no tags specified
	if len(tags) == 0 {
		tags = []string{"c8y"}
	}

	return tags
}

// Helper function to output session as JSON
func outputSession(session *core.CumulocitySession, reveal bool) error {
	// Create a copy of the session to avoid modifying the original
	outputSession := *session

	// Obfuscate sensitive fields if reveal is false
	if !reveal {
		if outputSession.Password != "" {
			outputSession.Password = "***"
		}
		if outputSession.TOTP != "" {
			outputSession.TOTP = "***"
		}
		if outputSession.TOTPSecret != "" {
			outputSession.TOTPSecret = "***"
		}
	}

	out, err := json.MarshalIndent(&outputSession, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", out)
	return nil
}
