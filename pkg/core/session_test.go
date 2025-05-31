package core

import (
	"testing"
)

func TestCumulocitySession_FilterValue(t *testing.T) {
	session := CumulocitySession{
		SessionURI: "op://Employee/test-item",
		Host:       "https://example.cumulocity.com",
		Username:   "testuser",
	}

	expected := "op://Employee/test-item https://example.cumulocity.com testuser"
	result := session.FilterValue()

	if result != expected {
		t.Errorf("FilterValue() = %q; expected %q", result, expected)
	}
}

func TestCumulocitySession_Title(t *testing.T) {
	session := CumulocitySession{
		Host: "https://example.cumulocity.com",
	}

	expected := "https://example.cumulocity.com"
	result := session.Title()

	if result != expected {
		t.Errorf("Title() = %q; expected %q", result, expected)
	}
}

func TestCumulocitySession_Description(t *testing.T) {
	session := CumulocitySession{
		Username:   "testuser",
		Tenant:     "testtenant",
		VaultName:  "testvault",
		Tags:       []string{"c8y", "test"},
		SessionURI: "op://Employee/test-item",
	}

	result := session.Description()

	// Check that all expected components are in the description
	expectedParts := []string{
		"Username=testuser",
		"Tenant=testtenant",
		"Vault=testvault",
		"Tags=c8y,test",
		"uri=op://Employee/test-item",
	}

	for _, part := range expectedParts {
		if !contains(result, part) {
			t.Errorf("Description() = %q; expected to contain %q", result, part)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
