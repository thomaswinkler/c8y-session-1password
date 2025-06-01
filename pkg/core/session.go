package core

import (
	"fmt"
	"strings"
)

// NormalizeURL removes protocol and trailing slash for better display and sorting
// Uses robust protocol parsing by splitting on "://" separator
func NormalizeURL(url string) string {
	if url == "" {
		return url
	}

	// Split on "://" to handle any protocol (http, https, ftp, etc.)
	parts := strings.Split(url, "://")
	var normalized string
	if len(parts) > 1 {
		// Use the part after the protocol
		normalized = parts[1]
	} else {
		// No protocol found, use the original URL
		normalized = url
	}

	// Remove trailing slash
	normalized = strings.TrimSuffix(normalized, "/")

	return normalized
}

type CumulocitySession struct {
	SessionURI string `json:"sessionUri,omitempty"`
	Name       string `json:"name,omitempty"`
	Host       string `json:"host,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Tenant     string `json:"tenant,omitempty"`
	TOTP       string `json:"totp,omitempty"`
	TOTPSecret string `json:"totpSecret,omitempty"`

	// 1Password specific
	ItemID    string   `json:"itemId,omitempty"`
	ItemName  string   `json:"itemName,omitempty"`
	VaultID   string   `json:"vaultId,omitempty"`
	VaultName string   `json:"vaultName,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

func (i CumulocitySession) FilterValue() string {
	return strings.Join([]string{i.SessionURI, i.Host, i.Username}, " ")
}

func (i CumulocitySession) Title() string {
	return NormalizeURL(i.Host)
}

func (i CumulocitySession) Description() string {
	fields := []string{
		"Username=%s",
	}
	args := []any{
		i.Username,
	}

	if i.Tenant != "" {
		fields = append(fields, ", Tenant=%s")
		args = append(args, i.Tenant)
	}

	if len(i.Tags) > 0 {
		fields = append(fields, ", Tags=%s")
		args = append(args, strings.Join(i.Tags, ","))
	}

	fields = append(fields, " | uri=%s")
	args = append(args, i.SessionURI)

	return fmt.Sprintf(strings.Join(fields, ""), args...)
}
