package security

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var (
	sensitiveKeyPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|auth|api_?key|private_?key|credit_?card|card_?number|ssn|cvv|cvc)`)
	jwtPattern          = regexp.MustCompile(`eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]+`)
	creditCardPattern   = regexp.MustCompile(`\b(?:\d{4}[ -]?){3}\d{4}\b`)
)

// Options configures the egress data sanitizer and redactor.
type RedactorOptions struct {
	RedactPII     bool
	MaxArrayItems int
	MaxBytes      int
}

// DefaultRedactorOptions returns baseline security settings.
func DefaultRedactorOptions() RedactorOptions {
	return RedactorOptions{
		RedactPII:     true,
		MaxArrayItems: 10,
		MaxBytes:      50 * 1024, // 50 KB
	}
}

// SanitizeResponseBody applies PII redaction and context window truncation to an API response.
func SanitizeResponseBody(raw []byte, opts RedactorOptions) ([]byte, error) {
	if len(raw) == 0 {
		return raw, nil
	}

	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		// Not JSON; apply string-level redactions and byte truncation
		text := string(raw)
		if opts.RedactPII {
			text = jwtPattern.ReplaceAllString(text, "[REDACTED_JWT]")
			text = creditCardPattern.ReplaceAllString(text, "[REDACTED_CARD]")
		}
		if opts.MaxBytes > 0 && len(text) > opts.MaxBytes {
			text = text[:opts.MaxBytes] + fmt.Sprintf("\n... [Truncated %d excess bytes to preserve LLM context window]", len(raw)-opts.MaxBytes)
		}
		return []byte(text), nil
	}

	// Recursively redact parsed JSON structure
	sanitized := redactRecursive(parsed, opts)

	out, err := json.Marshal(sanitized)
	if err != nil {
		return raw, err
	}

	if opts.MaxBytes > 0 && len(out) > opts.MaxBytes {
		truncated := string(out[:opts.MaxBytes])
		return []byte(truncated + fmt.Sprintf(`{"_toolcraft_notice":"Truncated %d excess bytes"}`, len(out)-opts.MaxBytes)), nil
	}

	return out, nil
}

func redactRecursive(val interface{}, opts RedactorOptions) interface{} {
	switch v := val.(type) {
	case map[string]interface{}:
		cleanMap := make(map[string]interface{}, len(v))
		for k, item := range v {
			if opts.RedactPII && sensitiveKeyPattern.MatchString(k) {
				cleanMap[k] = "[REDACTED]"
			} else {
				cleanMap[k] = redactRecursive(item, opts)
			}
		}
		return cleanMap

	case []interface{}:
		totalItems := len(v)
		limit := totalItems
		if opts.MaxArrayItems > 0 && limit > opts.MaxArrayItems {
			limit = opts.MaxArrayItems
		}

		cleanSlice := make([]interface{}, 0, limit+1)
		for i := 0; i < limit; i++ {
			cleanSlice = append(cleanSlice, redactRecursive(v[i], opts))
		}

		if totalItems > limit {
			cleanSlice = append(cleanSlice, map[string]interface{}{
				"_toolcraft_notice": fmt.Sprintf("Displaying %d of %d items. Instruct model to use filters or pagination parameters to narrow results.", limit, totalItems),
			})
		}
		return cleanSlice

	case string:
		if opts.RedactPII {
			s := jwtPattern.ReplaceAllString(v, "[REDACTED_JWT]")
			s = creditCardPattern.ReplaceAllString(s, "[REDACTED_CARD]")
			return s
		}
		return v

	default:
		return v
	}
}

// MaskSecret masks an API key or token for logging or display (e.g., "sk-ab...1234").
func MaskSecret(secret string) string {
	if len(secret) <= 6 {
		return strings.Repeat("*", len(secret))
	}
	prefix := secret[:3]
	suffix := secret[len(secret)-3:]
	return fmt.Sprintf("%s...%s", prefix, suffix)
}
