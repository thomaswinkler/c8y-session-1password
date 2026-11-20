package onepassword

import (
	"net/url"
	"strings"
	"testing"

	"github.com/thomaswinkler/c8y-session-1password/pkg/core"
)

func TestIsUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"abcdefghijklmnopqrstuvwxyz", true},     // Valid 26-char alphanumeric
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZ", true},     // Valid 26-char alphanumeric uppercase
		{"abc123XYZ789abc123XYZ789ab", true},     // Valid 26-char mixed
		{"abc", false},                           // Too short
		{"abcdefghijklmnopqrstuvwxyz123", false}, // Too long
		{"abcdefghijklmnopqrstuvwxy!", false},    // Contains special character
		{"", false},                              // Empty string
	}

	for _, test := range tests {
		result := isUID(test.input)
		if result != test.expected {
			t.Errorf("isUID(%q) = %v; expected %v", test.input, result, test.expected)
		}
	}
}

func TestNewClient(t *testing.T) {
	vault := "test-vault"
	tags := []string{"c8y", "test"}

	client := NewClient(vault, tags...)

	if client.Vault != vault {
		t.Errorf("Expected vault %q, got %q", vault, client.Vault)
	}

	if len(client.Tags) != len(tags) {
		t.Errorf("Expected %d tags, got %d", len(tags), len(client.Tags))
	}

	for i, tag := range tags {
		if client.Tags[i] != tag {
			t.Errorf("Expected tag %q at index %d, got %q", tag, i, client.Tags[i])
		}
	}
}

func TestParseOPURI(t *testing.T) {
	tests := []struct {
		uri               string
		expectedVault     string
		expectedItem      string
		expectedTargetURL string
		expectError       bool
	}{
		{"op://Employee/Cumulocity Production", "Employee", "Cumulocity Production", "", false},
		{"op://vault123/item456", "vault123", "item456", "", false},
		{"op://Personal/My Login", "Personal", "My Login", "", false},
		{"op://vault123/item456?target_url=https%3A%2F%2Fexample.cumulocity.com", "vault123", "item456", "https://example.cumulocity.com", false}, // Output of --output uri
		{"op://vault123/item456?other=value", "vault123", "item456", "", false},                                                                   // Unknown parameter
		{"op://vault123/item456?target_url=%zz", "", "", "", true},                                                                                // Invalid parameter
		{"1password://abc123", "", "", "", true},                                                                                                  // Wrong scheme
		{"op://vault", "", "", "", true},                                                                                                          // Missing item
		{"op:///item", "", "", "", true},                                                                                                          // Missing vault
		{"op://vault?target_url=x", "", "", "", true},                                                                                             // Missing item with parameter
		{"", "", "", "", true},        // Empty string
		{"invalid", "", "", "", true}, // Invalid format
	}

	for _, test := range tests {
		vault, item, targetURL, err := ParseOPURI(test.uri)

		if test.expectError {
			if err == nil {
				t.Errorf("ParseOPURI(%q) expected error but got none", test.uri)
			}
		} else {
			if err != nil {
				t.Errorf("ParseOPURI(%q) unexpected error: %v", test.uri, err)
			}
			if vault != test.expectedVault {
				t.Errorf("ParseOPURI(%q) vault = %q; expected %q", test.uri, vault, test.expectedVault)
			}
			if item != test.expectedItem {
				t.Errorf("ParseOPURI(%q) item = %q; expected %q", test.uri, item, test.expectedItem)
			}
			if targetURL != test.expectedTargetURL {
				t.Errorf("ParseOPURI(%q) targetURL = %q; expected %q", test.uri, targetURL, test.expectedTargetURL)
			}
		}
	}
}

func TestParseVaultNames(t *testing.T) {
	tests := []struct {
		name     string
		vault    string
		expected []string
	}{
		{
			name:     "single vault",
			vault:    "Employee",
			expected: []string{"Employee"},
		},
		{
			name:     "multiple vaults",
			vault:    "Employee,Shared",
			expected: []string{"Employee", "Shared"},
		},
		{
			name:     "multiple vaults with spaces",
			vault:    "Employee, Shared, Personal",
			expected: []string{"Employee", "Shared", "Personal"},
		},
		{
			name:     "empty vault",
			vault:    "",
			expected: []string{},
		},
		{
			name:     "vault with empty entries",
			vault:    "Employee,,Shared,",
			expected: []string{"Employee", "Shared"},
		},
		{
			name:     "single vault with trailing comma",
			vault:    "Employee,",
			expected: []string{"Employee"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := NewClient(test.vault)
			result := client.parseVaultNames()

			if len(result) != len(test.expected) {
				t.Errorf("parseVaultNames() returned %d vaults, expected %d", len(result), len(test.expected))
				return
			}

			for i, expected := range test.expected {
				if result[i] != expected {
					t.Errorf("parseVaultNames()[%d] = %q; expected %q", i, result[i], expected)
				}
			}
		})
	}
}

func TestParseVaultNamesFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single vault",
			input:    "vault1",
			expected: []string{"vault1"},
		},
		{
			name:     "multiple vaults",
			input:    "vault1,vault2,vault3",
			expected: []string{"vault1", "vault2", "vault3"},
		},
		{
			name:     "multiple vaults with spaces",
			input:    "vault1, vault2 , vault3",
			expected: []string{"vault1", "vault2", "vault3"},
		},
		{
			name:     "empty vault",
			input:    "",
			expected: []string{},
		},
		{
			name:     "vault with empty entries",
			input:    "vault1,,vault2,",
			expected: []string{"vault1", "vault2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parseVaultNamesFromString(test.input)
			if len(result) != len(test.expected) {
				t.Errorf("parseVaultNamesFromString() returned %d vaults, expected %d", len(result), len(test.expected))
				return
			}
			for i, expected := range test.expected {
				if result[i] != expected {
					t.Errorf("parseVaultNamesFromString()[%d] = %q; expected %q", i, result[i], expected)
				}
			}
		})
	}
}

func TestMapToSessions_MultipleURLs(t *testing.T) {
	item := &OPItem{
		ID:       "test123",
		Title:    "Test Service",
		Category: "LOGIN",
		Vault: OPVault{
			ID:   "vault123",
			Name: "Test Vault",
		},
		Fields: []OPField{
			{ID: "username", Value: "testuser"},
			{ID: "password", Value: "testpass"},
			{Label: "Tenant", Value: "testtenant"},
		},
		URLs: []OPURL{
			{Label: "Production", Primary: true, Href: "https://prod.example.com"},
			{Label: "Staging", Primary: false, Href: "https://staging.example.com"},
			{Label: "Development", Primary: false, Href: "https://dev.example.com"},
		},
		Tags: []string{"c8y", "test"},
	}

	vaults := map[string]string{"vault123": "Test Vault"}
	sessions := mapToSessions(item, vaults)

	// Should create 3 sessions for 3 URLs
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
	}

	// First session should be primary (sorted first)
	if !strings.Contains(sessions[0].Name, "Production") {
		t.Errorf("Expected first session to be Production, got %s", sessions[0].Name)
	}
	if sessions[0].Host != "https://prod.example.com" {
		t.Errorf("Expected first session host to be prod.example.com, got %s", sessions[0].Host)
	}

	// Check that all sessions have the same basic info
	for i, session := range sessions {
		if session.Username != "testuser" {
			t.Errorf("Session %d: Expected username 'testuser', got '%s'", i, session.Username)
		}
		if session.Tenant != "testtenant" {
			t.Errorf("Session %d: Expected tenant 'testtenant', got '%s'", i, session.Tenant)
		}
		if session.ItemID != "test123" {
			t.Errorf("Session %d: Expected ItemID 'test123', got '%s'", i, session.ItemID)
		}
		// The URL identifies the session, as the item has multiple URLs
		expectedURI := "op://vault123/test123?target_url=" + url.QueryEscape(session.Host)
		if session.SessionURI != expectedURI {
			t.Errorf("Session %d: Expected SessionURI '%s', got '%s'", i, expectedURI, session.SessionURI)
		}
	}

	// Check that names are properly differentiated
	expectedNames := []string{"Test Service (Production)", "Test Service (Staging)", "Test Service (Development)"}
	for i, expectedName := range expectedNames {
		if sessions[i].Name != expectedName {
			t.Errorf("Session %d: Expected name '%s', got '%s'", i, expectedName, sessions[i].Name)
		}
	}
}

func TestMapToSessions_SingleURL(t *testing.T) {
	item := &OPItem{
		ID:       "test456",
		Title:    "Single URL Service",
		Category: "LOGIN",
		Vault: OPVault{
			ID:   "vault456",
			Name: "Test Vault",
		},
		Fields: []OPField{
			{ID: "username", Value: "singleuser"},
			{ID: "password", Value: "singlepass"},
		},
		URLs: []OPURL{
			{Label: "", Primary: true, Href: "https://single.example.com"},
		},
		Tags: []string{"c8y"},
	}

	vaults := map[string]string{"vault456": "Test Vault"}
	sessions := mapToSessions(item, vaults)

	// Should create 1 session
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	session := sessions[0]
	if session.Name != "Single URL Service" {
		t.Errorf("Expected name 'Single URL Service', got '%s'", session.Name)
	}
	if session.Host != "https://single.example.com" {
		t.Errorf("Expected host 'https://single.example.com', got '%s'", session.Host)
	}
	// Without target_url, so the 1Password CLI can use it
	if session.SessionURI != "op://vault456/test456" {
		t.Errorf("Expected SessionURI 'op://vault456/test456', got '%s'", session.SessionURI)
	}
}

