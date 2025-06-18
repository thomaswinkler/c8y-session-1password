package core

import (
	"testing"
)

func TestFilterSessions(t *testing.T) {
	sessions := []*CumulocitySession{
		{
			Name:       "Production Session",
			ItemName:   "Prod Environment",
			Host:       "https://prod.example.com",
			Username:   "admin",
			SessionURI: "op://vault/prod-item",
		},
		{
			Name:       "Test Session",
			ItemName:   "Testing Environment",
			Host:       "https://test.example.com",
			Username:   "testuser",
			SessionURI: "op://vault/test-item",
		},
		{
			Name:       "Development Session",
			ItemName:   "Dev Environment",
			Host:       "https://dev.example.com",
			Username:   "developer",
			SessionURI: "op://vault/dev-item",
		},
	}

	tests := []struct {
		name          string
		filter        string
		expectedCount int
		expectedNames []string
	}{
		{
			name:          "empty filter returns all sessions",
			filter:        "",
			expectedCount: 3,
			expectedNames: []string{"Production Session", "Test Session", "Development Session"},
		},
		{
			name:          "filter by session name",
			filter:        "prod",
			expectedCount: 1,
			expectedNames: []string{"Production Session"},
		},
		{
			name:          "filter by item name",
			filter:        "testing",
			expectedCount: 1,
			expectedNames: []string{"Test Session"},
		},
		{
			name:          "filter by host URL",
			filter:        "dev.example",
			expectedCount: 1,
			expectedNames: []string{"Development Session"},
		},
		{
			name:          "filter by username",
			filter:        "admin",
			expectedCount: 1,
			expectedNames: []string{"Production Session"},
		},
		{
			name:          "filter by username - partial match",
			filter:        "test",
			expectedCount: 1,
			expectedNames: []string{"Test Session"},
		},
		{
			name:          "filter by username - case insensitive",
			filter:        "DEVELOPER",
			expectedCount: 1,
			expectedNames: []string{"Development Session"},
		},
		{
			name:          "filter matches multiple fields",
			filter:        "test",
			expectedCount: 1, // Should match both host (test.example.com) and username (testuser)
			expectedNames: []string{"Test Session"},
		},
		{
			name:          "no matches",
			filter:        "nonexistent",
			expectedCount: 0,
			expectedNames: []string{},
		},
		{
			name:          "case insensitive matching",
			filter:        "PROD",
			expectedCount: 1,
			expectedNames: []string{"Production Session"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterSessions(sessions, tt.filter)

			if len(result) != tt.expectedCount {
				t.Errorf("FilterSessions() returned %d sessions, expected %d",
					len(result), tt.expectedCount)
			}

			// Check that the expected sessions are in the result
			for i, expectedName := range tt.expectedNames {
				if i >= len(result) {
					t.Errorf("Missing expected session: %s", expectedName)
					continue
				}
				if result[i].Name != expectedName {
					t.Errorf("Expected session name %s at index %d, got %s",
						expectedName, i, result[i].Name)
				}
			}
		})
	}
}

func TestFilterSessions_UsernameSpecific(t *testing.T) {
	// Specific test to ensure username filtering works correctly
	sessions := []*CumulocitySession{
		{
			Name:     "Session A",
			ItemName: "Item A",
			Host:     "https://a.example.com",
			Username: "john.doe",
		},
		{
			Name:     "Session B",
			ItemName: "Item B",
			Host:     "https://b.example.com",
			Username: "jane.smith",
		},
		{
			Name:     "Session C",
			ItemName: "Item C",
			Host:     "https://c.example.com",
			Username: "bob.johnson",
		},
	}

	// Test filtering by username parts
	tests := []struct {
		filter   string
		expected []string
	}{
		{"john.doe", []string{"Session A"}},          // exact username match
		{"jane", []string{"Session B"}},              // partial username match
		{"bob", []string{"Session C"}},               // partial username match
		{"doe", []string{"Session A"}},               // username suffix match
		{"smith", []string{"Session B"}},             // username suffix match
		{"johnson", []string{"Session C"}},           // username suffix match
		{"JANE", []string{"Session B"}},              // case insensitive
		{"john", []string{"Session A", "Session C"}}, // matches both "john.doe" and "bob.johnson"
		{"JOHN", []string{"Session A", "Session C"}}, // case insensitive, matches both
		{"nonexistent", []string{}},
	}

	for _, tt := range tests {
		t.Run("username_filter_"+tt.filter, func(t *testing.T) {
			result := FilterSessions(sessions, tt.filter)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d sessions for filter '%s', got %d",
					len(tt.expected), tt.filter, len(result))
				return
			}

			for i, expectedName := range tt.expected {
				if result[i].Name != expectedName {
					t.Errorf("Expected session '%s' at index %d, got '%s'",
						expectedName, i, result[i].Name)
				}
			}
		})
	}
}
