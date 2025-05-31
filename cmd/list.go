package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thomaswinkler/c8y-session-1password/pkg/core/picker"
	"github.com/thomaswinkler/c8y-session-1password/pkg/onepassword"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Interactive session picker",
	Long: `Interactive session picker for Cumulocity IoT sessions from your 1Password vault

This command provides an interactive picker to browse and select from available sessions.
For direct item retrieval, use the root command with --vault/--item or --uri flags.

Examples:
  # Interactive selection with all sessions
  c8y-session-1password list
  
  # Filter by specific vault
  c8y-session-1password list --vault "Employee"
  
  # Filter by tags
  c8y-session-1password list --tags "c8y,production"`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		vault, err := cmd.Flags().GetString("vault")
		if err != nil {
			return err
		}

		tagsFlag, err := cmd.Flags().GetString("tags")
		if err != nil {
			return err
		}

		// Get default values from environment variables
		if vault == "" {
			vault = os.Getenv("C8YOP_VAULT")
			if vault == "" {
				vault = os.Getenv("CYOP_VAULT") // Fallback for compatibility
			}
		}

		var tags []string
		if tagsFlag != "" {
			tags = strings.Split(tagsFlag, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
		} else if envTags := os.Getenv("C8YOP_TAGS"); envTags != "" {
			tags = strings.Split(envTags, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
		} else if envTags := os.Getenv("CYOP_TAGS"); envTags != "" { // Fallback for compatibility
			tags = strings.Split(envTags, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
		}

		// Default to "c8y" tag if no tags specified - this ensures only
		// Cumulocity-related items are shown in the interactive picker
		if len(tags) == 0 {
			tags = []string{"c8y"}
		}

		client := onepassword.NewClient(vault, tags...)
		sessions, err := client.List()
		if err != nil {
			return err
		}

		// Always use interactive picker for list command
		session, err := picker.Pick(sessions)
		if err != nil {
			return err
		}

		// Check if TOTP secret is present and calc next code
		for _, s := range sessions {
			if session.ItemID == s.ItemID {
				session.Password = s.Password
				if s.TOTPSecret != "" {
					totp, totpErr := onepassword.GetTOTPCodeFromSecret(s.TOTPSecret)
					if totpErr == nil {
						session.TOTP = totp
					}
					break
				}
			}
		}

		out, err := json.MarshalIndent(session, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().String("vault", "", "Vault name or ID (defaults to C8YOP_VAULT or CYOP_VAULT env var)")
	listCmd.Flags().String("tags", "", "Comma-separated tags to filter by (defaults to C8YOP_TAGS or CYOP_TAGS env var, then 'c8y')")
}
