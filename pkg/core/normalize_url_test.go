package core

import "testing"

func TestNormalizeURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTPS URL with trailing slash",
			input:    "https://example.com/",
			expected: "example.com",
		},
		{
			name:     "HTTP URL with trailing slash",
			input:    "http://example.com/",
			expected: "example.com",
		},
		{
			name:     "HTTPS URL without trailing slash",
			input:    "https://example.com",
			expected: "example.com",
		},
		{
			name:     "HTTP URL without trailing slash",
			input:    "http://example.com",
			expected: "example.com",
		},
		{
			name:     "FTP URL",
			input:    "ftp://files.example.com/",
			expected: "files.example.com",
		},
		{
			name:     "Custom protocol",
			input:    "myprotocol://custom.example.com/",
			expected: "custom.example.com",
		},
		{
			name:     "No protocol",
			input:    "example.com",
			expected: "example.com",
		},
		{
			name:     "No protocol with trailing slash",
			input:    "example.com/",
			expected: "example.com",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Complex URL with path",
			input:    "https://api.example.com/v1/endpoint/",
			expected: "api.example.com/v1/endpoint",
		},
		{
			name:     "URL with port",
			input:    "https://example.com:8080/",
			expected: "example.com:8080",
		},
		{
			name:     "URL with subdomain",
			input:    "https://api.tenant.cumulocity.com/",
			expected: "api.tenant.cumulocity.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeDisplayURL(tc.input)
			if result != tc.expected {
				t.Errorf("NormalizeURL(%q) = %q; expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestBaseURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"base URL", "https://example.cumulocity.com", "https://example.cumulocity.com"},
		{"trailing slash", "https://example.cumulocity.com/", "https://example.cumulocity.com"},
		{"login path", "https://example.cumulocity.com/apps/cockpit/index.html#/", "https://example.cumulocity.com"},
		{"application path with query", "https://example.cumulocity.com/apps/devicemanagement/index.html?x=1#/device/123", "https://example.cumulocity.com"},
		{"port", "https://example.cumulocity.com:8443/apps/cockpit", "https://example.cumulocity.com:8443"},
		{"http", "http://localhost:8080/apps/cockpit/", "http://localhost:8080"},
		{"user info is removed", "https://user@example.cumulocity.com/apps", "https://example.cumulocity.com"},
		{"without scheme", "example.cumulocity.com/apps/cockpit", "example.cumulocity.com"},
		{"without scheme and path", "example.cumulocity.com", "example.cumulocity.com"},
		{"whitespace", "  https://example.cumulocity.com/apps  ", "https://example.cumulocity.com"},
		{"empty", "", ""},
		{"not a URL", "https://", "https://"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := BaseURL(tc.input); got != tc.expected {
				t.Errorf("BaseURL(%q) = %q; expected %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestMapToSessions_URLWithPath(t *testing.T) {
	item := Item{ID: "item1", Title: "Production", Vault: Vault{ID: "vault1", Name: "Employee"}}
	urls := []URLSource{
		{URL: "https://prod.example.com/apps/cockpit/index.html#/", Label: "cockpit", Primary: true, Source: "urls"},
		{URL: "https://prod.example.com/apps/devicemanagement/", Label: "devicemanagement", Source: "urls"},
		{URL: "https://test.example.com/apps/cockpit", Label: "test", Source: "urls"},
	}

	sessions := MapToSessions(item, ItemFields{Username: "admin"}, urls, "Employee", nil)

	// URLs of the same instance result in one session
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].Host != "https://prod.example.com" {
		t.Errorf("expected host https://prod.example.com, got %s", sessions[0].Host)
	}
	if sessions[1].Host != "https://test.example.com" {
		t.Errorf("expected host https://test.example.com, got %s", sessions[1].Host)
	}

	// One URL with path results in one session with the base URL
	sessions = MapToSessions(item, ItemFields{Username: "admin"}, urls[:1], "Employee", nil)
	if len(sessions) != 1 || sessions[0].Host != "https://prod.example.com" || sessions[0].Name != "Production" {
		t.Errorf("unexpected session: %+v", sessions[0])
	}
}
