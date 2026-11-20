package cmd

import "testing"

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
