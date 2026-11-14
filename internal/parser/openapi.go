package parser

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
	"gopkg.in/yaml.v3"
)

// OpenAPIParser parses OpenAPI 3.0.x and 3.1.x specifications.
type OpenAPIParser struct{}

func NewOpenAPIParser() *OpenAPIParser {
	return &OpenAPIParser{}
}

func (p *OpenAPIParser) Supports(input string) bool {
	trimmed := strings.TrimSpace(input)
	if strings.Contains(trimmed, "openapi:") || strings.Contains(trimmed, `"openapi":`) {
		return true
	}
	return false
}

type openAPIDoc struct {
	OpenAPI string                 `json:"openapi" yaml:"openapi"`
	Info    openAPIInfo            `json:"info" yaml:"info"`
	Servers []openAPIServer        `json:"servers" yaml:"servers"`
	Paths   map[string]openAPIPath `json:"paths" yaml:"paths"`
	Components struct {
		Schemas         map[string]openAPISchema         `json:"schemas" yaml:"schemas"`
		SecuritySchemes map[string]openAPISecurityScheme `json:"securitySchemes" yaml:"securitySchemes"`
	} `json:"components" yaml:"components"`
}

type openAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

type openAPIServer struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description" yaml:"description"`
}

type openAPIPath map[string]interface{}

type openAPIOperation struct {
	Summary     string                       `json:"summary" yaml:"summary"`
	Description string                       `json:"description" yaml:"description"`
	OperationID string                       `json:"operationId" yaml:"operationId"`
	Parameters  []openAPIParameter           `json:"parameters" yaml:"parameters"`
	RequestBody *openAPIRequestBody          `json:"requestBody" yaml:"requestBody"`
	Responses   map[string]interface{}       `json:"responses" yaml:"responses"`
	Tags        []string                     `json:"tags" yaml:"tags"`
	Security    []map[string][]string        `json:"security" yaml:"security"`
}

type openAPIParameter struct {
	Name        string         `json:"name" yaml:"name"`
	In          string         `json:"in" yaml:"in"` // path, query, header, cookie
	Description string         `json:"description" yaml:"description"`
	Required    bool           `json:"required" yaml:"required"`
	Schema      openAPISchema  `json:"schema" yaml:"schema"`
	Ref         string         `json:"$ref" yaml:"$ref"`
}

type openAPIRequestBody struct {
	Description string                        `json:"description" yaml:"description"`
	Required    bool                          `json:"required" yaml:"required"`
	Content     map[string]openAPIMediaType   `json:"content" yaml:"content"`
}

type openAPIMediaType struct {
	Schema openAPISchema `json:"schema" yaml:"schema"`
}

type openAPISchema struct {
	Type        string                   `json:"type" yaml:"type"`
	Format      string                   `json:"format" yaml:"format"`
	Description string                   `json:"description" yaml:"description"`
	Default     any                      `json:"default" yaml:"default"`
	Enum        []any                    `json:"enum" yaml:"enum"`
	Items       *openAPISchema           `json:"items" yaml:"items"`
	Properties  map[string]openAPISchema `json:"properties" yaml:"properties"`
	Required    []string                 `json:"required" yaml:"required"`
	Ref         string                   `json:"$ref" yaml:"$ref"`
}

type openAPISecurityScheme struct {
	Type         string `json:"type" yaml:"type"`
	Scheme       string `json:"scheme" yaml:"scheme"`
	BearerFormat string `json:"bearerFormat" yaml:"bearerFormat"`
	In           string `json:"in" yaml:"in"`
	Name         string `json:"name" yaml:"name"`
}

