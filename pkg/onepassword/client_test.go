package onepassword

import (
	"strings"
	"testing"
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
		uri           string
		expectedVault string
		expectedItem  string
		expectError   bool
	}{
		{"op://Employee/Cumulocity Production", "Employee", "Cumulocity Production", false},
		{"op://vault123/item456", "vault123", "item456", false},
		{"op://Personal/My Login", "Personal", "My Login", false},
		{"1password://abc123", "", "", true}, // Wrong scheme
		{"op://vault", "", "", true},         // Missing item
		{"op:///item", "", "", true},         // Missing vault
		{"", "", "", true},                   // Empty string
		{"invalid", "", "", true},            // Invalid format
	}

	for _, test := range tests {
		vault, item, err := ParseOPURI(test.uri)

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
		if !strings.Contains(session.SessionURI, "vault123/test123") {
			t.Errorf("Session %d: Expected SessionURI to contain vault/item, got '%s'", i, session.SessionURI)
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
	if strings.Contains(session.SessionURI, "#") {
		t.Errorf("Expected simple SessionURI without fragment, got '%s'", session.SessionURI)
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
