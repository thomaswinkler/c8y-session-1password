package cmd

import (
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/thomaswinkler/c8y-session-1password/pkg/core"
)

func TestIsNativeMessagingHost(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"started by chrome", []string{"chrome-extension://abcdefghijklmnop/"}, true},
		{"started by chrome on windows", []string{"chrome-extension://abcdefghijklmnop/", "--parent-window=1234"}, true},
		{"no arguments", []string{}, false},
		{"session provider", []string{"--reveal", "-v", "--vault", "Employee", "Production"}, false},
		{"origin not first argument", []string{"--reveal", "chrome-extension://abcdefghijklmnop/"}, false},
		{"filter", []string{"production"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNativeMessagingHost(tt.args); got != tt.expected {
				t.Errorf("isNativeMessagingHost(%v) = %v, expected %v", tt.args, got, tt.expected)
			}
		})
	}
}

// captureStdout returns what fn writes to stdout
func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	fnErr := fn()
	os.Stdout = stdout
	w.Close()
	out, _ := io.ReadAll(r)
	if fnErr != nil {
		t.Fatalf("unexpected error: %v", fnErr)
	}
	return string(out)
}

func TestOutputSessionList(t *testing.T) {
	sessions := []*core.CumulocitySession{
		{
			Name:       "Production",
			Host:       "https://prod.example.com",
			Username:   "admin",
			Password:   "secret-password",
			TOTP:       "123456",
			TOTPSecret: "secret-totp",
			ItemID:     "item1",
			VaultID:    "vault1",
		},
		{
			Name:     "Test",
			Host:     "https://test.example.com",
			Username: "tester",
			ItemID:   "item2",
			VaultID:  "vault1",
		},
	}

	t.Run("json without secrets", func(t *testing.T) {
		out := captureStdout(t, func() error { return outputSessionList(sessions, "json") })
		for _, secret := range []string{"secret-password", "123456", "secret-totp", "password", "totp"} {
			if strings.Contains(out, secret) {
				t.Errorf("output contains %q: %s", secret, out)
			}
		}
		var listed []core.CumulocitySession
		if err := json.Unmarshal([]byte(out), &listed); err != nil {
			t.Fatalf("output is not a JSON array: %v", err)
		}
		if len(listed) != 2 || listed[0].Host != "https://prod.example.com" || listed[1].ItemID != "item2" {
			t.Errorf("unexpected sessions: %+v", listed)
		}
		if sessions[0].Password != "secret-password" {
			t.Errorf("original session was modified")
		}
	})

	t.Run("empty json array", func(t *testing.T) {
		out := captureStdout(t, func() error { return outputSessionList(nil, "json") })
		if strings.TrimSpace(out) != "[]" {
			t.Errorf("expected [], got %q", out)
		}
	})

	t.Run("one uri per line", func(t *testing.T) {
		out := captureStdout(t, func() error { return outputSessionList(sessions, "uri") })
		expected := "op://vault1/item1?target_url=https%3A%2F%2Fprod.example.com\n" +
			"op://vault1/item2?target_url=https%3A%2F%2Ftest.example.com\n"
		if out != expected {
			t.Errorf("expected %q, got %q", expected, out)
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		if err := outputSessionList(sessions, "yaml"); err == nil {
			t.Errorf("expected error for unsupported format")
		}
	})
}

func TestResolveItem(t *testing.T) {
	tests := []struct {
		name              string
		vault             string
		item              string
		opURI             string
		envVault          string
		envItem           string
		expectedVault     string
		expectedItem      string
		expectedTargetURL string
		expectError       bool
	}{
		{name: "flags", vault: "Employee", item: "Production", expectedVault: "Employee", expectedItem: "Production"},
		{name: "environment variables", envVault: "Employee", envItem: "Production", expectedVault: "Employee", expectedItem: "Production"},
		{name: "flags before environment variables", vault: "Shared", item: "Test", envVault: "Employee", envItem: "Production", expectedVault: "Shared", expectedItem: "Test"},
		{name: "uri", opURI: "op://vault1/item1", expectedVault: "vault1", expectedItem: "item1"},
		{name: "uri with target url", opURI: "op://vault1/item1?target_url=https%3A%2F%2Fprod.example.com", expectedVault: "vault1", expectedItem: "item1", expectedTargetURL: "https://prod.example.com"},
		{name: "uri before environment variables", opURI: "op://vault1/item1", envVault: "Employee", envItem: "Production", expectedVault: "vault1", expectedItem: "item1"},
		{name: "uri before flags", opURI: "op://vault1/item1", vault: "Shared", item: "Test", expectedVault: "vault1", expectedItem: "item1"},
		{name: "invalid uri", opURI: "op://vault1", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("C8YOP_VAULT", tt.envVault)
			t.Setenv("C8YOP_ITEM", tt.envItem)
			t.Setenv("CYOP_VAULT", "")
			t.Setenv("CYOP_ITEM", "")

			vault, item, targetURL, err := resolveItem(tt.vault, tt.item, tt.opURI)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got vault=%q item=%q", vault, item)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if vault != tt.expectedVault || item != tt.expectedItem || targetURL != tt.expectedTargetURL {
				t.Errorf("resolveItem() = (%q, %q, %q), expected (%q, %q, %q)",
					vault, item, targetURL, tt.expectedVault, tt.expectedItem, tt.expectedTargetURL)
			}
		})
	}
}
