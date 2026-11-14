package catalog_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandrmotologa/toolcraft/internal/catalog"
	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/hardener"
	"github.com/alexandrmotologa/toolcraft/internal/parser"
)

func TestExportCatalogMarkdownAndHTML(t *testing.T) {
	specPath := filepath.Join("..", "..", "examples", "petstore_openapi3.yaml")
	spec, err := parser.DetectAndParse(specPath)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	hardened, _, err := hardener.Harden(spec, config.DefaultConfig().Hardening)
	if err != nil {
		t.Fatalf("failed to harden: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "toolcraft-catalog-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mdPath := filepath.Join(tmpDir, "CATALOG.md")
	if err := catalog.ExportCatalog(hardened, "markdown", mdPath); err != nil {
		t.Fatalf("markdown export failed: %v", err)
	}

	rawMD, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("failed to read md: %v", err)
	}
	if !strings.Contains(string(rawMD), "MCP Tools Catalog") {
		t.Errorf("expected markdown header in output")
	}
	if !strings.Contains(string(rawMD), "list_pets") {
		t.Errorf("expected list_pets in markdown output")
	}

	htmlPath := filepath.Join(tmpDir, "catalog.html")
	if err := catalog.ExportCatalog(hardened, "html", htmlPath); err != nil {
		t.Fatalf("html export failed: %v", err)
	}

	rawHTML, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("failed to read html: %v", err)
	}
	if !strings.Contains(string(rawHTML), "<title>Petstore Service API - MCP Tools Catalog</title>") {
		t.Errorf("expected html title in output")
	}
}