func TestMapToSessions_FallbackURL(t *testing.T) {
	item := &OPItem{
		ID:       "test789",
		Title:    "Fallback Service",
		Category: "LOGIN",
		Vault: OPVault{
			ID:   "vault789",
			Name: "Test Vault",
		},
		Fields: []OPField{
			{ID: "username", Value: "fallbackuser"},
			{ID: "password", Value: "fallbackpass"},
			{Label: "website", Value: "https://fallback.example.com"},
		},
		URLs: []OPURL{}, // No URLs in urls array
		Tags: []string{"c8y"},
	}

	vaults := map[string]string{"vault789": "Test Vault"}
	sessions := mapToSessions(item, vaults)

	// Should create 1 session using fallback URL
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	session := sessions[0]
	if session.Host != "https://fallback.example.com" {
		t.Errorf("Expected fallback host 'https://fallback.example.com', got '%s'", session.Host)
	}
}

func TestOPItem_Skip_WithFallbackURL(t *testing.T) {
	tests := []struct {
		name     string
		item     OPItem
		expected bool
	}{
		{
			name: "Has URLs array - don't skip",
			item: OPItem{
				Category: "LOGIN",
				URLs:     []OPURL{{Href: "https://example.com"}},
			},
			expected: false,
		},
		{
			name: "No URLs but has website field - don't skip",
			item: OPItem{
				Category: "LOGIN",
				URLs:     []OPURL{},
				Fields:   []OPField{{Label: "website", Value: "https://example.com"}},
			},
			expected: false,
		},
		{
			name: "No URLs but has URL field - don't skip",
			item: OPItem{
				Category: "LOGIN",
				URLs:     []OPURL{},
				Fields:   []OPField{{Label: "URL", Value: "https://example.com"}},
			},
			expected: false,
		},
		{
			name: "No URLs and no fallback fields - skip",
			item: OPItem{
				Category: "LOGIN",
				URLs:     []OPURL{},
				Fields:   []OPField{{Label: "notes", Value: "some notes"}},
			},
			expected: true,
		},
		{
			name: "Not a LOGIN item - skip",
			item: OPItem{
				Category: "SECURE_NOTE",
				URLs:     []OPURL{{Href: "https://example.com"}},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.item.Skip()
			if result != test.expected {
				t.Errorf("Skip() = %v; expected %v", result, test.expected)
			}
		})
	}
}

// Test helper function for backward compatibility with existing tests
func mapToSessions(item *OPItem, vaults map[string]string) []*core.CumulocitySession {
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

	// Determine vault name for URI
	vaultName := item.Vault.Name
	if name, found := vaults[item.Vault.ID]; found {
		vaultName = name
	}

	// No tag filtering for backward compatibility
	return core.MapToSessions(coreItem, coreFields, coreURLs, vaultName, nil)
}

func TestSelectSession(t *testing.T) {
	sessions := []*core.CumulocitySession{
		{Name: "Production (URL 1)", ItemName: "Production", Host: "https://prod.example.com"},
		{Name: "Production (URL 2)", ItemName: "Production", Host: "https://prod2.example.com/"},
	}

	tests := []struct {
		targetURL    string
		expectedName string
		expectError  bool
	}{
		{"", "Production (URL 1)", false},                           // First session without target URL
		{"https://prod2.example.com/", "Production (URL 2)", false}, // Exact match
		{"https://PROD2.example.com", "Production (URL 2)", false},  // Case and trailing slash ignored
		{"https://other.example.com", "", true},                     // URL of another item
	}

	for _, test := range tests {
		session, err := selectSession(sessions, test.targetURL)
		if test.expectError {
			if err == nil {
				t.Errorf("selectSession(%q) expected error but got session %q", test.targetURL, session.Name)
			}
			continue
		}
		if err != nil {
			t.Errorf("selectSession(%q) unexpected error: %v", test.targetURL, err)
			continue
		}
		if session.Name != test.expectedName {
			t.Errorf("selectSession(%q) = %q; expected %q", test.targetURL, session.Name, test.expectedName)
		}
	}
}
