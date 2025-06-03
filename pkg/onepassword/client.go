package onepassword

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cli/safeexec"
	"github.com/pquerna/otp/totp"
	"github.com/thomaswinkler/c8y-session-1password/pkg/core"
)

type Client struct {
	Vault   string
	Tags    []string
	checked bool // Track if 1Password has been checked for this session
}

type Vault struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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

func NewClient(vault string, tags ...string) *Client {
	return &Client{
		Vault: vault,
		Tags:  tags,
	}
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

// URLSource represents a URL from any source (URLs array or fields)
type URLSource struct {
	URL     string
	Label   string
	Primary bool
	Source  string // "urls" or "field"
}

// extractItemFields extracts common fields from a 1Password item
type itemFields struct {
	username   string
	password   string
	totpSecret string
	tenant     string
}

// parseVaultNamesFromString splits a comma-separated vault string and returns a slice of vault names
func parseVaultNamesFromString(vaultStr string) []string {
	if vaultStr == "" {
		return []string{}
	}

	vaults := strings.Split(vaultStr, ",")
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

// parseVaultNames splits a comma-separated vault string and returns a slice of vault names
func (c *Client) parseVaultNames() []string {
	return parseVaultNamesFromString(c.Vault)
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
	if opi.Category != "LOGIN" {
		slog.Debug("Item skipped: not LOGIN category", "item_id", opi.ID, "category", opi.Category)
		return true
	}

	// Don't skip if URLs array has entries
	if len(opi.URLs) > 0 {
		slog.Debug("Item accepted: has URLs array", "item_id", opi.ID, "urls_count", len(opi.URLs))
		return false
	}

	// Check for URL fields if no urls array
	urlFieldCount := 0
	for _, field := range opi.Fields {
		if isURLField(field) {
			urlFieldCount++
		}
	}

	if urlFieldCount > 0 {
		slog.Debug("Item accepted: has URL fields", "item_id", opi.ID, "url_fields_count", urlFieldCount)
		return false
	}

	// Skip if no URLs found anywhere
	slog.Debug("Item skipped: no URLs found", "item_id", opi.ID, "urls_count", len(opi.URLs), "url_fields_count", urlFieldCount)
	return true
}

func (opi *OPItem) GetUsername() string {
	fields := opi.extractFields()
	return fields.username
}

func (opi *OPItem) GetPassword() string {
	fields := opi.extractFields()
	return fields.password
}

func (opi *OPItem) GetTOTPSecret() string {
	fields := opi.extractFields()
	return fields.totpSecret
}

func check1Password() error {
	if _, err := safeexec.LookPath("op"); err != nil {
		return fmt.Errorf("could not find 'op' (1Password CLI). Check if it is installed on your machine")
	}

	// Check if user is signed in
	start := time.Now()
	slog.Debug("op command", "command", "op account get")
	cmd := exec.Command("op", "account", "get")
	err := cmd.Run()
	duration := time.Since(start)
	slog.Debug("op command completed", "duration_ms", duration.Milliseconds())
	if err != nil {
		return fmt.Errorf("not signed in to 1Password. Please run 'op signin' first")
	}

	return nil
}

// ensureChecked calls check1Password only once per client session
func (c *Client) ensureChecked() error {
	if !c.checked {
		if err := check1Password(); err != nil {
			return err
		}
		c.checked = true
	}
	return nil
}

func (opi *OPItem) extractFields() itemFields {
	fields := itemFields{}

	for _, field := range opi.Fields {
		switch field.ID {
		case "username":
			fields.username = field.Value
		case "password":
			fields.password = field.Value
		}

		if field.Type == "OTP" {
			fields.totpSecret = field.TOTPDetails.Secret
		}

		// Extract tenant from custom field
		if strings.HasPrefix(strings.ToLower(field.Label), "tenant") && fields.tenant == "" {
			fields.tenant = field.Value
		}
	}

	// Handle tenant/username combination (format: tenant/username)
	if strings.Contains(fields.username, "/") {
		parts := strings.SplitN(fields.username, "/", 2)
		if len(parts) == 2 {
			if fields.tenant == "" {
				fields.tenant = parts[0]
			}
			fields.username = parts[1]
		}
	}

	return fields
}

// isURLField checks if a field contains a URL
func isURLField(field OPField) bool {
	if strings.TrimSpace(field.Value) == "" {
		return false
	}
	fieldLabel := strings.ToLower(field.Label)
	fieldType := strings.ToUpper(field.Type)
	return (fieldLabel == "website" || fieldLabel == "url") || fieldType == "URL"
}

// collectURLs gathers all URLs from both URLs array and URL fields
func (opi *OPItem) collectURLs() []URLSource {
	allURLs := make([]URLSource, 0, len(opi.URLs)+len(opi.Fields))

	// Add URLs from the urls array
	for _, url := range opi.URLs {
		allURLs = append(allURLs, URLSource{
			URL:     url.Href,
			Label:   url.Label,
			Primary: url.Primary,
			Source:  "urls",
		})
	}

	// Add URLs from fields (Type="URL" or Label="website"/"url")
	for _, field := range opi.Fields {
		if isURLField(field) {
			allURLs = append(allURLs, URLSource{
				URL:     field.Value,
				Label:   field.Label,
				Primary: false,
				Source:  "field",
			})
		}
	}

	// Sort URLs to prioritize primary URLs first
	sort.Slice(allURLs, func(i, j int) bool {
		return allURLs[i].Primary && !allURLs[j].Primary
	})

	return allURLs
}

// mapToSessions creates one or more sessions from a 1Password item, handling multiple URLs
func (c *Client) mapToSessions(item *OPItem, vaults map[string]string) []*core.CumulocitySession {
	// Determine vault name for URI
	vaultName := item.Vault.Name
	if name, found := vaults[item.Vault.ID]; found {
		vaultName = name
	}

	// Convert to core types
	coreItem := core.Item{
		ID:    item.ID,
		Title: item.Title,
		Tags:  item.Tags,
		Vault: core.Vault{
			ID:   item.Vault.ID,
			Name: item.Vault.Name,
		},
	}

	// Extract fields
	fields := item.extractFields()
	coreFields := core.ItemFields{
		Username:   fields.username,
		Password:   fields.password,
		TOTPSecret: fields.totpSecret,
		Tenant:     fields.tenant,
	}

	// Collect URLs
	allURLs := item.collectURLs()
	coreURLs := make([]core.URLSource, len(allURLs))
	for i, url := range allURLs {
		coreURLs[i] = core.URLSource{
			URL:     url.URL,
			Label:   url.Label,
			Primary: url.Primary,
			Source:  url.Source,
		}
	}

	// Use unified session mapping with tag filtering
	return core.MapToSessions(coreItem, coreFields, coreURLs, vaultName, c.Tags)
}

func isUID(v string) bool {
	// 1Password item IDs are different format than UUIDs
	r := regexp.MustCompile("^[a-zA-Z0-9]{26}$")
	return r.MatchString(v)
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
	if err := c.ensureChecked(); err != nil {
		return err
	}

	start := time.Now()
	slog.Debug("op command", "command", "op "+strings.Join(args, " "))
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
	go func() {
		_ = op.Wait() // ignore error as we already have the data
		duration := time.Since(start)
		slog.Debug("op command completed", "duration_ms", duration.Milliseconds())
	}()

	return parseErr
}

func (c *Client) List(name ...string) ([]*core.CumulocitySession, error) {
	if err := c.ensureChecked(); err != nil {
		return nil, err
	}

	vaultNames := c.parseVaultNames()
	slog.Debug("Parsed vault names", "vaultNames", vaultNames, "count", len(vaultNames))
	allSessions := make([]*core.CumulocitySession, 0)

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
				// For single vault, return error immediately
				// For multiple vaults, continue with others but log the error
				if len(vaultNames) == 1 {
					return nil, err
				}
				slog.Warn("Failed to search vault", "vault", vaultName, "error", err)
				continue
			}
			allSessions = append(allSessions, sessions...)
		}
	}

	// Sort sessions by Host URL for better organization
	sort.Slice(allSessions, func(i, j int) bool {
		// Normalize URLs for better sorting (remove protocol and trailing slash)
		normalizedI := core.NormalizeDisplayURL(allSessions[i].Host)
		normalizedJ := core.NormalizeDisplayURL(allSessions[j].Host)
		return normalizedI < normalizedJ
	})

	slog.Debug("List method completed", "total_sessions", len(allSessions), "vaults_searched", len(vaultNames))
	return allSessions, nil
}

