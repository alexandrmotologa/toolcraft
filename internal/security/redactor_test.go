package security_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alexandrmotologa/toolcraft/internal/security"
)

func TestSanitizeResponseBodyPIIRedaction(t *testing.T) {
	inputJSON := []byte(`{
		"id": "usr_100",
		"name": "Jane Doe",
		"password_hash": "argon2$secret$98124",
		"api_key": "sk_live_9988223311",
		"auth_token": "bearer-token-xyz",
		"profile": {
			"credit_card": "4532-1234-5678-9012",
			"notes": "Customer token is eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozS6q"
		}
	}`)

	opts := security.RedactorOptions{
		RedactPII:     true,
		MaxArrayItems: 10,
		MaxBytes:      10000,
	}

	sanitized, err := security.SanitizeResponseBody(inputJSON, opts)
	if err != nil {
		t.Fatalf("sanitization failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(sanitized, &parsed); err != nil {
		t.Fatalf("failed to parse sanitized json: %v", err)
	}

	if parsed["password_hash"] != "[REDACTED]" {
		t.Errorf("expected password_hash to be [REDACTED], got %v", parsed["password_hash"])
	}
	if parsed["api_key"] != "[REDACTED]" {
		t.Errorf("expected api_key to be [REDACTED], got %v", parsed["api_key"])
	}

	profile := parsed["profile"].(map[string]interface{})
	if profile["credit_card"] != "[REDACTED]" {
		t.Errorf("expected credit_card to be [REDACTED], got %v", profile["credit_card"])
	}

	notes := profile["notes"].(string)
	if strings.Contains(notes, "eyJhbGci") {
		t.Errorf("expected JWT token in notes string to be masked, got %s", notes)
	}
}

func TestSanitizeResponseBodyArrayTruncation(t *testing.T) {
	items := make([]map[string]string, 25)
	for i := 0; i < 25; i++ {
		items[i] = map[string]string{"id": "item"}
	}
	raw, _ := json.Marshal(items)

	opts := security.RedactorOptions{
		RedactPII:     false,
		MaxArrayItems: 5,
		MaxBytes:      10000,
	}

	sanitized, err := security.SanitizeResponseBody(raw, opts)
	if err != nil {
		t.Fatalf("sanitization failed: %v", err)
	}

	var res []map[string]interface{}
	if err := json.Unmarshal(sanitized, &res); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	// 5 items + 1 truncation notice
	if len(res) != 6 {
		t.Errorf("expected 6 elements (5 items + 1 notice), got %d", len(res))
	}

	notice, ok := res[5]["_toolcraft_notice"].(string)
	if !ok || !strings.Contains(notice, "Displaying 5 of 25") {
		t.Errorf("expected truncation notice in final item, got: %v", res[5])
	}
}

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"sk-123456789", "sk-...789"},
		{"short", "*****"},
		{"123456", "******"},
	}

	for _, tt := range tests {
		got := security.MaskSecret(tt.input)
		if got != tt.want {
			t.Errorf("MaskSecret(%q) = %q; want %q", tt.input, got, tt.want)
		}
	}
}
