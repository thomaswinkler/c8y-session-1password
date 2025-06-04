package core

import (
	"fmt"
	"strings"
)

type CumulocitySession struct {
	SessionURI string `json:"sessionUri,omitempty"`
	Name       string `json:"name,omitempty"`
	Host       string `json:"host,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Tenant     string `json:"tenant,omitempty"`
	TOTP       string `json:"totp,omitempty"`
	TOTPSecret string `json:"totpSecret,omitempty"`

	// 1Password specific
	ItemID    string   `json:"itemId,omitempty"`
	ItemName  string   `json:"itemName,omitempty"`
	VaultID   string   `json:"vaultId,omitempty"`
	VaultName string   `json:"vaultName,omitempty"`
	Tags      []string `json:"tags,omitempty"` // Only the matching requested tags
}

// URLSource represents a URL from any source (URLs array or fields)
type URLSource struct {
	URL     string
	Label   string
	Primary bool
	Source  string // "urls" or "field"
}

// ItemFields contains extracted fields from a 1Password item
type ItemFields struct {
	Username   string
	Password   string
	TOTPSecret string
	Tenant     string
}

// Item represents a simplified 1Password item for session creation
type Item struct {
	ID    string
	Title string
	Tags  []string
	Vault Vault
}

// Vault represents a 1Password vault
type Vault struct {
	ID   string
	Name string
}

// Session building utilities
// These handle the creation and management of sessions from 1Password items

// NormalizeDisplayURL removes protocol and trailing slash for better display and sorting
// Uses robust protocol parsing by splitting on "://" separator
func NormalizeDisplayURL(url string) string {
	if url == "" {
		return url
	}

	// Split on "://" to handle any protocol (http, https, ftp, etc.)
	parts := strings.Split(url, "://")
	var normalized string
	if len(parts) > 1 {
		// Use the part after the protocol
		normalized = parts[1]
	} else {
		// No protocol found, use the original URL
		normalized = url
	}

	// Remove trailing slash
	normalized = strings.TrimSuffix(normalized, "/")

	return normalized
}

func (i CumulocitySession) FilterValue() string {
	return strings.Join([]string{i.SessionURI, i.Host, i.Username}, " ")
}

func (i CumulocitySession) Title() string {
	return NormalizeDisplayURL(i.Host)
}

func (i CumulocitySession) Description() string {
	fields := []string{
		"Username=%s",
	}
	args := []any{
		i.Username,
	}

	if i.Tenant != "" {
		fields = append(fields, ", Tenant=%s")
		args = append(args, i.Tenant)
	}

	if len(i.Tags) > 0 {
		fields = append(fields, ", Tags=%s")
		args = append(args, strings.Join(i.Tags, ","))
	}

	fields = append(fields, " | %s")
	var vault, item, session string
	vault = i.VaultName
	if vault == "" {
		vault = i.VaultID
	}
	item = i.ItemName
	if item == "" {
		item = i.ItemID
	}

	session = i.SessionURI
	if item != "" && vault != "" {
		session = BuildSessionURI(vault, item)
	}
	args = append(args, session)

	return fmt.Sprintf(strings.Join(fields, ""), args...)
}

// BuildSessionName creates an appropriate name for a session based on URL source and count
func BuildSessionName(item Item, urlSource URLSource, urlIndex int, totalURLs int, labelCounts map[string]int) string {
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

func BuildSessionURI(vault, item string) string {
	return fmt.Sprintf("op://%s/%s", vault, item)
}

// CreateSession builds a CumulocitySession from extracted data
// If useFiltering is true, filteredTags will be used; otherwise all item tags are used
func CreateSession(item Item, fields ItemFields, vaultName string, urlSource URLSource, sessionName, sessionURI string, filteredTags []string, useFiltering bool) *CumulocitySession {
	var tags []string
	if useFiltering {
		tags = filteredTags // Use filtered tags (could be empty)
	} else {
		tags = item.Tags // Use all item tags
	}

	return &CumulocitySession{
		SessionURI: sessionURI,
		Name:       sessionName,
		ItemID:     item.ID,
		ItemName:   item.Title,
		Username:   fields.Username,
		Password:   fields.Password,
		Tenant:     fields.Tenant,
		Host:       urlSource.URL,
		VaultID:    item.Vault.ID,
		VaultName:  vaultName,
		TOTPSecret: fields.TOTPSecret,
		Tags:       tags,
	}
}

// FilterMatchingTags returns only the tags from item that match the requested tags
func FilterMatchingTags(itemTags []string, requestedTags []string) []string {
	if len(requestedTags) == 0 {
		return nil // Return nil to indicate no filtering should be done
	}

	var matchingTags []string
	for _, itemTag := range itemTags {
		for _, requestedTag := range requestedTags {
			if strings.EqualFold(itemTag, requestedTag) {
				matchingTags = append(matchingTags, itemTag)
				break
			}
		}
	}
	return matchingTags
}

// MapToSessions creates one or more sessions from a 1Password item, handling multiple URLs
// If requestedTags is provided, only matching tags will be included in the sessions
func MapToSessions(item Item, fields ItemFields, allURLs []URLSource, vaultName string, requestedTags []string) []*CumulocitySession {
	// Filter tags if requested
	var filteredTags []string
	var useFiltering bool
	if len(requestedTags) > 0 {
		filteredTags = FilterMatchingTags(item.Tags, requestedTags)
		useFiltering = true
	}

	// If no URLs found anywhere, create one session without URL
	if len(allURLs) == 0 {
		emptyURL := URLSource{URL: "", Label: "", Primary: false, Source: "none"}
		sessionName := BuildSessionName(item, emptyURL, 0, 1, nil)
		sessionURI := BuildSessionURI(item.Vault.ID, item.ID)
		session := CreateSession(item, fields, vaultName, emptyURL, sessionName, sessionURI, filteredTags, useFiltering)
		return []*CumulocitySession{session}
	}

	// Pre-calculate label counts for naming decisions
	labelCounts := make(map[string]int)
	for _, url := range allURLs {
		labelCounts[url.Label]++
	}

	// Create sessions for all URLs
	sessions := make([]*CumulocitySession, 0, len(allURLs))
	for i, urlSource := range allURLs {
		sessionName := BuildSessionName(item, urlSource, i, len(allURLs), labelCounts)
		sessionURI := BuildSessionURI(item.Vault.ID, item.ID)
		session := CreateSession(item, fields, vaultName, urlSource, sessionName, sessionURI, filteredTags, useFiltering)
		sessions = append(sessions, session)
	}

	return sessions
}

// extractHostname extracts a meaningful hostname part for display
func extractHostname(urlStr string) string {
	// Remove protocol
	hostname := NormalizeDisplayURL(urlStr)

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

// FilterSessions filters sessions based on a query string that matches against
// session name, item name, or host URL (case-insensitive)
func FilterSessions(sessions []*CumulocitySession, filter string) []*CumulocitySession {
	if filter == "" {
		return sessions
	}

	filter = strings.ToLower(filter)
	var filtered []*CumulocitySession

	for _, session := range sessions {
		// Check if filter matches any of these fields (case-insensitive)
		if strings.Contains(strings.ToLower(session.Name), filter) ||
			strings.Contains(strings.ToLower(session.ItemName), filter) ||
			strings.Contains(strings.ToLower(session.Host), filter) {
			filtered = append(filtered, session)
		}
	}

	return filtered
}
