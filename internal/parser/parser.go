package parser

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

// Parser defines the common contract for all specification parsers.
type Parser interface {
	Parse(input string) (*domain.Spec, error)
	Supports(input string) bool
}

// DetectAndParse automatically identifies the specification format and converts it to a domain.Spec.
func DetectAndParse(pathOrContent string) (*domain.Spec, error) {
	return DetectAndParseWithHeaders(pathOrContent, nil)
}

// DetectAndParseWithHeaders fetches remote specs or reads local files and converts them to domain.Spec.
func DetectAndParseWithHeaders(pathOrContent string, customHeaders map[string]string) (*domain.Spec, error) {
	content := pathOrContent

	// Check if input is a remote URL
	if strings.HasPrefix(pathOrContent, "http://") || strings.HasPrefix(pathOrContent, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		req, err := http.NewRequest("GET", pathOrContent, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for remote spec %s: %w", pathOrContent, err)
		}
		req.Header.Set("User-Agent", "toolcraft-parser/1.0")
		req.Header.Set("Accept", "application/json, application/yaml, text/yaml, */*")
		for k, v := range customHeaders {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch remote spec from %s: %w", pathOrContent, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("remote spec request to %s returned status HTTP %d", pathOrContent, resp.StatusCode)
		}

		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read remote spec body from %s: %w", pathOrContent, err)
		}
		content = string(raw)
	} else if fileInfo, err := os.Stat(pathOrContent); err == nil && !fileInfo.IsDir() {
		raw, err := os.ReadFile(pathOrContent)
		if err != nil {
			return nil, fmt.Errorf("failed to read spec file %s: %w", pathOrContent, err)
		}
		content = string(raw)
	}

	parsers := []Parser{
		NewCurlParser(),
		NewPostmanParser(),
		NewSwaggerParser(),
		NewOpenAPIParser(),
	}

	for _, p := range parsers {
		if p.Supports(content) {
			return p.Parse(content)
		}
	}

	return nil, fmt.Errorf("unrecognized specification format; expected OpenAPI 3.x, Swagger 2.0, Postman Collection v2, or cURL")
}