// listFromVault searches for sessions in a specific vault (or all vaults if empty)
func (c *Client) listFromVault(vaultName string) ([]*core.CumulocitySession, error) {
	listArgs := []string{
		"item", "list",
		"--format", "json",
		"--categories", "Login",
	}

	var vaults map[string]string
	var vaultErr error

	if vaultName != "" {
		slog.Debug("Checking vault", "vault", vaultName, "isUID", isUID(vaultName))
		if isUID(vaultName) {
			// Filter by vault id (no additional lookup required)
			listArgs = append(listArgs, "--vault", vaultName)
		} else {
			// Filter by vault name/pattern (additional lookup required)
			vaults, vaultErr = c.ListVaults(vaultName)
			if vaultErr != nil {
				return nil, vaultErr
			}
			slog.Debug("Vault lookup result", "vault", vaultName, "found_count", len(vaults))
			if len(vaults) > 0 {
				// Use the first matching vault
				for vaultID := range vaults {
					slog.Debug("Using vault", "vaultID", vaultID, "vaultName", vaults[vaultID])
					listArgs = append(listArgs, "--vault", vaultID)
					break
				}
			} else {
				// Vault specified but not found - return error instead of silently searching all vaults
				slog.Debug("Vault not found, returning error", "vault", vaultName)
				return nil, fmt.Errorf("Vault '%s' not found", vaultName)
			}
		}
	}

	// Add tags filter if specified
	if len(c.Tags) > 0 {
		for _, tag := range c.Tags {
			listArgs = append(listArgs, "--tags", tag)
		}
	}

	slog.Debug("Starting optimized fetch")

	// First get the list of items
	items := make([]OPItem, 0)
	err := c.exec(listArgs, &items)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return []*core.CumulocitySession{}, nil
	}

	var detailedItems []OPItem

	// Use bulk fetch for multiple items, individual fetch for single item
	if len(items) > 1 {
		slog.Debug("Using bulk fetch for multiple items", "count", len(items))
		detailedItems, err = c.bulkGetItems(listArgs)
		if err != nil {
			slog.Warn("Bulk fetch failed, falling back to individual fetches", "error", err)
			detailedItems, err = c.individualGetItems(items)
		}
	} else {
		slog.Debug("Using individual fetch for single item")
		detailedItems, err = c.individualGetItems(items)
	}

	if err != nil {
		return nil, err
	}

	slog.Debug("Completed fetch", "count", len(detailedItems))

	// Get vault names for proper display if not already loaded
	if vaults == nil {
		vaults, err = c.ListVaults()
		if err != nil {
			slog.Warn("Failed to list vaults", "error", err)
			vaults = make(map[string]string)
		}
	}

	sessions := make([]*core.CumulocitySession, 0)
	for _, item := range detailedItems {
		slog.Debug("Processing item", "item_id", item.ID, "item_title", item.Title, "category", item.Category, "urls_count", len(item.URLs), "tags", item.Tags)

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
				slog.Debug("Skipping item", "item_id", item.ID, "reason", "missing required tags", "required_tags", c.Tags, "item_tags", item.Tags)
				continue
			}
		}

		// Create sessions for this item (may create multiple sessions for multiple URLs)
		itemSessions := c.mapToSessions(&item, vaults)
		sessions = append(sessions, itemSessions...)
	}

	slog.Debug("Item filtering completed", "total_items", len(detailedItems), "sessions_created", len(sessions), "vault_name", vaultName)
	return sessions, nil
}

