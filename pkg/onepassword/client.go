package onepassword

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/cli/safeexec"
	"github.com/pquerna/otp/totp"
	session "github.com/thomaswinkler/c8y-session-1password/pkg/core"
)

type Client struct {
	Vault string
	Tags  []string
}

func NewClient(vault string, tags ...string) *Client {
	return &Client{
		Vault: vault,
		Tags:  tags,
	}
}

// parseVaultNames splits a comma-separated vault string and returns a slice of vault names
func (c *Client) parseVaultNames() []string {
	if c.Vault == "" {
		return []string{}
	}

	vaults := strings.Split(c.Vault, ",")
	for i := range vaults {
		vaults[i] = strings.TrimSpace(vaults[i])
	}

	// Remove empty entries
	filtered := make([]string, 0, len(vaults))
	for _, vault := range vaults {
		if vault != "" {
			filtered = append(filtered, vault)
		}
	}

	return filtered
}

// OPItem 1Password item containing the login information
type OPItem struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Category string    `json:"category"`
	Vault    OPVault   `json:"vault"`
	Fields   []OPField `json:"fields"`
	URLs     []OPURL   `json:"urls"`
	Tags     []string  `json:"tags"`
}

func (opi *OPItem) HasTenantField() bool {
	for _, field := range opi.Fields {
		label := strings.ToLower(field.Label)
		if strings.Contains(label, "tenant") && strings.TrimSpace(field.Value) != "" {
			return true
		}
	}
	return false
}

func (opi *OPItem) Skip() bool {
	return len(opi.URLs) == 0 || opi.Category != "LOGIN"
}

func (opi *OPItem) GetUsername() string {
	for _, field := range opi.Fields {
		if field.ID == "username" {
			return field.Value
		}
	}
	return ""
}

func (opi *OPItem) GetPassword() string {
	for _, field := range opi.Fields {
		if field.ID == "password" {
			return field.Value
		}
	}
	return ""
}

func (opi *OPItem) GetTOTPSecret() string {
	for _, field := range opi.Fields {
		if field.Type == "OTP" {
			return field.TOTPDetails.Secret
		}
	}
	return ""
}

// OPField 1Password custom fields
type OPField struct {
	ID          string        `json:"id"`
	Type        string        `json:"type"`
	Purpose     string        `json:"purpose"`
	Label       string        `json:"label"`
	Value       string        `json:"value"`
	TOTPDetails OPTOTPDetails `json:"totp,omitempty"`
}

type OPTOTPDetails struct {
	Secret string `json:"secret"`
}

// OPVault 1Password vault information
type OPVault struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OPURL 1Password URL associated with the login credentials
type OPURL struct {
	Label   string `json:"label"`
	Primary bool   `json:"primary"`
	Href    string `json:"href"`
}

func check1Password() error {
	if _, err := safeexec.LookPath("op"); err != nil {
		return fmt.Errorf("could not find 'op' (1Password CLI). Check if it is installed on your machine")
	}

	// Check if user is signed in
	cmd := exec.Command("op", "account", "get")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not signed in to 1Password. Please run 'op signin' first")
	}

	return nil
}

func mapToSession(item *OPItem, vaults map[string]string) *session.CumulocitySession {
	// Determine vault name for URI
	vaultName := item.Vault.Name
	if name, found := vaults[item.Vault.ID]; found {
		vaultName = name
	}

	session := &session.CumulocitySession{
		SessionURI: fmt.Sprintf("op://%s/%s", vaultName, item.Title),
		Name:       item.Title,
		ItemID:     item.ID,
		ItemName:   item.Title,
		Username:   item.GetUsername(),
		Password:   item.GetPassword(),
		VaultID:    item.Vault.ID,
		VaultName:  vaultName,
		TOTPSecret: item.GetTOTPSecret(),
		Tags:       item.Tags,
	}

	if len(item.URLs) > 0 {
		session.Host = item.URLs[0].Href
	}

	if len(item.Fields) > 0 {
		for _, field := range item.Fields {
			if strings.HasPrefix(strings.ToLower(field.Label), "tenant") {
				session.Tenant = field.Value
				break
			}
		}
	}

	if strings.Contains(item.GetUsername(), "/") {
		parts := strings.SplitN(item.GetUsername(), "/", 2)
		if len(parts) == 2 {
			if session.Tenant == "" {
				session.Tenant = parts[0]
			}
			session.Username = parts[1]
		}
	}
	return session
}

