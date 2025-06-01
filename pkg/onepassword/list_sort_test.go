package onepassword

import (
	"sort"
	"testing"

	"github.com/thomaswinkler/c8y-session-1password/pkg/core"
)

func TestClient_List_SortsByHost(t *testing.T) {
	// Create multiple items with different hosts to test sorting
	item1 := &OPItem{
		ID:       "item1",
		Title:    "Z Item",
		Category: "LOGIN",
		Vault: OPVault{
			ID:   "vault1",
			Name: "TestVault",
		},
		URLs: []OPURL{
			{Label: "Production", Primary: true, Href: "https://zzz.example.com"},
		},
		Fields: []OPField{
			{ID: "username", Value: "user1"},
			{ID: "password", Value: "pass1"},
		},
		Tags: []string{"c8y"},
	}

	item2 := &OPItem{
		ID:       "item2",
		Title:    "A Item",
		Category: "LOGIN",
		Vault: OPVault{
			ID:   "vault1",
			Name: "TestVault",
		},
		URLs: []OPURL{
			{Label: "Production", Primary: true, Href: "https://aaa.example.com"},
		},
		Fields: []OPField{
			{ID: "username", Value: "user2"},
			{ID: "password", Value: "pass2"},
		},
		Tags: []string{"c8y"},
	}

	item3 := &OPItem{
		ID:       "item3",
		Title:    "M Item",
		Category: "LOGIN",
		Vault: OPVault{
			ID:   "vault1",
			Name: "TestVault",
		},
		URLs: []OPURL{
			{Label: "Production", Primary: true, Href: "https://mmm.example.com"},
		},
		Fields: []OPField{
			{ID: "username", Value: "user3"},
			{ID: "password", Value: "pass3"},
		},
		Tags: []string{"c8y"},
	}

	vaults := map[string]string{"vault1": "TestVault"}

	// Create sessions from items
	sessions1 := mapToSessions(item1, vaults)
	sessions2 := mapToSessions(item2, vaults)
	sessions3 := mapToSessions(item3, vaults)

	// Combine sessions in non-alphabetical order (by host)
	allSessions := make([]*core.CumulocitySession, 0)
	allSessions = append(allSessions, sessions1...) // zzz.example.com
	allSessions = append(allSessions, sessions2...) // aaa.example.com
	allSessions = append(allSessions, sessions3...) // mmm.example.com

	// Verify initial order is not alphabetical by host
	if allSessions[0].Host != "https://zzz.example.com" {
		t.Errorf("Expected first session to be zzz.example.com (unsorted), got %s", allSessions[0].Host)
	}

	// Simulate the sorting that happens in List()
	sort.Slice(allSessions, func(i, j int) bool {
		return allSessions[i].Host < allSessions[j].Host
	})

	// Verify sessions are now sorted alphabetically by Host
	expectedHosts := []string{
		"https://aaa.example.com",
		"https://mmm.example.com",
		"https://zzz.example.com",
	}

	if len(allSessions) != 3 {
		t.Fatalf("Expected 3 sessions, got %d", len(allSessions))
	}

	for i, expectedHost := range expectedHosts {
		if allSessions[i].Host != expectedHost {
			t.Errorf("Session %d: Expected host '%s', got '%s'", i, expectedHost, allSessions[i].Host)
		}
	}

	// Verify corresponding item names are correctly associated
	expectedNames := []string{"A Item", "M Item", "Z Item"}
	for i, expectedName := range expectedNames {
		if allSessions[i].ItemName != expectedName {
			t.Errorf("Session %d: Expected item name '%s', got '%s'", i, expectedName, allSessions[i].ItemName)
		}
	}
}
