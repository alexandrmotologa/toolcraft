package parser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
	"gopkg.in/yaml.v3"
)

// SwaggerParser parses legacy Swagger 2.0 specifications.
type SwaggerParser struct{}

func NewSwaggerParser() *SwaggerParser {
	return &SwaggerParser{}
}

func (p *SwaggerParser) Supports(input string) bool {
	trimmed := strings.TrimSpace(input)
	if strings.Contains(trimmed, `"swagger": "2.0"`) || strings.Contains(trimmed, `swagger: "2.0"`) || strings.Contains(trimmed, `swagger: '2.0'`) {
		return true
	}
	return false
}

type swaggerDoc struct {
	Swagger  string                            `json:"swagger" yaml:"swagger"`
	Info     openAPIInfo                       `json:"info" yaml:"info"`
	Host     string                            `json:"host" yaml:"host"`
	BasePath string                            `json:"basePath" yaml:"basePath"`
	Schemes  []string                          `json:"schemes" yaml:"schemes"`
	Paths    map[string]map[string]swaggerOp   `json:"paths" yaml:"paths"`
	Defs     map[string]swaggerSchema          `json:"definitions" yaml:"definitions"`
}

type swaggerOp struct {
	Summary     string             `json:"summary" yaml:"summary"`
	Description string             `json:"description" yaml:"description"`
	OperationID string             `json:"operationId" yaml:"operationId"`
	Parameters  []swaggerParameter `json:"parameters" yaml:"parameters"`
	Responses   map[string]any     `json:"responses" yaml:"responses"`
	Tags        []string           `json:"tags" yaml:"tags"`
}

type swaggerParameter struct {
	Name        string        `json:"name" yaml:"name"`
	In          string        `json:"in" yaml:"in"` // query, path, header, formData, body
	Description string        `json:"description" yaml:"description"`
	Required    bool          `json:"required" yaml:"required"`
	Type        string        `json:"type" yaml:"type"`
	Format      string        `json:"format" yaml:"format"`
	Default     any           `json:"default" yaml:"default"`
	Enum        []any         `json:"enum" yaml:"enum"`
	Schema      swaggerSchema `json:"schema" yaml:"schema"`
}

type swaggerSchema struct {
	Type        string                   `json:"type" yaml:"type"`
	Description string                   `json:"description" yaml:"description"`
	Properties  map[string]swaggerSchema `json:"properties" yaml:"properties"`
	Required    []string                 `json:"required" yaml:"required"`
	Ref         string                   `json:"$ref" yaml:"$ref"`
}

func (p *SwaggerParser) Parse(input string) (*domain.Spec, error) {
	var doc swaggerDoc
	trimmed := strings.TrimSpace(input)

	var err error
	if strings.HasPrefix(trimmed, "{") {
		err = json.Unmarshal([]byte(trimmed), &doc)
	} else {
		err = yaml.Unmarshal([]byte(trimmed), &doc)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse Swagger 2.0 document: %w", err)
	}

	scheme := "https"
	if len(doc.Schemes) > 0 {
		scheme = doc.Schemes[0]
	}
	baseURL := ""
	if doc.Host != "" {
		baseURL = fmt.Sprintf("%s://%s%s", scheme, doc.Host, doc.BasePath)
	} else if doc.BasePath != "" {
		baseURL = doc.BasePath
	}

	spec := &domain.Spec{
		Title:       doc.Info.Title,
		Description: doc.Info.Description,
		Version:     doc.Info.Version,
		BaseURL:     baseURL,
		Servers:     []string{baseURL},
		Tools:       make([]domain.ToolDefinition, 0),
	}

	resolveDef := func(s swaggerSchema) swaggerSchema {
		if s.Ref != "" {
			refName := strings.TrimPrefix(s.Ref, "#/definitions/")
			if resolved, ok := doc.Defs[refName]; ok {
				return resolved
			}
		}
		return s
	}

	paths := make([]string, 0, len(doc.Paths))
	for p := range doc.Paths {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	validMethods := map[string]bool{
		"get":     true,
		"post":    true,
		"put":     true,
		"delete":  true,
		"patch":   true,
		"options": true,
		"head":    true,
	}

	for _, pathKey := range paths {
		methods := doc.Paths[pathKey]
		for methodKey, op := range methods {
			methodLower := strings.ToLower(methodKey)
			if !validMethods[methodLower] {
				continue
			}

			toolID := domain.SanitizeToolID(op.OperationID, strings.ToUpper(methodLower), pathKey)
			methodUpper := strings.ToUpper(methodLower)
			isReadOnly := methodUpper == "GET" || methodUpper == "HEAD" || methodUpper == "OPTIONS"

			tool := domain.ToolDefinition{
				ID:          toolID,
				OriginalID:  op.OperationID,
				Method:      methodUpper,
				Path:        pathKey,
				Summary:     op.Summary,
				Description: op.Description,
				Parameters:  make([]domain.ParameterDefinition, 0),
				Responses:   make(map[string]string),
				ReadOnly:    isReadOnly,
				Tags:        op.Tags,
				Enabled:     true,
			}

			for _, param := range op.Parameters {
				if param.In == "body" {
					resolved := resolveDef(param.Schema)
					rbDef := &domain.RequestBodyDefinition{
						Description: param.Description,
						Required:    param.Required,
						ContentType: "application/json",
						Properties:  make([]domain.ParameterDefinition, 0),
					}

					reqSet := make(map[string]bool)
					for _, r := range resolved.Required {
						reqSet[r] = true
					}

					propNames := make([]string, 0, len(resolved.Properties))
					for k := range resolved.Properties {
						propNames = append(propNames, k)
					}
					sort.Strings(propNames)

					for _, propName := range propNames {
						propSchema := resolveDef(resolved.Properties[propName])
						rbDef.Properties = append(rbDef.Properties, domain.ParameterDefinition{
							Name:        propName,
							In:          "body",
							Description: propSchema.Description,
							Required:    reqSet[propName],
							Type:        propSchema.Type,
						})
					}
					tool.RequestBody = rbDef
					continue
				}

				enums := make([]string, 0)
				for _, e := range param.Enum {
					enums = append(enums, fmt.Sprintf("%v", e))
				}

				pType := param.Type
				if pType == "" {
					pType = "string"
				}

				tool.Parameters = append(tool.Parameters, domain.ParameterDefinition{
					Name:        param.Name,
					In:          param.In,
					Description: param.Description,
					Required:    param.Required,
					Type:        pType,
					Format:      param.Format,
					Default:     param.Default,
					Enum:        enums,
				})
			}

			spec.Tools = append(spec.Tools, tool)
		}
	}

	return spec, nil
}
