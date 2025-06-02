package core

import (
	"testing"
)

func TestSessionTagsSimple(t *testing.T) {
	// Test session with only matching tags
	session := CumulocitySession{
		Name:       "Test Session",
		Host:       "https://test.example.com",
		Username:   "testuser",
		Tags:       []string{"c8y"}, // Only matching tags, not all item tags
		SessionURI: "op://vault/item",
	}

	description := session.Description()
	expected := "Username=testuser, Tags=c8y | op://vault/item"

	if description != expected {
		t.Errorf("Expected description %q, got %q", expected, description)
	}
}

func TestSessionMultipleMatchingTags(t *testing.T) {
	// Test session with multiple matching tags
	session := CumulocitySession{
		Name:       "Test Session",
		Host:       "https://test.example.com",
		Username:   "testuser",
		Tenant:     "testtenant",
		Tags:       []string{"c8y", "prod"}, // Only the requested/matching tags
		SessionURI: "op://vault/item",
	}

	description := session.Description()
	expected := "Username=testuser, Tenant=testtenant, Tags=c8y,prod | op://vault/item"

	if description != expected {
		t.Errorf("Expected description %q, got %q", expected, description)
	}
}

func TestSessionNoTags(t *testing.T) {
	// Test session with no tags
	session := CumulocitySession{
		Name:       "Test Session",
		Host:       "https://test.example.com",
		Username:   "testuser",
		Tags:       []string{}, // No tags
		SessionURI: "op://vault/item",
	}

	description := session.Description()
	expected := "Username=testuser | op://vault/item"

	if description != expected {
		t.Errorf("Expected description %q, got %q", expected, description)
	}
}
