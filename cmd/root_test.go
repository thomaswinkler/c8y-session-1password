package cmd

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"reflect"
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
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
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
			VaultName:  "Employee",
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

	t.Run("json with vault as group", func(t *testing.T) {
		out := captureStdout(t, func() error { return outputSessionList(sessions, "json") })
		var listed []struct {
			Group string `json:"group"`
		}
		if err := json.Unmarshal([]byte(out), &listed); err != nil {
			t.Fatalf("output is not a JSON array: %v", err)
		}
		if len(listed) != 2 || listed[0].Group != "Employee" || listed[1].Group != "vault1" {
			t.Errorf("expected groups Employee and vault1 (vault ID without vault name), got %+v", listed)
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

// TestSessionWithoutSecrets makes sure that every field except the secrets is kept, so a field
// added to core.CumulocitySession later must be classified explicitly
func TestSessionWithoutSecrets(t *testing.T) {
	secrets := map[string]bool{"Password": true, "TOTP": true, "TOTPSecret": true}

	session := core.CumulocitySession{}
	value := reflect.ValueOf(&session).Elem()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		switch field.Kind() {
		case reflect.String:
			field.SetString("value-" + value.Type().Field(i).Name)
		case reflect.Slice:
			field.Set(reflect.ValueOf([]string{"value"}))
		default:
			t.Fatalf("unhandled field kind %s of field %s", field.Kind(), value.Type().Field(i).Name)
		}
	}

	listed := reflect.ValueOf(sessionWithoutSecrets(&session))
	for i := 0; i < listed.NumField(); i++ {
		name := listed.Type().Field(i).Name
		if secrets[name] {
			if !listed.Field(i).IsZero() {
				t.Errorf("secret field %s must not be listed", name)
			}
			continue
		}
		if !reflect.DeepEqual(listed.Field(i).Interface(), value.Field(i).Interface()) {
			t.Errorf("field %s is not copied by sessionWithoutSecrets", name)
		}
	}
}

// TestOutputSessionReveal makes sure that secrets are masked by default and only output
// unchanged when reveal is requested, for the CLI and for native messaging
func TestOutputSessionReveal(t *testing.T) {
	newSession := func() *core.CumulocitySession {
		return &core.CumulocitySession{
			Name:       "Production",
			Host:       "https://prod.example.com",
			Username:   "admin",
			Password:   "secret-password",
			TOTP:       "123456",
			TOTPSecret: "secret-totp",
		}
	}

	// decodeNativeMessage checks the 4-byte length prefix and returns the JSON payload
	decodeNativeMessage := func(t *testing.T, out string) []byte {
		t.Helper()
		if len(out) < 4 {
			t.Fatalf("native message too short: %q", out)
		}
		length := binary.LittleEndian.Uint32([]byte(out[:4]))
		payload := []byte(out[4:])
		if int(length) != len(payload) {
			t.Fatalf("length prefix %d does not match payload length %d", length, len(payload))
		}
		return payload
	}

	tests := []struct {
		name     string
		reveal   bool
		expected core.CumulocitySession
	}{
		{
			name:   "masked by default",
			reveal: false,
			expected: core.CumulocitySession{
				Password: "***", TOTP: "***", TOTPSecret: "***",
			},
		},
		{
			name:   "revealed",
			reveal: true,
			expected: core.CumulocitySession{
				Password: "secret-password", TOTP: "123456", TOTPSecret: "secret-totp",
			},
		},
	}

	assertSecrets := func(t *testing.T, got core.CumulocitySession, expected core.CumulocitySession) {
		t.Helper()
		if got.Password != expected.Password || got.TOTP != expected.TOTP || got.TOTPSecret != expected.TOTPSecret {
			t.Errorf("expected password=%q totp=%q totpSecret=%q, got password=%q totp=%q totpSecret=%q",
				expected.Password, expected.TOTP, expected.TOTPSecret, got.Password, got.TOTP, got.TOTPSecret)
		}
		if got.Host != "https://prod.example.com" || got.Username != "admin" {
			t.Errorf("non-secret fields not output: %+v", got)
		}
	}

	for _, tt := range tests {
		t.Run("cli json "+tt.name, func(t *testing.T) {
			session := newSession()
			out := captureStdout(t, func() error { return outputSession(session, tt.reveal, "json") })
			var got core.CumulocitySession
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("output is not a JSON object: %v", err)
			}
			assertSecrets(t, got, tt.expected)
			if session.Password != "secret-password" {
				t.Errorf("original session was modified")
			}
		})

		t.Run("native messaging single "+tt.name, func(t *testing.T) {
			session := newSession()
			out := captureStdout(t, func() error { return outputSessionNativeMessaging(session, tt.reveal) })
			var got core.CumulocitySession
			if err := json.Unmarshal(decodeNativeMessage(t, out), &got); err != nil {
				t.Fatalf("payload is not a JSON object: %v", err)
			}
			assertSecrets(t, got, tt.expected)
			if session.Password != "secret-password" {
				t.Errorf("original session was modified")
			}
		})

		t.Run("native messaging multiple "+tt.name, func(t *testing.T) {
			sessions := []*core.CumulocitySession{newSession(), newSession()}
			out := captureStdout(t, func() error { return outputSessionsNativeMessaging(sessions, tt.reveal) })
			var got []core.CumulocitySession
			if err := json.Unmarshal(decodeNativeMessage(t, out), &got); err != nil {
				t.Fatalf("payload is not a JSON array: %v", err)
			}
			if len(got) != 2 {
				t.Fatalf("expected 2 sessions, got %d", len(got))
			}
			for _, session := range got {
				assertSecrets(t, session, tt.expected)
			}
		})
	}
}

func TestParseFilterArg(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		uriFlagSet bool
		filter     string
		uri        string
		wantErr    bool
	}{
		{name: "no argument"},
		{name: "filter", args: []string{"production"}, filter: "production"},
		{name: "uri", args: []string{"op://vault1/item1"}, uri: "op://vault1/item1"},
		{
			name: "uri with target_url",
			args: []string{"op://vault1/item1?target_url=https%3A%2F%2Fprod.example.com"},
			uri:  "op://vault1/item1?target_url=https%3A%2F%2Fprod.example.com",
		},
		{name: "filter with --uri", args: []string{"production"}, uriFlagSet: true, filter: "production"},
		{name: "uri with --uri", args: []string{"op://vault1/item1"}, uriFlagSet: true, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, uri, err := parseFilterArg(tt.args, tt.uriFlagSet)
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if filter != tt.filter || uri != tt.uri {
				t.Errorf("expected filter %q and uri %q, got %q and %q", tt.filter, tt.uri, filter, uri)
			}
		})
	}
}

func TestApplyDefaultMode(t *testing.T) {
	newSessions := func() []*core.CumulocitySession {
		return []*core.CumulocitySession{{Host: "https://a.example.com"}, {Host: "https://b.example.com", Mode: "prod"}}
	}

	sessions := newSessions()
	applyDefaultMode(sessions, "dev")
	if sessions[0].Mode != "dev" {
		t.Errorf("expected the default mode for a session without mode, got %q", sessions[0].Mode)
	}
	if sessions[1].Mode != "prod" {
		t.Errorf("expected the mode of the item to take precedence, got %q", sessions[1].Mode)
	}

	sessions = newSessions()
	applyDefaultMode(sessions, "")
	if sessions[0].Mode != "" || sessions[1].Mode != "prod" {
		t.Errorf("expected no change without default mode, got %q and %q", sessions[0].Mode, sessions[1].Mode)
	}
}