func isUID(v string) bool {
	// 1Password item IDs are different format than UUIDs
	r := regexp.MustCompile("^[a-zA-Z0-9]{26}$")
	return r.MatchString(v)
}

type Vault struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *Client) ListVaults(name ...string) (map[string]string, error) {
	vaults := make([]Vault, 0)

	args := []string{
		"vault", "list",
		"--format", "json",
	}

	err := c.exec(args, &vaults)

	vaultMap := make(map[string]string)
	for _, vault := range vaults {
		if len(name) == 0 || strings.Contains(strings.ToLower(vault.Name), strings.ToLower(name[0])) {
			vaultMap[vault.ID] = vault.Name
		}
	}

	return vaultMap, err
}

func (c *Client) exec(args []string, data any) error {
	if err := check1Password(); err != nil {
		return err
	}

	op := exec.Command("op", args...)
	stdout, err := op.StdoutPipe()
	if err != nil {
		return err
	}

	err = op.Start()
	if err != nil {
		return err
	}

	parseErr := json.NewDecoder(stdout).Decode(data)

	// wait for command to finish in background
	go op.Wait()

	return parseErr
}

func (c *Client) List(name ...string) ([]*session.CumulocitySession, error) {
	if err := check1Password(); err != nil {
		return nil, err
	}

	vaultNames := c.parseVaultNames()
	allSessions := make([]*session.CumulocitySession, 0)

	// If no vaults specified, search all vaults
	if len(vaultNames) == 0 {
		sessions, err := c.listFromVault("")
		if err != nil {
			return nil, err
		}
		allSessions = append(allSessions, sessions...)
	} else {
		// Search each vault in order
		for _, vaultName := range vaultNames {
			sessions, err := c.listFromVault(vaultName)
			if err != nil {
				slog.Warn("Failed to search vault", "vault", vaultName, "error", err)
				continue
			}
			allSessions = append(allSessions, sessions...)
		}
	}

	return allSessions, nil
}

// listFromVault searches for sessions in a specific vault (or all vaults if empty)
func (c *Client) listFromVault(vaultName string) ([]*session.CumulocitySession, error) {
	cmdArgs := []string{
		"item", "list",
		"--format", "json",
		"--categories", "Login",
	}

	var vaults map[string]string
	var vaultErr error

	if vaultName != "" {
		if isUID(vaultName) {
			// Filter by vault id (no additional lookup required)
			cmdArgs = append(cmdArgs, "--vault", vaultName)
		} else {
			// Filter by vault name/pattern (additional lookup required)
			vaults, vaultErr = c.ListVaults(vaultName)
			if vaultErr != nil {
				return nil, vaultErr
			}
			if len(vaults) > 0 {
				// Use the first matching vault
				for vaultID := range vaults {
					cmdArgs = append(cmdArgs, "--vault", vaultID)
					break
				}
			}
		}
	}

	// Add tags filter if specified
	if len(c.Tags) > 0 {
		for _, tag := range c.Tags {
			cmdArgs = append(cmdArgs, "--tags", tag)
		}
	}

	slog.Debug("Starting", "time", time.Now().Format(time.RFC3339Nano))

	items := make([]OPItem, 0)
	err := c.exec(cmdArgs, &items)
	if err != nil {
		return nil, err
	}

	// Get detailed item information including fields
	detailedItems := make([]OPItem, 0)
	for _, item := range items {
		var detailedItem OPItem
		detailArgs := []string{
			"item", "get", item.ID,
			"--format", "json",
		}
		if err := c.exec(detailArgs, &detailedItem); err != nil {
			slog.Warn("Failed to get item details", "id", item.ID, "error", err)
			continue
		}
		detailedItems = append(detailedItems, detailedItem)
	}

	// Get vault names for proper display if not already loaded
	if vaults == nil {
		vaults, err = c.ListVaults()
		if err != nil {
			slog.Warn("Failed to list vaults", "error", err)
			vaults = make(map[string]string)
		}
	}

	sessions := make([]*session.CumulocitySession, 0)
	for _, item := range detailedItems {
		if item.Skip() {
			continue
		}

		// Filter by tags if specified and not already filtered by op command
		if len(c.Tags) > 0 {
			hasRequiredTag := false
			for _, requiredTag := range c.Tags {
				for _, itemTag := range item.Tags {
					if strings.EqualFold(itemTag, requiredTag) {
						hasRequiredTag = true
						break
					}
				}
				if hasRequiredTag {
					break
				}
			}
			if !hasRequiredTag {
				continue
			}
		}

		sessions = append(sessions, mapToSession(&item, vaults))
	}

	return sessions, nil
}

