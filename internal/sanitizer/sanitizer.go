package sanitizer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
	"gopkg.in/yaml.v3"
)

type openAPI31Doc struct {
	OpenAPI string                            `json:"openapi" yaml:"openapi"`
	Info    openAPI31Info                     `json:"info" yaml:"info"`
	Servers []openAPI31Server                 `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths   map[string]map[string]openAPI31Op `json:"paths" yaml:"paths"`
}

type openAPI31Info struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

type openAPI31Server struct {
	URL string `json:"url" yaml:"url"`
}

type openAPI31Op struct {
	Summary     string                  `json:"summary" yaml:"summary"`
	Description string                  `json:"description" yaml:"description"`
	OperationID string                  `json:"operationId" yaml:"operationId"`
	Parameters  []openAPI31Param        `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *openAPI31RequestBody   `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]openAPI31Res `json:"responses" yaml:"responses"`
	Tags        []string                `json:"tags,omitempty" yaml:"tags,omitempty"`
}

type openAPI31Param struct {
	Name        string         `json:"name" yaml:"name"`
	In          string         `json:"in" yaml:"in"`
	Description string         `json:"description" yaml:"description"`
	Required    bool           `json:"required" yaml:"required"`
	Schema      openAPI31Type  `json:"schema" yaml:"schema"`
}

type openAPI31Type struct {
	Type    string   `json:"type" yaml:"type"`
	Default any      `json:"default,omitempty" yaml:"default,omitempty"`
	Enum    []string `json:"enum,omitempty" yaml:"enum,omitempty"`
}

type openAPI31RequestBody struct {
	Required bool                           `json:"required" yaml:"required"`
	Content  map[string]openAPI31MediaType  `json:"content" yaml:"content"`
}

type openAPI31MediaType struct {
	Schema openAPI31SchemaObj `json:"schema" yaml:"schema"`
}

type openAPI31SchemaObj struct {
	Type       string                    `json:"type" yaml:"type"`
	Properties map[string]openAPI31Type  `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty" yaml:"required,omitempty"`
}

type openAPI31Res struct {
	Description string `json:"description" yaml:"description"`
}

// ExportSanitizedOpenAPI serializes a hardened domain.Spec into an optimized OpenAPI 3.1 file.
func ExportSanitizedOpenAPI(spec *domain.Spec, outputPath string) error {
	doc := openAPI31Doc{
		OpenAPI: "3.1.0",
		Info: openAPI31Info{
			Title:       spec.Title + " (Hardened for LLM)",
			Description: spec.Description,
			Version:     spec.Version,
		},
		Paths: make(map[string]map[string]openAPI31Op),
	}

	if spec.BaseURL != "" {
		doc.Servers = []openAPI31Server{{URL: spec.BaseURL}}
	}

	for _, tool := range spec.Tools {
		if doc.Paths[tool.Path] == nil {
			doc.Paths[tool.Path] = make(map[string]openAPI31Op)
		}

		op := openAPI31Op{
			Summary:     tool.Summary,
			Description: tool.Description,
			OperationID: tool.ID,
			Tags:        tool.Tags,
			Responses: map[string]openAPI31Res{
				"200": {Description: "Successful operation"},
			},
		}

		for _, p := range tool.Parameters {
			op.Parameters = append(op.Parameters, openAPI31Param{
				Name:        p.Name,
				In:          p.In,
				Description: p.Description,
				Required:    p.Required,
				Schema: openAPI31Type{
					Type:    p.Type,
					Default: p.Default,
					Enum:    p.Enum,
				},
			})
		}

		if tool.RequestBody != nil && len(tool.RequestBody.Properties) > 0 {
			props := make(map[string]openAPI31Type)
			var required []string

			for _, prop := range tool.RequestBody.Properties {
				props[prop.Name] = openAPI31Type{
					Type:    prop.Type,
					Default: prop.Default,
					Enum:    prop.Enum,
				}
				if prop.Required {
					required = append(required, prop.Name)
				}
			}

			op.RequestBody = &openAPI31RequestBody{
				Required: tool.RequestBody.Required,
				Content: map[string]openAPI31MediaType{
					"application/json": {
						Schema: openAPI31SchemaObj{
							Type:       "object",
							Properties: props,
							Required:   required,
						},
					},
				},
			}
		}

		methodKey := strings.ToLower(tool.Method)
		doc.Paths[tool.Path][methodKey] = op
	}

	var outputBytes []byte
	var err error
	ext := strings.ToLower(filepath.Ext(outputPath))
	if ext == ".json" {
		outputBytes, err = json.MarshalIndent(doc, "", "  ")
	} else {
		outputBytes, err = yaml.Marshal(doc)
	}

	if err != nil {
		return fmt.Errorf("failed to serialize OpenAPI document: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil && filepath.Dir(outputPath) != "." {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, outputBytes, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	return nil
}
