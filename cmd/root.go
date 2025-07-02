package cmd

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
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

// NativeMessagingRequest represents the JSON request from Chrome extension
type NativeMessagingRequest struct {
	Type   string   `json:"type,omitempty"` // Optional type field for special commands
	Vaults []string `json:"vaults"`
	Tags   []string `json:"tags"`
	Search string   `json:"search"`
	Reveal bool     `json:"reveal,omitempty"` // Optional flag to reveal sensitive information
}

var rootCmd = &cobra.Command{
	Use:   "c8y-session-1password [filter]",
	Short: "go-c8y-cli 1Password session selector",
	Long: `Smart go-c8y-cli session picker from 1Password vaults

This command provides smart filtering and selection of sessions:
- Shows interactive picker for multiple sessions matching the filter
- Automatically returns the session when filter matches exactly one item
- Filter matches against session names, item names, URLs, and usernames
- Support multiple urls per 1Password item showing one session per URL

By default, sensitive information (passwords, TOTP secrets) is obfuscated in the output.
Use --reveal to show the actual values.

Direct item access:
- Use --item flag for direct item retrieval by ID or name
- Use --uri flag for direct item retrieval using op://vault/item format
- Use --vault flag to limit searches to specific vault(s)

Native messaging mode:
- Automatically activated when JSON is piped via stdin
- Compatible with Chrome extension native messaging protocol
- Reads JSON with vaults, tags, and search criteria

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
 * C8YOP_LOG_LEVEL - Logging level (debug, info, warn, error; defaults to warn)`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if there's input available on stdin (automatic detection)
		stat, err := os.Stdin.Stat()
		if err == nil && (stat.Mode()&os.ModeCharDevice) == 0 {
			// stdin has data (pipe or redirect), switch to native messaging mode
			return runNativeMessaging()
		}

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

		outputFormat, err := cmd.Flags().GetString("output")
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

			return outputSession(session, reveal, outputFormat)
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
			return outputSession(session, reveal, outputFormat)
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
			return outputSession(session, reveal, outputFormat)
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
		level = slog.LevelWarn // Default to warning level
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
	rootCmd.PersistentFlags().StringP("output", "o", "json", "Output format (json, uri)")
	rootCmd.PersistentFlags().Bool("reveal", false, "Show sensitive information like passwords and TOTP secrets in output")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable colored output in picker")
	rootCmd.PersistentFlags().Bool("noColor", false, "Disable colored output in picker (go-c8y-cli compatibility)")

	// Hidden flags which are only there to satisfy the go-c8y-cli session interface
	rootCmd.PersistentFlags().String("loginType", "", "Not used (hidden)")
	rootCmd.PersistentFlags().Bool("clear", false, "Not used (hidden)")
	rootCmd.PersistentFlags().MarkHidden("loginType")
	rootCmd.PersistentFlags().MarkHidden("clear")

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

// Helper function to output session in the specified format
func outputSession(session *core.CumulocitySession, reveal bool, outputFormat string) error {
	switch outputFormat {
	case "json":
		return outputSessionAsJSON(session, reveal)
	case "uri":
		return outputSessionAsURI(session)
	default:
		return fmt.Errorf("unsupported output format: %s (supported: json, uri)", outputFormat)
	}
}

// Helper function to output session as JSON
func outputSessionAsJSON(session *core.CumulocitySession, reveal bool) error {
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

// Helper function to output session as op:// URI
func outputSessionAsURI(session *core.CumulocitySession) error {
	if session.VaultID == "" || session.ItemID == "" {
		return fmt.Errorf("missing vault name or item ID for URI output")
	}

	uri := fmt.Sprintf("op://%s/%s", session.VaultID, session.ItemID)

	// Add target_url parameter if session has a host URL
	if session.Host != "" {
		encodedURL := url.QueryEscape(session.Host)
		uri += fmt.Sprintf("?target_url=%s", encodedURL)
	}

	fmt.Printf("%s\n", uri)
	return nil
}

// Helper function to run the command in native messaging mode
func runNativeMessaging() error {
	slog.Debug("Starting native messaging mode")

	// Chrome Native Messaging protocol: persistent connection with message loop
	for {
		slog.Debug("Waiting for next message from Chrome extension")

		// Step 1: Read 4-byte length prefix
		lengthBytes := make([]byte, 4)
		n, err := io.ReadFull(os.Stdin, lengthBytes)
		if err != nil {
			if err == io.EOF {
				slog.Debug("Chrome extension closed connection")
				return nil // Normal exit when Chrome closes the pipe
			}
			return fmt.Errorf("failed to read message length: %w", err)
		}
		if n != 4 {
			return fmt.Errorf("incomplete length header: got %d bytes, expected 4", n)
		}

		// Step 2: Parse message length
		messageLength := binary.LittleEndian.Uint32(lengthBytes)
		slog.Debug("Received message length", "length", messageLength)

		// Sanity check on message length (prevent excessive memory allocation)
		if messageLength == 0 || messageLength > 1024*1024 { // Max 1MB
			return fmt.Errorf("invalid message length: %d", messageLength)
		}

		// Step 3: Read exactly messageLength bytes for the JSON message
		messageBytes := make([]byte, messageLength)
		n, err = io.ReadFull(os.Stdin, messageBytes)
		if err != nil {
			return fmt.Errorf("failed to read message data: %w", err)
		}
		if uint32(n) != messageLength {
			return fmt.Errorf("incomplete message: got %d bytes, expected %d", n, messageLength)
		}

		slog.Debug("Received complete message", "data", string(messageBytes))

		// Step 4: Parse the JSON message
		var req NativeMessagingRequest
		err = json.Unmarshal(messageBytes, &req)
		if err != nil {
			slog.Debug("Failed to parse JSON message", "error", err, "data", string(messageBytes))
			// Send error response and continue listening
			errorResponse := map[string]interface{}{
				"type":  "error",
				"error": fmt.Sprintf("Invalid JSON: %v", err),
			}
			if sendErr := sendNativeMessagingResponse(errorResponse); sendErr != nil {
				return fmt.Errorf("failed to send error response: %w", sendErr)
			}
			continue
		}

		slog.Debug("Parsed native messaging request", "request", req)

		// Step 5: Process the message and send response
		err = processNativeMessagingRequest(req)
		if err != nil {
			slog.Debug("Failed to process request", "error", err)
			// Send error response and continue listening
			errorResponse := map[string]interface{}{
				"type":  "error",
				"error": err.Error(),
			}
			if sendErr := sendNativeMessagingResponse(errorResponse); sendErr != nil {
				return fmt.Errorf("failed to send error response: %w", sendErr)
			}
			continue
		}

		// Successfully processed request, continue loop for next message
	}
}

// Helper function to process a single native messaging request
func processNativeMessagingRequest(req NativeMessagingRequest) error {
	// Handle special request types
	if req.Type == "test_auth" {
		slog.Debug("Handling test_auth request")
		return handleAuthTest(true) // Always use native messaging format in native messaging mode
	}

	// Extract vaults and tags, use search as filter
	var vaults []string
	var tags []string
	var filter string

	if len(req.Vaults) > 0 {
		vaults = req.Vaults
	} else {
		// No specific vaults requested, use default or all vaults
		defaultVault := getEnvWithFallback("C8YOP_VAULT", "CYOP_VAULT")
		if defaultVault != "" {
			vaults = []string{defaultVault}
		} else {
			vaults = nil // All vaults
		}
	}

	if len(req.Tags) > 0 {
		tags = req.Tags
	} else {
		// No specific tags, use default or "c8y" tag
		defaultTags := getEnvWithFallback("C8YOP_TAGS", "CYOP_TAGS")
		if defaultTags != "" {
			tags = splitAndTrimString(defaultTags)
		} else {
			tags = []string{"c8y"}
		}
	}

	filter = req.Search

	// Log the effective vaults, tags, and filter
	slog.Debug("Effective vaults, tags, and filter", "vaults", vaults, "tags", tags, "filter", filter)

	// Convert vaults slice to comma-separated string for NewClient
	var vaultString string
	if len(vaults) > 0 {
		vaultString = strings.Join(vaults, ",")
	}

	// Use the existing logic to process the request
	client := onepassword.NewClient(vaultString, tags...)
	sessions, err := client.List()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return fmt.Errorf("no sessions found matching vaults: %v and tags: %v", vaults, tags)
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
		return outputSessionNativeMessaging(session, req.Reveal, true) // Use reveal flag from request
	} else {
		// Multiple sessions found, return as JSON array
		var outputSessions []*core.CumulocitySession
		for _, session := range filteredSessions {
			// Populate session details and TOTP from the full session list
			populateSessionFromList(session, sessions)
			outputSessions = append(outputSessions, session)
		}
		return outputSessionsNativeMessaging(outputSessions, req.Reveal, true) // Use reveal flag from request
	}
}

// Helper function to send a response using native messaging format
func sendNativeMessagingResponse(response interface{}) error {
	jsonData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response to JSON: %w", err)
	}
	return writeNativeMessage(jsonData)
}

