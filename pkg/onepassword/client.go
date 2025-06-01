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
	Vault string
	Tags  []string
}

func NewClient(vault string, tags ...string) *Client {
	return &Client{
		Vault: vault,
		Tags:  tags,
	}
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
	if opi.Category != "LOGIN" {
		return true
	}

	// Don't skip if URLs array has entries
	if len(opi.URLs) > 0 {
		return false
	}

	// Check for URL fields if no urls array
	for _, field := range opi.Fields {
		if isURLField(field) {
			return false
		}
	}

	// Skip if no URLs found anywhere
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

// extractItemFields extracts common fields from a 1Password item
type itemFields struct {
	username   string
	password   string
	totpSecret string
	tenant     string
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

// buildSessionName creates an appropriate name for a session based on URL source and count
func buildSessionName(item *OPItem, urlSource URLSource, urlIndex int, totalURLs int, labelCounts map[string]int) string {
	if totalURLs == 1 {
		return item.Title
	}

	// If current URL's label appears multiple times, use hostname for distinction
	if labelCounts[urlSource.Label] > 1 {
		hostname := extractHostname(urlSource.URL)
		if urlSource.Primary {
			return fmt.Sprintf("%s (%s - Primary)", item.Title, hostname)
		}
		return fmt.Sprintf("%s (%s)", item.Title, hostname)
	}

	if urlSource.Label != "" && urlSource.Label != "website" {
		return fmt.Sprintf("%s (%s)", item.Title, urlSource.Label)
	}

	if urlSource.Primary {
		return fmt.Sprintf("%s (Primary)", item.Title)
	}

	return fmt.Sprintf("%s (URL %d)", item.Title, urlIndex+1)
}

// buildSessionURI creates an appropriate URI for a session
func buildSessionURI(vaultName, itemTitle string, urlSource URLSource, urlIndex int, totalURLs int, labelCounts map[string]int) string {
	baseURI := fmt.Sprintf("op://%s/%s", vaultName, itemTitle)
	if totalURLs == 1 {
		return baseURI
	}

	var fragment string
	if labelCounts[urlSource.Label] > 1 {
		hostname := extractHostname(urlSource.URL)
		if urlSource.Primary {
			fragment = hostname + "-primary"
		} else {
			fragment = hostname
		}
	} else if urlSource.Label != "" && urlSource.Label != "website" {
		fragment = urlSource.Label
	} else if urlSource.Primary {
		fragment = "primary"
	} else {
		fragment = fmt.Sprintf("url%d", urlIndex+1)
	}

	return fmt.Sprintf("%s#%s", baseURI, fragment)
}

// createSession builds a CumulocitySession from extracted data
func createSession(item *OPItem, fields itemFields, vaultName string, urlSource URLSource, sessionName, sessionURI string) *core.CumulocitySession {
	return &core.CumulocitySession{
		SessionURI: sessionURI,
		Name:       sessionName,
		ItemID:     item.ID,
		ItemName:   item.Title,
		Username:   fields.username,
		Password:   fields.password,
		Tenant:     fields.tenant,
		Host:       urlSource.URL,
		VaultID:    item.Vault.ID,
		VaultName:  vaultName,
		TOTPSecret: fields.totpSecret,
		Tags:       item.Tags,
	}
}

func mapToSession(item *OPItem, vaults map[string]string) *core.CumulocitySession {
	// Use mapToSessions and return the first session for backward compatibility
	sessions := mapToSessions(item, vaults)
	if len(sessions) > 0 {
		return sessions[0]
	}
	return nil
}

// mapToSessions creates one or more sessions from a 1Password item, handling multiple URLs
func mapToSessions(item *OPItem, vaults map[string]string) []*core.CumulocitySession {
	// Determine vault name for URI
	vaultName := item.Vault.Name
	if name, found := vaults[item.Vault.ID]; found {
		vaultName = name
	}

	// Extract all fields at once
	fields := item.extractFields()

	// Collect all URLs from both sources
	allURLs := item.collectURLs()

	// If no URLs found anywhere, create one session without URL
	if len(allURLs) == 0 {
		emptyURL := URLSource{URL: "", Label: "", Primary: false, Source: "none"}
		sessionName := buildSessionName(item, emptyURL, 0, 1, nil)
		sessionURI := buildSessionURI(vaultName, item.Title, emptyURL, 0, 1, nil)
		session := createSession(item, fields, vaultName, emptyURL, sessionName, sessionURI)
		result := make([]*core.CumulocitySession, 1)
		result[0] = session
		return result
	}

	// Pre-calculate label counts for naming decisions
	labelCounts := make(map[string]int)
	for _, url := range allURLs {
		labelCounts[url.Label]++
	}

	// Create sessions for all URLs
	sessions := make([]*core.CumulocitySession, 0, len(allURLs))
	for i, urlSource := range allURLs {
		sessionName := buildSessionName(item, urlSource, i, len(allURLs), labelCounts)
		sessionURI := buildSessionURI(vaultName, item.Title, urlSource, i, len(allURLs), labelCounts)
		session := createSession(item, fields, vaultName, urlSource, sessionName, sessionURI)
		sessions = append(sessions, session)
	}

	return sessions
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
	go func() {
		_ = op.Wait() // ignore error as we already have the data
	}()

	return parseErr
}

func (c *Client) List(name ...string) ([]*core.CumulocitySession, error) {
	if err := check1Password(); err != nil {
		return nil, err
	}

	vaultNames := c.parseVaultNames()
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
				slog.Warn("Failed to search vault", "vault", vaultName, "error", err)
				continue
			}
			allSessions = append(allSessions, sessions...)
		}
	}

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
		if isUID(vaultName) {
			// Filter by vault id (no additional lookup required)
			listArgs = append(listArgs, "--vault", vaultName)
		} else {
			// Filter by vault name/pattern (additional lookup required)
			vaults, vaultErr = c.ListVaults(vaultName)
			if vaultErr != nil {
				return nil, vaultErr
			}
			if len(vaults) > 0 {
				// Use the first matching vault
				for vaultID := range vaults {
					listArgs = append(listArgs, "--vault", vaultID)
					break
				}
			}
		}
	}

	// Add tags filter if specified
	if len(c.Tags) > 0 {
		for _, tag := range c.Tags {
			listArgs = append(listArgs, "--tags", tag)
		}
	}

	slog.Debug("Starting optimized fetch", "time", time.Now().Format(time.RFC3339Nano))

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

	slog.Debug("Completed fetch", "count", len(detailedItems), "time", time.Now().Format(time.RFC3339Nano))

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

		// Create sessions for this item (may create multiple sessions for multiple URLs)
		itemSessions := mapToSessions(&item, vaults)
		sessions = append(sessions, itemSessions...)
	}

	return sessions, nil
}

