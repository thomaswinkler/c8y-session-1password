package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	Use:   "c8y-session-1password",
	Short: "go-c8y-cli 1Password session selector",
	Long: `Select a session from your 1Password password manager

Pre-requisites:

 * 1Password CLI (op) - https://developer.1password.com/docs/cli/

Sign in to your 1Password account from the command line

	$ op signin

Environment Variables:

 * C8YOP_VAULT - Default vault to search in (can be vault name or ID)
 * C8YOP_TAGS - Default tags to filter by (comma-separated, defaults to "c8y" if not set)
 * C8YOP_ITEM - Default item to retrieve (item ID or name)
 
 For compatibility, CYOP_* variants are also supported:
 * CYOP_VAULT - Fallback for C8YOP_VAULT
 * CYOP_TAGS - Fallback for C8YOP_TAGS (defaults to "c8y" if neither is set)
 * CYOP_ITEM - Fallback for C8YOP_ITEM

Usage modes:
 * No arguments: Interactive session selection (same as 'list' command)
 * With --vault and --item: Direct item retrieval
 * With --uri: Direct item retrieval using op://vault/item format
`,
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

		// If we have both vault and item, get the item directly
		if vault != "" && item != "" {
			client := onepassword.NewClient(vault)
			session, err := client.GetItem(vault, item)
			if err != nil {
				return err
			}

			// Get TOTP if available
			populateTOTP(session)

			return outputSession(session)
		}

		// If no specific item is requested, fall back to interactive list
		// Get tags using helper function
		tags := parseTags("")

		client := onepassword.NewClient(vault, tags...)
		sessions, err := client.List()
		if err != nil {
			return err
		}

		// Use interactive picker
		session, err := picker.Pick(sessions)
		if err != nil {
			return err
		}

		// Populate session details and TOTP from the full session list
		populateSessionFromList(session, sessions)

		return outputSession(session)
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().String("vault", "", "Vault name or ID (defaults to C8YOP_VAULT or CYOP_VAULT env var)")
	rootCmd.Flags().String("item", "", "Specific item ID or name to retrieve (defaults to C8YOP_ITEM or CYOP_ITEM env var)")
	rootCmd.Flags().String("uri", "", "op://vault/item URI to retrieve specific item")
	rootCmd.AddCommand(versionCmd)
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
func outputSession(session *core.CumulocitySession) error {
	out, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", out)
	return nil
}