// Helper function to apply reveal flag to a session copy
func applyRevealFlag(session *core.CumulocitySession, reveal bool) *core.CumulocitySession {
	// Create a copy to avoid modifying the original
	outputSession := *session

	// Obfuscate sensitive information if reveal is false
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

	return &outputSession
}

// Helper function to output JSON data for native messaging
func outputJSONNativeMessaging(data interface{}, isNativeMessagingFormat bool) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	if isNativeMessagingFormat {
		return writeNativeMessage(jsonData)
	} else {
		// Plain JSON output
		fmt.Printf("%s\n", jsonData)
		return nil
	}
}

// Helper function to output single session for native messaging
func outputSessionNativeMessaging(session *core.CumulocitySession, reveal bool, isNativeMessagingFormat bool) error {
	outputSession := applyRevealFlag(session, reveal)
	return outputJSONNativeMessaging(outputSession, isNativeMessagingFormat)
}

// Helper function to output multiple sessions for native messaging
func outputSessionsNativeMessaging(sessions []*core.CumulocitySession, reveal bool, isNativeMessagingFormat bool) error {
	// Apply reveal flag to all sessions using the helper function
	var outputSessions []*core.CumulocitySession
	for _, session := range sessions {
		outputSession := applyRevealFlag(session, reveal)
		outputSessions = append(outputSessions, outputSession)
	}

	return outputJSONNativeMessaging(outputSessions, isNativeMessagingFormat)
}

