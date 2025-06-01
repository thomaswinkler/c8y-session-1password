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
			result := NormalizeURL(tc.input)
			if result != tc.expected {
				t.Errorf("NormalizeURL(%q) = %q; expected %q", tc.input, result, tc.expected)
			}
		})
	}
}
