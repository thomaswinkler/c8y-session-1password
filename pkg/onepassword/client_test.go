package onepassword

import (
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
