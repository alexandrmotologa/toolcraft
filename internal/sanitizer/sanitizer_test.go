package sanitizer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/hardener"
	"github.com/alexandrmotologa/toolcraft/internal/parser"
	"github.com/alexandrmotologa/toolcraft/internal/sanitizer"
)

func TestExportSanitizedOpenAPIYAMLAndJSON(t *testing.T) {
	specPath := filepath.Join("..", "..", "examples", "petstore_openapi3.yaml")
	spec, err := parser.DetectAndParse(specPath)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	hardened, _, err := hardener.Harden(spec, config.DefaultConfig().Hardening)
	if err != nil {
		t.Fatalf("failed to harden: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "toolcraft-sanitizer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	yamlOut := filepath.Join(tmpDir, "sanitized.yaml")
	if err := sanitizer.ExportSanitizedOpenAPI(hardened, yamlOut); err != nil {
		t.Fatalf("failed to export YAML: %v", err)
	}

	contentYAML, err := os.ReadFile(yamlOut)
	if err != nil {
		t.Fatalf("failed to read YAML output: %v", err)
	}
	if !strings.Contains(string(contentYAML), "openapi: 3.1.0") {
		t.Errorf("expected openapi: 3.1.0 in YAML output")
	}

	jsonOut := filepath.Join(tmpDir, "sanitized.json")
	if err := sanitizer.ExportSanitizedOpenAPI(hardened, jsonOut); err != nil {
		t.Fatalf("failed to export JSON: %v", err)
	}

	contentJSON, err := os.ReadFile(jsonOut)
	if err != nil {
		t.Fatalf("failed to read JSON output: %v", err)
	}
	if !strings.Contains(string(contentJSON), `"openapi": "3.1.0"`) {
		t.Errorf("expected \"openapi\": \"3.1.0\" in JSON output")
	}
}
