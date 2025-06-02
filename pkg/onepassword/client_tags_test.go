package onepassword

import (
	"strings"
	"testing"

	"github.com/thomaswinkler/c8y-session-1password/pkg/core"
)

func TestCreateSessionWithTags_FiltersTags(t *testing.T) {
	client := NewClient("test-vault", "c8y", "prod")

	// Mock item with multiple tags
	item := &OPItem{
		ID:    "test-id",
		Title: "Test Service",
		Tags:  []string{"Shared", "C8Y instances", "DTM", "c8y", "production", "test"},
		URLs: []OPURL{
			{Href: "https://test.cumulocity.com"},
		},
		Fields: []OPField{
			{ID: "username", Value: "testuser"},
			{ID: "password", Value: "testpass"},
		},
		Vault: OPVault{
			ID:   "vault123",
			Name: "TestVault",
		},
	}

	// Extract fields and URLs
	fields := item.extractFields()
	vaultName := "TestVault"
	urlSource := URLSource{URL: "https://test.cumulocity.com", Label: "", Primary: false, Source: "urls"}
	sessionName := "Test Service"
	sessionURI := "op://vault123/test-id"

	session := client.createSessionWithTags(item, fields, vaultName, urlSource, sessionName, sessionURI)

	// Should only contain matching tags from requested tags: "c8y", "prod"
	// From item tags ["Shared", "C8Y instances", "DTM", "c8y", "production", "test"]
	// Should match: "c8y" (exact match)
	// "prod" doesn't match anything (case-insensitive exact matching)
	expectedTags := []string{"c8y"}

	if len(session.Tags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d: %v", len(expectedTags), len(session.Tags), session.Tags)
	}

	for _, expectedTag := range expectedTags {
		found := false
		for _, sessionTag := range session.Tags {
			if strings.EqualFold(sessionTag, expectedTag) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag '%s' not found in session tags: %v", expectedTag, session.Tags)
		}
	}

	// Test the display
	description := session.Description()
	if !strings.Contains(description, "Tags=c8y") {
		t.Errorf("Expected description to contain 'Tags=c8y', got: %s", description)
	}

	// Should not contain unmatched tags
	if strings.Contains(description, "Shared") || strings.Contains(description, "DTM") {
		t.Errorf("Description should not contain unmatched tags, got: %s", description)
	}
}

func TestCreateSessionWithTags_NoMatchingTags(t *testing.T) {
	client := NewClient("test-vault", "nonexistent")

	item := &OPItem{
		ID:    "test-id",
		Title: "Test Service",
		Tags:  []string{"Shared", "C8Y instances", "DTM"},
		URLs: []OPURL{
			{Href: "https://test.cumulocity.com"},
		},
		Fields: []OPField{
			{ID: "username", Value: "testuser"},
			{ID: "password", Value: "testpass"},
		},
		Vault: OPVault{
			ID:   "vault123",
			Name: "TestVault",
		},
	}

	// Extract fields and URLs
	fields := item.extractFields()
	vaultName := "TestVault"
	urlSource := URLSource{URL: "https://test.cumulocity.com", Label: "", Primary: false, Source: "urls"}
	sessionName := "Test Service"
	sessionURI := "op://vault123/test-id"

	session := client.createSessionWithTags(item, fields, vaultName, urlSource, sessionName, sessionURI)

	// Should have no tags since none match
	if len(session.Tags) != 0 {
		t.Errorf("Expected 0 tags, got %d: %v", len(session.Tags), session.Tags)
	}

	// Description should not show Tags section when no tags
	description := session.Description()
	if strings.Contains(description, "Tags=") {
		t.Errorf("Description should not contain 'Tags=' when no matching tags, got: %s", description)
	}
}

func TestCreateSessionWithTags_CaseInsensitive(t *testing.T) {
	client := NewClient("test-vault", "C8Y", "shared")

	item := &OPItem{
		ID:    "test-id",
		Title: "Test Service",
		Tags:  []string{"Shared", "c8y", "production"},
		URLs: []OPURL{
			{Href: "https://test.cumulocity.com"},
		},
		Fields: []OPField{
			{ID: "username", Value: "testuser"},
			{ID: "password", Value: "testpass"},
		},
		Vault: OPVault{
			ID:   "vault123",
			Name: "TestVault",
		},
	}

	// Extract fields and URLs
	fields := item.extractFields()
	vaultName := "TestVault"
	urlSource := URLSource{URL: "https://test.cumulocity.com", Label: "", Primary: false, Source: "urls"}
	sessionName := "Test Service"
	sessionURI := "op://vault123/test-id"

	session := client.createSessionWithTags(item, fields, vaultName, urlSource, sessionName, sessionURI)

	// Should match both "C8Y" -> "c8y" and "shared" -> "Shared" (case insensitive)
	expectedCount := 2
	if len(session.Tags) != expectedCount {
		t.Errorf("Expected %d tags, got %d: %v", expectedCount, len(session.Tags), session.Tags)
	}

	// Check that both tags are present (preserving original case from item)
	hasC8Y := false
	hasShared := false
	for _, tag := range session.Tags {
		if strings.EqualFold(tag, "c8y") {
			hasC8Y = true
		}
		if strings.EqualFold(tag, "shared") {
			hasShared = true
		}
	}

	if !hasC8Y {
		t.Errorf("Expected to find 'c8y' tag, got: %v", session.Tags)
	}
	if !hasShared {
		t.Errorf("Expected to find 'Shared' tag, got: %v", session.Tags)
	}
}

// Helper function for testing tag filtering behavior
func (c *Client) createSessionWithTags(item *OPItem, fields itemFields, vaultName string, urlSource URLSource, sessionName, sessionURI string) *core.CumulocitySession {
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

	coreFields := core.ItemFields{
		Username:   fields.username,
		Password:   fields.password,
		TOTPSecret: fields.totpSecret,
		Tenant:     fields.tenant,
	}

	coreURLs := []core.URLSource{{
		URL:     urlSource.URL,
		Label:   urlSource.Label,
		Primary: urlSource.Primary,
		Source:  urlSource.Source,
	}}

	// Use the core function with tag filtering
	sessions := core.MapToSessions(coreItem, coreFields, coreURLs, vaultName, c.Tags)

	// Return the first session (there should only be one for a single URL)
	if len(sessions) > 0 {
		return sessions[0]
	}

	// This shouldn't happen in normal usage
	return &core.CumulocitySession{}
}
