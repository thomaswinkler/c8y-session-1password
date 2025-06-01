package cmd

import (
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
			vault = getEnvWithFallback("C8YOP_VAULT", "CYOP_VAULT")
		}

		// Get tags using helper function
		tags := parseTags(tagsFlag)

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

		// Populate session details and TOTP from the full session list
		populateSessionFromList(session, sessions)

		return outputSession(session)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().String("vault", "", "Vault name or ID (defaults to C8YOP_VAULT or CYOP_VAULT env var)")
	listCmd.Flags().String("tags", "", "Comma-separated tags to filter by (defaults to C8YOP_TAGS or CYOP_TAGS env var, then 'c8y')")
}
