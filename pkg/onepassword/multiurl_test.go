package onepassword

import (
	"testing"
)

func TestMapToSessions_MultipleURLsInArray(t *testing.T) {
	// Test item with multiple URLs in urls array
	item := &OPItem{
		ID:       "test123",
		Title:    "Test Item",
		Category: "LOGIN",
		Vault: OPVault{
			ID:   "vault123",
			Name: "TestVault",
		},
		URLs: []OPURL{
			{Label: "Production", Primary: true, Href: "https://prod.example.com"},
			{Label: "Staging", Primary: false, Href: "https://staging.example.com"},
			{Label: "", Primary: false, Href: "https://dev.example.com"},
		},
		Fields: []OPField{
			{ID: "username", Value: "testuser"},
			{ID: "password", Value: "testpass"},
		},
		Tags: []string{"c8y"},
	}

	vaults := map[string]string{"vault123": "TestVault"}
	sessions := mapToSessions(item, vaults)

	// Should create 3 sessions, one for each URL
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions, got %d", len(sessions))
		return
	}

	// First session should be primary and have proper naming
	if sessions[0].Name != "Test Item (Production)" {
		t.Errorf("Expected first session name 'Test Item (Production)', got '%s'", sessions[0].Name)
	}
	if sessions[0].Host != "https://prod.example.com" {
		t.Errorf("Expected first session host 'https://prod.example.com', got '%s'", sessions[0].Host)
	}
}

func TestMapToSessions_URLFields(t *testing.T) {
	// Test item with URLs in custom fields (Type: "URL")
	item := &OPItem{
		ID:       "test456",
		Title:    "Test Item with URL Fields",
		Category: "LOGIN",
		Vault: OPVault{
			ID:   "vault123",
			Name: "TestVault",
		},
		URLs: []OPURL{}, // No URLs in urls array
		Fields: []OPField{
			{ID: "username", Value: "testuser"},
			{ID: "password", Value: "testpass"},
			{Label: "Production URL", Type: "URL", Value: "https://prod.field.com"},
			{Label: "Staging URL", Type: "URL", Value: "https://staging.field.com"},
			{Label: "website", Value: "https://website.field.com"},
		},
		Tags: []string{"c8y"},
	}

	vaults := map[string]string{"vault123": "TestVault"}
	sessions := mapToSessions(item, vaults)

	t.Logf("Created %d sessions", len(sessions))
	for i, session := range sessions {
		t.Logf("Session %d: Name=%s, Host=%s", i, session.Name, session.Host)
	}

	// Currently this will only create 1 session (the first website field found)
	// But we want it to create 3 sessions, one for each URL field
	if len(sessions) != 3 {
		t.Errorf("Expected 3 sessions for URL fields, got %d", len(sessions))
	}
}

func TestActualMultiURL(t *testing.T) {
	// Test with the actual structure from the 1Password item (anonymized)
	item := &OPItem{
		ID:       "item-uuid-123",
		Title:    "autouser tenants: tenant1.example.com, xyz-*.example.com",
		Category: "LOGIN",
		Vault: OPVault{
			ID:   "vault-uuid-456",
			Name: "Shared Vault",
		},
		URLs: []OPURL{
			{Label: "website", Href: "https://tenant1.example.com/"},
			{Label: "website", Primary: true, Href: "https://primary.example.com/"},
			{Label: "website", Href: "https://xyz-01.example.com/"},
			{Label: "website", Href: "https://xyz-02.example.com/"},
			{Label: "website", Href: "https://xyz-github.example.com/"},
		},
		Fields: []OPField{
			{ID: "username", Value: "user123"},
			{ID: "password", Value: "password123!"},
			{Label: "url", Type: "URL", Value: "https://secondary.example.com/"},
		},
		Tags: []string{"Shared/Instances", "example"},
	}

	vaults := map[string]string{"vault-uuid-456": "Shared Vault"}
	sessions := mapToSessions(item, vaults)

	t.Logf("Created %d sessions", len(sessions))
	for i, session := range sessions {
		t.Logf("Session %d: Name=%s, Host=%s", i, session.Name, session.Host)
	}

	// Should create 6 sessions (5 from URLs array + 1 from URL field)
	expectedSessions := 6
	if len(sessions) != expectedSessions {
		t.Errorf("Expected %d sessions, got %d", expectedSessions, len(sessions))
	}

	// Check that primary URL is first
	primaryFound := false
	for i, session := range sessions {
		if session.Host == "https://primary.example.com/" {
			if i != 0 {
				t.Errorf("Primary URL should be first, but found at position %d", i)
			}
			primaryFound = true
		}
	}

	if !primaryFound {
		t.Errorf("Primary URL not found in sessions")
	}
}
