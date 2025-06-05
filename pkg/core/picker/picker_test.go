package picker

import (
	"testing"

	"github.com/thomaswinkler/c8y-session-1password/pkg/core"
)

func TestBuildTitle(t *testing.T) {
	tests := []struct {
		name         string
		sessionCount int
		metadata     PickerMetadata
		expected     string
	}{
		{
			name:         "single vault and tag",
			sessionCount: 5,
			metadata: PickerMetadata{
				Vaults: []string{"Employee"},
				Tags:   []string{"c8y"},
			},
			expected: "Sessions (5) • Vault: Employee • Tag: c8y",
		},
		{
			name:         "multiple vaults and tags",
			sessionCount: 12,
			metadata: PickerMetadata{
				Vaults: []string{"Employee", "Shared"},
				Tags:   []string{"c8y", "prod"},
			},
			expected: "Sessions (12) • Vaults: Employee, Shared • Tags: c8y, prod",
		},
		{
			name:         "no vaults specified (all vaults)",
			sessionCount: 8,
			metadata: PickerMetadata{
				Vaults: []string{},
				Tags:   []string{"c8y"},
			},
			expected: "Sessions (8) • All Vaults • Tag: c8y",
		},
		{
			name:         "no vaults, multiple tags",
			sessionCount: 3,
			metadata: PickerMetadata{
				Vaults: []string{},
				Tags:   []string{"c8y", "test", "dev"},
			},
			expected: "Sessions (3) • All Vaults • Tags: c8y, test, dev",
		},
		{
			name:         "single vault, no tags",
			sessionCount: 1,
			metadata: PickerMetadata{
				Vaults: []string{"Personal"},
				Tags:   []string{},
			},
			expected: "Sessions (1) • Vault: Personal",
		},
		{
			name:         "no metadata",
			sessionCount: 0,
			metadata: PickerMetadata{
				Vaults: []string{},
				Tags:   []string{},
			},
			expected: "Sessions (0) • All Vaults",
		},
		{
			name:         "with filter only",
			sessionCount: 7,
			metadata: PickerMetadata{
				Vaults: []string{},
				Tags:   []string{},
				Filter: "production",
			},
			expected: "Sessions (7) • All Vaults • Filter: production",
		},
		{
			name:         "vault, tag, and filter",
			sessionCount: 4,
			metadata: PickerMetadata{
				Vaults: []string{"Employee"},
				Tags:   []string{"c8y"},
				Filter: "prod",
			},
			expected: "Sessions (4) • Vault: Employee • Tag: c8y • Filter: prod",
		},
		{
			name:         "multiple vaults, tags, and filter",
			sessionCount: 15,
			metadata: PickerMetadata{
				Vaults: []string{"Employee", "Shared", "Personal"},
				Tags:   []string{"c8y", "test", "staging"},
				Filter: "environment",
			},
			expected: "Sessions (15) • Vaults: Employee, Shared, Personal • Tags: c8y, test, staging • Filter: environment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildTitle(tt.sessionCount, tt.metadata)
			if result != tt.expected {
				t.Errorf("buildTitle() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestPickerMetadata(t *testing.T) {
	// Test basic picker metadata creation
	metadata := PickerMetadata{
		Vaults: []string{"Employee", "Shared"},
		Tags:   []string{"c8y", "production"},
		Filter: "environment",
	}

	if len(metadata.Vaults) != 2 {
		t.Errorf("Expected 2 vaults, got %d", len(metadata.Vaults))
	}

	if len(metadata.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(metadata.Tags))
	}

	if metadata.Vaults[0] != "Employee" {
		t.Errorf("Expected first vault to be 'Employee', got %s", metadata.Vaults[0])
	}

	if metadata.Tags[0] != "c8y" {
		t.Errorf("Expected first tag to be 'c8y', got %s", metadata.Tags[0])
	}

	if metadata.Filter != "environment" {
		t.Errorf("Expected filter to be 'environment', got %s", metadata.Filter)
	}
}

func TestNewModelWithMetadata(t *testing.T) {
	// Create test sessions
	sessions := []*core.CumulocitySession{
		{
			Host:      "https://test1.cumulocity.com",
			Username:  "user1",
			ItemName:  "Test Session 1",
			VaultName: "Employee",
		},
		{
			Host:      "https://test2.cumulocity.com",
			Username:  "user2",
			ItemName:  "Test Session 2",
			VaultName: "Shared",
		},
	}

	generator := randomItemGenerator{
		sessions: sessions,
	}

	metadata := PickerMetadata{
		Vaults: []string{"Employee", "Shared"},
		Tags:   []string{"c8y"},
		Filter: "test-env",
	}

	model := newModel(generator, metadata)

	// Verify the model was created correctly
	if len(model.list.Items()) != 2 {
		t.Errorf("Expected 2 items in list, got %d", len(model.list.Items()))
	}

	if model.metadata.Vaults[0] != "Employee" {
		t.Errorf("Expected first vault to be 'Employee', got %s", model.metadata.Vaults[0])
	}

	if model.metadata.Tags[0] != "c8y" {
		t.Errorf("Expected first tag to be 'c8y', got %s", model.metadata.Tags[0])
	}

	if model.metadata.Filter != "test-env" {
		t.Errorf("Expected filter to be 'test-env', got %s", model.metadata.Filter)
	}

	// Check that the title includes the metadata
	expectedTitle := "Sessions (2) • Vaults: Employee, Shared • Tag: c8y • Filter: test-env"
	if model.list.Title != expectedTitle {
		t.Errorf("Expected title %q, got %q", expectedTitle, model.list.Title)
	}
}

func TestPickerMetadataString(t *testing.T) {
	tests := []struct {
		name     string
		metadata PickerMetadata
		expected string
	}{
		{
			name: "all fields populated",
			metadata: PickerMetadata{
				Vaults: []string{"Employee", "Shared"},
				Tags:   []string{"c8y", "prod"},
				Filter: "environment",
			},
			expected: "Vaults: Employee, Shared\nTags: c8y, prod\nFilter: environment\n",
		},
		{
			name: "only vaults",
			metadata: PickerMetadata{
				Vaults: []string{"Personal"},
				Tags:   []string{},
				Filter: "",
			},
			expected: "Vaults: Personal\n",
		},
		{
			name: "only tags",
			metadata: PickerMetadata{
				Vaults: []string{},
				Tags:   []string{"c8y"},
				Filter: "",
			},
			expected: "Tags: c8y\n",
		},
		{
			name: "only filter",
			metadata: PickerMetadata{
				Vaults: []string{},
				Tags:   []string{},
				Filter: "production",
			},
			expected: "Filter: production\n",
		},
		{
			name: "vaults and filter, no tags",
			metadata: PickerMetadata{
				Vaults: []string{"Employee", "Personal"},
				Tags:   []string{},
				Filter: "staging",
			},
			expected: "Vaults: Employee, Personal\nFilter: staging\n",
		},
		{
			name: "empty metadata",
			metadata: PickerMetadata{
				Vaults: []string{},
				Tags:   []string{},
				Filter: "",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.metadata.String()
			if result != tt.expected {
				t.Errorf("PickerMetadata.String() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestPickerMetadataNoColor(t *testing.T) {
	tests := []struct {
		name     string
		metadata PickerMetadata
		expected bool
	}{
		{
			name: "no color disabled by default",
			metadata: PickerMetadata{
				Vaults: []string{"Employee"},
				Tags:   []string{"c8y"},
				Filter: "",
			},
			expected: false,
		},
		{
			name: "no color explicitly disabled",
			metadata: PickerMetadata{
				Vaults:  []string{"Employee"},
				Tags:    []string{"c8y"},
				Filter:  "",
				NoColor: false,
			},
			expected: false,
		},
		{
			name: "no color enabled",
			metadata: PickerMetadata{
				Vaults:  []string{"Employee"},
				Tags:    []string{"c8y"},
				Filter:  "",
				NoColor: true,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metadata.NoColor != tt.expected {
				t.Errorf("PickerMetadata.NoColor = %v, expected %v", tt.metadata.NoColor, tt.expected)
			}
		})
	}
}

func TestBuildTitleWithNoColorMetadata(t *testing.T) {
	// Test that buildTitle works correctly regardless of NoColor flag
	metadata := PickerMetadata{
		Vaults:  []string{"Employee"},
		Tags:    []string{"c8y"},
		Filter:  "production",
		NoColor: true,
	}

	result := buildTitle(5, metadata)
	expected := "Sessions (5) • Vault: Employee • Tag: c8y • Filter: production"

	if result != expected {
		t.Errorf("buildTitle() with NoColor = %q, expected %q", result, expected)
	}
}

func TestPickerMetadataColorFlags(t *testing.T) {
	tests := []struct {
		name        string
		metadata    PickerMetadata
		expectColor bool
	}{
		{
			name: "default - colors enabled",
			metadata: PickerMetadata{
				Vaults: []string{"Employee"},
				Tags:   []string{"c8y"},
			},
			expectColor: true,
		},
		{
			name: "no color disabled",
			metadata: PickerMetadata{
				Vaults:  []string{"Employee"},
				Tags:    []string{"c8y"},
				NoColor: true,
			},
			expectColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldHaveColor := !tt.metadata.NoColor
			if shouldHaveColor != tt.expectColor {
				t.Errorf("Expected color support %v, got %v for metadata: NoColor=%v",
					tt.expectColor, shouldHaveColor, tt.metadata.NoColor)
			}
		})
	}
}
