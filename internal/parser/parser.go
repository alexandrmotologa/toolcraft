package parser

import (
	"fmt"
	"os"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

// Parser defines the common contract for all specification parsers.
type Parser interface {
	Parse(input string) (*domain.Spec, error)
	Supports(input string) bool
}

// DetectAndParse automatically identifies the specification format and converts it to a domain.Spec.
func DetectAndParse(pathOrContent string) (*domain.Spec, error) {
	content := pathOrContent
	if fileInfo, err := os.Stat(pathOrContent); err == nil && !fileInfo.IsDir() {
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
