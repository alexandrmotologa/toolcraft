package hardener

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

var injectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)ignore\s+(all\s+)?previous\s+instructions`),
	regexp.MustCompile(`(?i)system\s+prompt`),
	regexp.MustCompile(`(?i)disregard\s+prior`),
	regexp.MustCompile(`(?i)you\s+are\s+now`),
}

// EnhanceDescription sanitizes descriptions and enriches short descriptions with clear agent guidance.
func EnhanceDescription(tool *domain.ToolDefinition) string {
	rawDesc := strings.TrimSpace(tool.Description)
	if rawDesc == "" {
		rawDesc = strings.TrimSpace(tool.Summary)
	}

	// Sanitize prompt injection strings
	for _, p := range injectionPatterns {
		rawDesc = p.ReplaceAllString(rawDesc, "[filtered]")
	}

	// If description is sufficiently detailed and long, return cleaned version
	if len(rawDesc) >= 50 && !isGenericSummary(rawDesc) {
		return rawDesc
	}

	// Synthesize an actionable prompt instruction
	actionVerb := mapMethodToAction(tool.Method)
	resourceName := inferResourceName(tool.Path)

	var requiredFields []string
	for _, p := range tool.Parameters {
		if p.Required {
			requiredFields = append(requiredFields, p.Name)
		}
	}
	if tool.RequestBody != nil {
		for _, p := range tool.RequestBody.Properties {
			if p.Required {
				requiredFields = append(requiredFields, p.Name)
			}
		}
	}

	var builder strings.Builder
	if rawDesc != "" {
		builder.WriteString(rawDesc)
		if !strings.HasSuffix(rawDesc, ".") {
			builder.WriteString(".")
		}
		builder.WriteString(" ")
	} else {
		builder.WriteString(fmt.Sprintf("%s %s. ", actionVerb, resourceName))
	}

	builder.WriteString(fmt.Sprintf("Use this tool to execute %s on %s.", tool.Method, tool.Path))

	if len(requiredFields) > 0 {
		builder.WriteString(fmt.Sprintf(" Required inputs: %s.", strings.Join(requiredFields, ", ")))
	}

	if tool.ReadOnly {
		builder.WriteString(" Read-only operation without state side-effects.")
	}

	return strings.TrimSpace(builder.String())
}

func isGenericSummary(s string) bool {
	lower := strings.ToLower(s)
	generic := []string{"get items", "list", "fetch data", "create", "update", "delete"}
	for _, g := range generic {
		if lower == g {
			return true
		}
	}
	return false
}

func mapMethodToAction(method string) string {
	switch strings.ToUpper(method) {
	case "GET":
		return "Retrieve details for"
	case "POST":
		return "Create a new record for"
	case "PUT", "PATCH":
		return "Update existing details for"
	case "DELETE":
		return "Permanently remove"
	default:
		return "Interact with"
	}
}

func inferResourceName(path string) string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for i := len(segments) - 1; i >= 0; i-- {
		s := segments[i]
		if !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, ":") && s != "" {
			return s
		}
	}
	return "target resource"
}