// bulkGetItems efficiently fetches detailed item information using piped commands
// This eliminates N+1 queries by using: op item list ... | op item get -
func (c *Client) bulkGetItems(listArgs []string) ([]OPItem, error) {
	if err := check1Password(); err != nil {
		return nil, err
	}

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
	if err := check1Password(); err != nil {
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

// extractHostname extracts a meaningful hostname part for display
func extractHostname(urlStr string) string {
	// Remove protocol
	hostname := strings.TrimPrefix(urlStr, "https://")
	hostname = strings.TrimPrefix(hostname, "http://")

	// Remove trailing slash and path
	if idx := strings.Index(hostname, "/"); idx != -1 {
		hostname = hostname[:idx]
	}

	// For cases like "integration-tests-01.dtm-dev.stage.c8y.io",
	// extract the meaningful part before the common domain
	parts := strings.Split(hostname, ".")
	if len(parts) > 0 {
		// Take the first part which is usually the most meaningful
		firstPart := parts[0]

		// For very long hostnames, try to get a shorter meaningful name
		if len(firstPart) > 20 {
			// If it contains hyphens, take parts around them
			if strings.Contains(firstPart, "-") {
				subParts := strings.Split(firstPart, "-")
				if len(subParts) >= 2 {
					return strings.Join(subParts[:2], "-")
				}
			}
			return firstPart[:20] + "..."
		}
		return firstPart
	}

	return hostname
}
