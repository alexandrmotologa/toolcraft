package hardener

import (
	"strings"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

// ValidateAndNormalizeType maps raw schema type names to canonical JSON Schema primitives.
func ValidateAndNormalizeType(rawType string) string {
	switch strings.ToLower(rawType) {
	case "integer", "int", "int32", "int64":
		return "integer"
	case "number", "float", "double", "decimal":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "array", "list":
		return "array"
	case "object", "map":
		return "object"
	default:
		return "string"
	}
}

// ValidateParameter verifies constraints and assigns safe defaults.
func ValidateParameter(param *domain.ParameterDefinition) {
	param.Type = ValidateAndNormalizeType(param.Type)
	if param.Type == "array" && param.ItemsType == "" {
		param.ItemsType = "string"
	}
}