// Helper function to write native messaging format
func writeNativeMessage(jsonData []byte) error {
	// Write 4-byte little-endian length prefix
	length := uint32(len(jsonData))
	lengthBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthBytes, length)

	if _, err := os.Stdout.Write(lengthBytes); err != nil {
		return fmt.Errorf("failed to write length prefix: %w", err)
	}

	// Write JSON data
	if _, err := os.Stdout.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write JSON data: %w", err)
	}

	return nil
}

// Helper function to handle authentication test
func handleAuthTest(isNativeMessagingFormat bool) error {
	slog.Debug("Running op signin for authentication test")

	// Run op signin command
	cmd := exec.Command("op", "signin")
	cmd.Stdin = os.Stdin   // Allow interactive signin
	cmd.Stdout = os.Stderr // Redirect output to stderr so it doesn't interfere with native messaging
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	// Prepare response
	var response map[string]interface{}
	if err != nil {
		slog.Debug("op signin failed", "error", err)
		response = map[string]interface{}{
			"type":    "auth_result",
			"success": false,
			"error":   err.Error(),
		}
	} else {
		slog.Debug("op signin succeeded")
		response = map[string]interface{}{
			"type":    "auth_result",
			"success": true,
		}
	}

	// Send response in appropriate format
	jsonData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal auth response: %w", err)
	}

	if isNativeMessagingFormat {
		return writeNativeMessage(jsonData)
	} else {
		fmt.Printf("%s\n", jsonData)
		return nil
	}
}
