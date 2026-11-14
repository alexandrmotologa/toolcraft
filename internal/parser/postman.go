package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

// PostmanParser parses Postman Collection v2.1 documents.
type PostmanParser struct{}

func NewPostmanParser() *PostmanParser {
	return &PostmanParser{}
}

func (p *PostmanParser) Supports(input string) bool {
	trimmed := strings.TrimSpace(input)
	if strings.Contains(trimmed, "_postman_id") || strings.Contains(trimmed, "schema.getpostman.com") {
		return true
	}
	return false
}

type postmanCollection struct {
	Info postmanInfo  `json:"info"`
	Item []postmanItem `json:"item"`
}

type postmanInfo struct {
	ID          string `json:"_postman_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Schema      string `json:"schema"`
}

type postmanItem struct {
	Name    string          `json:"name"`
	Item    []postmanItem   `json:"item,omitempty"` // nested folders
	Request *postmanRequest `json:"request,omitempty"`
}

type postmanRequest struct {
	Method      string          `json:"method"`
	Header      []postmanHeader `json:"header"`
	URL         postmanURL      `json:"url"`
	Description string          `json:"description"`
}

type postmanHeader struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type postmanURL struct {
	Raw      string            `json:"raw"`
	Protocol string            `json:"protocol"`
	Host     []string          `json:"host"`
	Path     []string          `json:"path"`
	Query    []postmanQuery    `json:"query"`
	Variable []postmanVariable `json:"variable"`
}

type postmanQuery struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type postmanVariable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

func (p *PostmanParser) Parse(input string) (*domain.Spec, error) {
	var coll postmanCollection
	if err := json.Unmarshal([]byte(input), &coll); err != nil {
		return nil, fmt.Errorf("failed to parse Postman collection: %w", err)
	}

	spec := &domain.Spec{
		Title:       coll.Info.Name,
		Description: coll.Info.Description,
		Version:     "1.0.0",
		Tools:       make([]domain.ToolDefinition, 0),
	}

	var extractItems func(items []postmanItem, folderTags []string)
	extractItems = func(items []postmanItem, folderTags []string) {
		for _, item := range items {
			if len(item.Item) > 0 {
				newTags := append(folderTags, item.Name)
				extractItems(item.Item, newTags)
				continue
			}

			if item.Request == nil {
				continue
			}

			req := item.Request
			method := strings.ToUpper(req.Method)
			if method == "" {
				method = "GET"
			}

			pathParts := make([]string, 0)
			for _, part := range req.URL.Path {
				if strings.HasPrefix(part, ":") {
					pathParts = append(pathParts, "{"+strings.TrimPrefix(part, ":")+"}")
				} else {
					pathParts = append(pathParts, part)
				}
			}
			pathStr := "/" + strings.Join(pathParts, "/")

			toolID := domain.SanitizeToolID(item.Name, method, pathStr)
			isReadOnly := method == "GET" || method == "HEAD" || method == "OPTIONS"

			tool := domain.ToolDefinition{
				ID:          toolID,
				OriginalID:  item.Name,
				Method:      method,
				Path:        pathStr,
				Summary:     item.Name,
				Description: req.Description,
				Parameters:  make([]domain.ParameterDefinition, 0),
				Responses:   make(map[string]string),
				ReadOnly:    isReadOnly,
				Tags:        folderTags,
				Enabled:     true,
			}

			// Path variables
			for _, v := range req.URL.Variable {
				tool.Parameters = append(tool.Parameters, domain.ParameterDefinition{
					Name:        v.Key,
					In:          "path",
					Description: v.Description,
					Required:    true,
					Type:        "string",
					Default:     v.Value,
				})
			}

			// Query parameters
			for _, q := range req.URL.Query {
				tool.Parameters = append(tool.Parameters, domain.ParameterDefinition{
					Name:        q.Key,
					In:          "query",
					Description: q.Description,
					Required:    false,
					Type:        "string",
					Default:     q.Value,
				})
			}

			// Headers
			for _, h := range req.Header {
				tool.Parameters = append(tool.Parameters, domain.ParameterDefinition{
					Name:        h.Key,
					In:          "header",
					Description: h.Description,
					Required:    false,
					Type:        "string",
					Default:     h.Value,
				})
			}

			spec.Tools = append(spec.Tools, tool)
		}
	}

	extractItems(coll.Item, nil)
	return spec, nil
}
