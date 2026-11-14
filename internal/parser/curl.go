package parser

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

// CurlParser parses raw cURL command snippets.
type CurlParser struct{}

func NewCurlParser() *CurlParser {
	return &CurlParser{}
}

func (p *CurlParser) Supports(input string) bool {
	trimmed := strings.TrimSpace(input)
	return strings.HasPrefix(trimmed, "curl ") || strings.Contains(trimmed, "\ncurl ")
}

func (p *CurlParser) Parse(input string) (*domain.Spec, error) {
	// Normalize line continuations
	normalized := strings.ReplaceAll(input, "\\\r\n", " ")
	normalized = strings.ReplaceAll(normalized, "\\\n", " ")
	normalized = strings.ReplaceAll(normalized, "\\\r", " ")

	lines := strings.Split(normalized, "\n")
	commands := make([]string, 0)
	var currentCmd strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "curl ") {
			if currentCmd.Len() > 0 {
				commands = append(commands, currentCmd.String())
				currentCmd.Reset()
			}
			currentCmd.WriteString(trimmed)
		} else if currentCmd.Len() > 0 && trimmed != "" {
			currentCmd.WriteString(" " + trimmed)
		}
	}
	if currentCmd.Len() > 0 {
		commands = append(commands, currentCmd.String())
	}

	if len(commands) == 0 {
		return nil, fmt.Errorf("no cURL commands detected")
	}

	spec := &domain.Spec{
		Title:       "cURL Ingested API",
		Description: "API specification synthesized from cURL command snippets",
		Version:     "1.0.0",
		Tools:       make([]domain.ToolDefinition, 0),
	}

	for i, cmd := range commands {
		tool, err := parseSingleCurl(cmd, i+1)
		if err != nil {
			continue
		}
		spec.Tools = append(spec.Tools, *tool)
	}

	if len(spec.Tools) == 0 {
		return nil, fmt.Errorf("failed to parse any valid cURL commands")
	}

	return spec, nil
}

func parseSingleCurl(cmd string, index int) (*domain.ToolDefinition, error) {
	tokens := tokenizeCmd(cmd)
	method := "GET"
	hasData := false
	var rawURL string
	headers := make(map[string]string)
	var dataBody string

	for i := 0; i < len(tokens); i++ {
		t := tokens[i]
		switch t {
		case "-X", "--request":
			if i+1 < len(tokens) {
				method = strings.ToUpper(tokens[i+1])
				i++
			}
		case "-H", "--header":
			if i+1 < len(tokens) {
				headerVal := tokens[i+1]
				parts := strings.SplitN(headerVal, ":", 2)
				if len(parts) == 2 {
					headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
				}
				i++
			}
		case "-d", "--data", "--data-raw", "--data-binary":
			if i+1 < len(tokens) {
				hasData = true
				dataBody = tokens[i+1]
				i++
			}
		default:
			if strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://") {
				rawURL = t
			}
		}
	}

	if hasData && method == "GET" {
		method = "POST"
	}

	if rawURL == "" {
		return nil, fmt.Errorf("no URL found in cURL command")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %s: %w", rawURL, err)
	}

	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	toolID := domain.SanitizeToolID(fmt.Sprintf("curl_op_%d", index), method, path)
	isReadOnly := method == "GET" || method == "HEAD" || method == "OPTIONS"

	tool := &domain.ToolDefinition{
		ID:          toolID,
		OriginalID:  fmt.Sprintf("curl_cmd_%d", index),
		Method:      method,
		Path:        path,
		Summary:     fmt.Sprintf("%s %s", method, path),
		Description: fmt.Sprintf("Execute %s request to %s", method, path),
		Parameters:  make([]domain.ParameterDefinition, 0),
		Responses:   make(map[string]string),
		ReadOnly:    isReadOnly,
		Tags:        []string{"cURL"},
		Enabled:     true,
	}

	// Extract query parameters
	for qName, qVals := range parsedURL.Query() {
		val := ""
		if len(qVals) > 0 {
			val = qVals[0]
		}
		tool.Parameters = append(tool.Parameters, domain.ParameterDefinition{
			Name:        qName,
			In:          "query",
			Description: fmt.Sprintf("Query parameter %s", qName),
			Required:    false,
			Type:        "string",
			Default:     val,
		})
	}

	// Extract headers
	for hName, hVal := range headers {
		tool.Parameters = append(tool.Parameters, domain.ParameterDefinition{
			Name:        hName,
			In:          "header",
			Description: fmt.Sprintf("Header %s", hName),
			Required:    false,
			Type:        "string",
			Default:     hVal,
		})
	}

	// Request body if present
	if dataBody != "" {
		rb := &domain.RequestBodyDefinition{
			Description: "Request payload",
			Required:    true,
			ContentType: "application/json",
			Properties:  make([]domain.ParameterDefinition, 0),
		}

		var parsedJSON map[string]interface{}
		if err := json.Unmarshal([]byte(dataBody), &parsedJSON); err == nil {
			for k, v := range parsedJSON {
				vType := "string"
				switch v.(type) {
				case float64:
					vType = "number"
				case bool:
					vType = "boolean"
				case []interface{}:
					vType = "array"
				case map[string]interface{}:
					vType = "object"
				}

				rb.Properties = append(rb.Properties, domain.ParameterDefinition{
					Name:        k,
					In:          "body",
					Description: fmt.Sprintf("Field %s", k),
					Required:    true,
					Type:        vType,
					Default:     v,
				})
			}
		}
		tool.RequestBody = rb
	}

	return tool, nil
}

func tokenizeCmd(cmd string) []string {
	re := regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)'`)
	matches := re.FindAllStringSubmatch(cmd, -1)
	tokens := make([]string, 0, len(matches))
	for _, m := range matches {
		if m[1] != "" {
			tokens = append(tokens, m[1])
		} else if m[2] != "" {
			tokens = append(tokens, m[2])
		} else {
			tokens = append(tokens, m[0])
		}
	}
	return tokens
}