func (p *OpenAPIParser) Parse(input string) (*domain.Spec, error) {
	var doc openAPIDoc
	trimmed := strings.TrimSpace(input)

	var err error
	if strings.HasPrefix(trimmed, "{") {
		err = json.Unmarshal([]byte(trimmed), &doc)
	} else {
		err = yaml.Unmarshal([]byte(trimmed), &doc)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI document: %w", err)
	}

	spec := &domain.Spec{
		Title:       doc.Info.Title,
		Description: doc.Info.Description,
		Version:     doc.Info.Version,
		Tools:       make([]domain.ToolDefinition, 0),
	}

	for _, s := range doc.Servers {
		spec.Servers = append(spec.Servers, s.URL)
	}
	if len(spec.Servers) > 0 {
		spec.BaseURL = spec.Servers[0]
	}

	// Canonical resolution helper for $ref
	resolveSchema := func(s openAPISchema) openAPISchema {
		if s.Ref != "" {
			refName := strings.TrimPrefix(s.Ref, "#/components/schemas/")
			if resolved, ok := doc.Components.Schemas[refName]; ok {
				if s.Description != "" {
					resolved.Description = s.Description
				}
				return resolved
			}
		}
		return s
	}

	// Sort paths deterministically
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
		pathItem := doc.Paths[pathKey]
		// Each path item contains methods
		for methodKey, rawOp := range pathItem {
			methodLower := strings.ToLower(methodKey)
			if !validMethods[methodLower] {
				continue
			}

			opBytes, err := json.Marshal(rawOp)
			if err != nil {
				continue
			}
			var op openAPIOperation
			if err := json.Unmarshal(opBytes, &op); err != nil {
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

			// Parse parameters
			for _, param := range op.Parameters {
				resolvedSchema := resolveSchema(param.Schema)
				typeStr := resolvedSchema.Type
				if typeStr == "" {
					typeStr = "string"
				}

				enums := make([]string, 0)
				for _, e := range resolvedSchema.Enum {
					enums = append(enums, fmt.Sprintf("%v", e))
				}

				itemsType := ""
				if resolvedSchema.Items != nil {
					itemsResolved := resolveSchema(*resolvedSchema.Items)
					itemsType = itemsResolved.Type
				}

				tool.Parameters = append(tool.Parameters, domain.ParameterDefinition{
					Name:        param.Name,
					In:          param.In,
					Description: param.Description,
					Required:    param.Required,
					Type:        typeStr,
					Format:      resolvedSchema.Format,
					Default:     resolvedSchema.Default,
					Enum:        enums,
					ItemsType:   itemsType,
				})
			}

			// Parse requestBody if present
			if op.RequestBody != nil {
				for contentType, media := range op.RequestBody.Content {
					resolvedSchema := resolveSchema(media.Schema)
					rbDef := &domain.RequestBodyDefinition{
						Description: op.RequestBody.Description,
						Required:    op.RequestBody.Required,
						ContentType: contentType,
						Properties:  make([]domain.ParameterDefinition, 0),
					}

					// Flatten object properties into parameters if object
					if resolvedSchema.Type == "object" || len(resolvedSchema.Properties) > 0 {
						reqSet := make(map[string]bool)
						for _, r := range resolvedSchema.Required {
							reqSet[r] = true
						}

						propNames := make([]string, 0, len(resolvedSchema.Properties))
						for k := range resolvedSchema.Properties {
							propNames = append(propNames, k)
						}
						sort.Strings(propNames)

						for _, propName := range propNames {
							propSchema := resolveSchema(resolvedSchema.Properties[propName])
							pType := propSchema.Type
							if pType == "" {
								pType = "string"
							}
							enums := make([]string, 0)
							for _, e := range propSchema.Enum {
								enums = append(enums, fmt.Sprintf("%v", e))
							}

							rbDef.Properties = append(rbDef.Properties, domain.ParameterDefinition{
								Name:        propName,
								In:          "body",
								Description: propSchema.Description,
								Required:    reqSet[propName],
								Type:        pType,
								Format:      propSchema.Format,
								Default:     propSchema.Default,
								Enum:        enums,
							})
						}
					}

					tool.RequestBody = rbDef
					break
				}
			}

			// Determine security requirement
			if len(doc.Components.SecuritySchemes) > 0 {
				for _, scheme := range doc.Components.SecuritySchemes {
					switch strings.ToLower(scheme.Type) {
					case "http":
						if strings.ToLower(scheme.Scheme) == "bearer" {
							tool.Security = domain.SecurityRequirement{
								AuthType:   "bearer",
								HeaderName: "Authorization",
								EnvVar:     "API_TOKEN",
							}
						} else if strings.ToLower(scheme.Scheme) == "basic" {
							tool.Security = domain.SecurityRequirement{
								AuthType:   "basic",
								HeaderName: "Authorization",
								EnvVar:     "BASIC_AUTH",
							}
						}
					case "apikey":
						tool.Security = domain.SecurityRequirement{
							AuthType:   "apikey",
							HeaderName: scheme.Name,
							EnvVar:     "API_KEY",
						}
					}
					break
				}
			}

			spec.Tools = append(spec.Tools, tool)
		}
	}

	return spec, nil
}