// bulkGetItems efficiently fetches detailed item information using piped commands
// This eliminates N+1 queries by using: op item list ... | op item get -
func (c *Client) bulkGetItems(listArgs []string) ([]OPItem, error) {
	if err := c.ensureChecked(); err != nil {
		return nil, err
	}

	start := time.Now()
	slog.Debug("op command", "command", "op "+strings.Join(listArgs, " ")+" | op item get - --format json")

	// Create the list command
	listCmd := exec.Command("op", listArgs...)

	// Create the get command that reads from list output
	getCmd := exec.Command("op", "item", "get", "-", "--format", "json")

	// Connect the commands via pipe
	pipe, err := listCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}

	getCmd.Stdin = pipe

	// Start the list command
	if err := listCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start list command: %w", err)
	}

	// Get the output from the get command
	output, err := getCmd.Output()
	if err != nil {
		// Make sure to wait for list command to finish
		_ = listCmd.Wait()
		return nil, fmt.Errorf("failed to get detailed items: %w", err)
	}

	// Wait for list command to finish
	if err := listCmd.Wait(); err != nil {
		return nil, fmt.Errorf("list command failed: %w", err)
	}

	duration := time.Since(start)
	slog.Debug("op command completed", "duration_ms", duration.Milliseconds())

	// Parse multiple JSON objects from the output
	// The output contains multiple pretty-printed JSON objects
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return []OPItem{}, nil
	}

	// Use a JSON decoder to parse multiple JSON objects
	items := make([]OPItem, 0)
	decoder := json.NewDecoder(strings.NewReader(outputStr))

	for {
		var item OPItem
		if err := decoder.Decode(&item); err != nil {
			if err == io.EOF {
				break // End of input
			}
			// Skip invalid JSON and continue
			slog.Warn("Failed to parse JSON object", "error", err)
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// individualGetItems fetches detailed information for items one by one (fallback method)
func (c *Client) individualGetItems(items []OPItem) ([]OPItem, error) {
	detailedItems := make([]OPItem, 0, len(items))
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
	return detailedItems, nil
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
func (c *Client) GetItem(vaultIdentifier, itemIdentifier string) (*core.CumulocitySession, error) {
	if err := c.ensureChecked(); err != nil {
		return nil, err
	}

	// Parse vault names if comma-separated using the helper function
	vaultNames := parseVaultNamesFromString(vaultIdentifier)

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
func (c *Client) getItemFromVault(vaultIdentifier, itemIdentifier string) (*core.CumulocitySession, error) {
	// Build the op item get command
	args := []string{"item", "get", itemIdentifier, "--format", "json"}
	if vaultIdentifier != "" {
		args = append(args, "--vault", vaultIdentifier)
	}

	start := time.Now()
	slog.Debug("op command", "command", "op "+strings.Join(args, " "))
	cmd := exec.Command("op", args...)
	output, err := cmd.Output()
	duration := time.Since(start)
	slog.Debug("op command completed", "duration_ms", duration.Milliseconds())
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

	// Use mapToSessions to get properly formatted sessions
	sessions := c.mapToSessions(&item, vaults)
	slog.Debug("Created sessions for single item", "item_id", item.ID, "item_title", item.Title, "session_count", len(sessions))
	if len(sessions) > 0 {
		slog.Debug("Returning first session", "session_host", sessions[0].Host, "session_item_id", sessions[0].ItemID)
		return sessions[0], nil
	}
	return nil, fmt.Errorf("no valid session found for item")
}
