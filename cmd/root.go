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
			vault = os.Getenv("C8YOP_VAULT")
			if vault == "" {
				vault = os.Getenv("CYOP_VAULT") // Fallback for compatibility
			}
		}

		if item == "" {
			item = os.Getenv("C8YOP_ITEM")
			if item == "" {
				item = os.Getenv("CYOP_ITEM") // Fallback for compatibility
			}
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
			if session.TOTPSecret != "" {
				totp, totpErr := onepassword.GetTOTPCodeFromSecret(session.TOTPSecret)
				if totpErr == nil {
					session.TOTP = totp
				}
			}

			out, err := json.MarshalIndent(session, "", "  ")
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", out)
			return nil
		}

		// If no specific item is requested, fall back to interactive list
		// Get default tags
		var tags []string
		if envTags := os.Getenv("C8YOP_TAGS"); envTags != "" {
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

		// Use interactive picker
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
