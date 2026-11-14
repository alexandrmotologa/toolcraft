package domain

import (
	"fmt"
	"strings"
	"unicode"
)

// Spec represents a normalized API specification containing tools and metadata.
type Spec struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Version     string           `json:"version"`
	BaseURL     string           `json:"baseUrl"`
	Servers     []string         `json:"servers"`
	Tools       []ToolDefinition `json:"tools"`
}

// ToolDefinition represents a single executable tool derived from an API endpoint.
type ToolDefinition struct {
	ID          string                 `json:"id"`
	OriginalID  string                 `json:"originalId"`
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	Parameters  []ParameterDefinition  `json:"parameters"`
	RequestBody *RequestBodyDefinition `json:"requestBody,omitempty"`
	Responses   map[string]string      `json:"responses"`
	Security    SecurityRequirement    `json:"security"`
	ReadOnly    bool                   `json:"readOnly"`
	Tags        []string               `json:"tags"`
	Enabled     bool                   `json:"enabled"`
}

// ParameterDefinition represents an input parameter for a tool.
type ParameterDefinition struct {
	Name          string        `json:"name"`
	In            string        `json:"in"` // "path", "query", "header", "cookie"
	Description   string        `json:"description"`
	Required      bool          `json:"required"`
	Type          string        `json:"type"` // "string", "integer", "number", "boolean", "array", "object"
	Format        string        `json:"format,omitempty"`
	Default       any           `json:"default,omitempty"`
	Enum          []string      `json:"enum,omitempty"`
	ItemsType     string        `json:"itemsType,omitempty"`
	Properties    []PropertyDef `json:"properties,omitempty"`
	DiscardReason string        `json:"discardReason,omitempty"`
}

// PropertyDef represents an object field property for nested parameters or body schemas.
type PropertyDef struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Enum        []string `json:"enum,omitempty"`
}

// RequestBodyDefinition represents an HTTP request payload definition.
type RequestBodyDefinition struct {
	Description string                 `json:"description"`
	Required    bool                   `json:"required"`
	ContentType string                 `json:"contentType"`
	Properties  []ParameterDefinition  `json:"properties,omitempty"`
	SchemaRaw   map[string]interface{} `json:"schemaRaw,omitempty"`
}

// SecurityRequirement details the authentication method needed for outbound calls.
type SecurityRequirement struct {
	AuthType   string `json:"authType"` // "none", "bearer", "apikey", "basic"
	HeaderName string `json:"headerName"`
	QueryName  string `json:"queryName"`
	EnvVar     string `json:"envVar"`
}

// HardeningReport records the optimization statistics produced by the hardener.
type HardeningReport struct {
	OriginalOperations      int     `json:"originalOperations"`
	HardenedOperations      int     `json:"hardenedOperations"`
	OriginalParameters      int     `json:"originalParameters"`
	HardenedParameters      int     `json:"hardenedParameters"`
	PrunedParametersCount   int     `json:"prunedParametersCount"`
	EstimatedRawTokens      int     `json:"estimatedRawTokens"`
	EstimatedHardenedTokens int     `json:"estimatedHardenedTokens"`
	ReductionPercent        float64 `json:"reductionPercent"`
}

// SanitizeToolID converts an HTTP method and path or operationId into a valid snake_case tool name.
func SanitizeToolID(operationID, method, path string) string {
	if operationID != "" {
		cleaned := toSnakeCase(operationID)
		if isValidIdentifier(cleaned) {
			return cleaned
		}
	}

	methodPrefix := strings.ToLower(method)
	cleanPath := strings.ReplaceAll(path, "/", "_")
	cleanPath = strings.ReplaceAll(cleanPath, "{", "")
	cleanPath = strings.ReplaceAll(cleanPath, "}", "")
	cleanPath = strings.ReplaceAll(cleanPath, "-", "_")

	candidate := fmt.Sprintf("%s_%s", methodPrefix, cleanPath)
	return toSnakeCase(candidate)
}

func toSnakeCase(s string) string {
	var builder strings.Builder
	var prevIsUnderscore bool

	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if unicode.IsUpper(r) {
				if i > 0 && !prevIsUnderscore && unicode.IsLower(runes[i-1]) {
					builder.WriteRune('_')
				}
				builder.WriteRune(unicode.ToLower(r))
			} else {
				builder.WriteRune(r)
			}
			prevIsUnderscore = false
		} else {
			if !prevIsUnderscore && builder.Len() > 0 {
				builder.WriteRune('_')
				prevIsUnderscore = true
			}
		}
	}

	res := strings.Trim(builder.String(), "_")
	for strings.Contains(res, "__") {
		res = strings.ReplaceAll(res, "__", "_")
	}
	return res
}

func isValidIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if i == 0 && (unicode.IsDigit(r) || r == '_') {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}
