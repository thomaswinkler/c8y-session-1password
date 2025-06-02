package core

import (
	"testing"
)

func TestCumulocitySession_Description_NoVault(t *testing.T) {
	// Test to ensure vault is excluded from description to save space
	session := CumulocitySession{
		Username:   "testuser",
		Tenant:     "testtenant",
		VaultName:  "MyVault",
		Tags:       []string{"c8y"},
		SessionURI: "op://MyVault/test-item",
	}

	result := session.Description()

	// Should contain username, tenant, tags, and URI
	expectedParts := []string{
		"Username=testuser",
		"Tenant=testtenant",
		"Tags=c8y",
		"op://MyVault/test-item",
	}

	for _, part := range expectedParts {
		if !contains(result, part) {
			t.Errorf("Description() = %q; expected to contain %q", result, part)
		}
	}

	// Should NOT contain vault (since it's in the URI already)
	if contains(result, "Vault=MyVault") {
		t.Errorf("Description() = %q; should not contain vault name (space saving)", result)
	}

	// Should NOT contain "Vault=" at all
	if contains(result, "Vault=") {
		t.Errorf("Description() = %q; should not contain any vault information", result)
	}
}

func TestCumulocitySession_Description_Minimal(t *testing.T) {
	// Test minimal session with just username
	session := CumulocitySession{
		Username:   "user",
		SessionURI: "op://Vault/Item",
	}

	result := session.Description()
	expected := "Username=user | op://Vault/Item"

	if result != expected {
		t.Errorf("Description() = %q; expected %q", result, expected)
	}
}