func GetTOTPCode(secret string, t time.Time) (string, error) {
	if t.Year() == 0 {
		t = time.Now()
	}
	return totp.GenerateCode(secret, t)
}

func GetTOTPCodeFromSecret(secret string) (string, error) {
	now := time.Now()
	totpTime := now
	totpPeriod := 30
	totpNextTransition := totpPeriod - now.Second()%30
	if totpNextTransition < 5 {
		totpTime = now.Add(30 * time.Second)
	}
	return GetTOTPCode(secret, totpTime)
}

// ParseOPURI parses an op://vault/item URI and returns vault and item components
func ParseOPURI(uri string) (vault, item string, err error) {
	if !strings.HasPrefix(uri, "op://") {
		return "", "", fmt.Errorf("invalid op:// URI format: %s", uri)
	}

	path := strings.TrimPrefix(uri, "op://")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid op:// URI format: expected op://vault/item, got %s", uri)
	}

	vault = parts[0]
	item = parts[1]

	if vault == "" {
		return "", "", fmt.Errorf("invalid op:// URI format: vault cannot be empty, got %s", uri)
	}

	if item == "" {
		return "", "", fmt.Errorf("invalid op:// URI format: item cannot be empty, got %s", uri)
	}

	return vault, item, nil
}

// GetItem retrieves a specific item from 1Password by vault and item identifier
func (c *Client) GetItem(vaultIdentifier, itemIdentifier string) (*session.CumulocitySession, error) {
	if err := check1Password(); err != nil {
		return nil, err
	}

	// Parse vault names if comma-separated
	vaultNames := []string{}
	if vaultIdentifier != "" {
		vaults := strings.Split(vaultIdentifier, ",")
		for _, vault := range vaults {
			vault = strings.TrimSpace(vault)
			if vault != "" {
				vaultNames = append(vaultNames, vault)
			}
		}
	}

	// If no vaults specified, try without vault filter
	if len(vaultNames) == 0 {
		return c.getItemFromVault("", itemIdentifier)
	}

	// Try each vault in order until we find the item
	var lastErr error
	for _, vaultName := range vaultNames {
		session, err := c.getItemFromVault(vaultName, itemIdentifier)
		if err == nil {
			return session, nil
		}
		lastErr = err
		slog.Debug("Item not found in vault", "vault", vaultName, "item", itemIdentifier, "error", err)
	}

	// If we get here, the item wasn't found in any vault
	return nil, fmt.Errorf("item '%s' not found in any of the specified vaults [%s]: %w",
		itemIdentifier, strings.Join(vaultNames, ", "), lastErr)
}

// getItemFromVault retrieves an item from a specific vault (or any vault if empty)
func (c *Client) getItemFromVault(vaultIdentifier, itemIdentifier string) (*session.CumulocitySession, error) {
	// Build the op item get command
	args := []string{"item", "get", itemIdentifier, "--format", "json"}
	if vaultIdentifier != "" {
		args = append(args, "--vault", vaultIdentifier)
	}

	cmd := exec.Command("op", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get item from 1Password: %w", err)
	}

	var item OPItem
	if err := json.Unmarshal(output, &item); err != nil {
		return nil, fmt.Errorf("failed to parse 1Password item: %w", err)
	}

	// Get vault information for proper naming
	vaults, err := c.ListVaults()
	if err != nil {
		slog.Warn("Failed to list vaults", "error", err)
		vaults = make(map[string]string)
	}

	session := mapToSession(&item, vaults)
	return session, nil
}
