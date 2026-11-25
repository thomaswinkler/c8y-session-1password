package onepassword

import "testing"

func TestMapToSessions_Mode(t *testing.T) {
	newItem := func(fields ...OPField) *OPItem {
		return &OPItem{
			ID:       "item1",
			Title:    "Dev Tenant",
			Category: "LOGIN",
			Vault:    OPVault{ID: "vault1", Name: "Employee"},
			Fields:   append([]OPField{{ID: "username", Value: "admin"}}, fields...),
			URLs: []OPURL{
				{Primary: true, Href: "https://dev.example.com"},
				{Href: "https://dev2.example.com"},
			},
		}
	}

	tests := []struct {
		name   string
		fields []OPField
		mode   string
	}{
		{name: "without mode field"},
		{name: "mode field", fields: []OPField{{Label: "mode", Value: "dev"}}, mode: "dev"},
		{name: "mode field in other case", fields: []OPField{{Label: "Mode", Value: "QUAL"}}, mode: "qual"},
		{name: "invalid mode is ignored", fields: []OPField{{Label: "mode", Value: "delete-all"}}},
		{name: "first mode field", fields: []OPField{{Label: "mode", Value: "prod"}, {Label: "mode", Value: "dev"}}, mode: "prod"},
		{name: "other label containing mode", fields: []OPField{{Label: "session mode notes", Value: "dev"}}},
	}

	client := NewClient("")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := client.mapToSessions(newItem(tt.fields...), map[string]string{"vault1": "Employee"})
			if len(sessions) != 2 {
				t.Fatalf("expected 2 sessions, got %d", len(sessions))
			}
			for _, session := range sessions {
				if session.Mode != tt.mode {
					t.Errorf("expected mode %q for %s, got %q", tt.mode, session.Host, session.Mode)
				}
			}
		})
	}
}
