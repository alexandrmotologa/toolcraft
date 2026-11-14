package generator_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/generator"
	"github.com/alexandrmotologa/toolcraft/internal/hardener"
	"github.com/alexandrmotologa/toolcraft/internal/parser"
)

func TestGenerateGoServer(t *testing.T) {
	specPath := filepath.Join("..", "..", "examples", "petstore_openapi3.yaml")
	spec, err := parser.DetectAndParse(specPath)
	if err != nil {
		t.Fatalf("failed to parse spec: %v", err)
	}

	hardened, _, err := hardener.Harden(spec, config.HardeningConfig{
		ReadOnly:             false,
		MaxParametersPerTool: 6,
		EnrichDescriptions:   true,
	})
	if err != nil {
		t.Fatalf("failed to harden spec: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "toolcraft-go-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Output.Language = "go"
	cfg.Output.OutputDir = tmpDir
	cfg.Output.ServerName = "test-petstore-mcp"

	gen, err := generator.GetGenerator("go")
	if err != nil {
		t.Fatalf("failed to get Go generator: %v", err)
	}

	if err := gen.Generate(hardened, cfg); err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Verify generated files exist
	for _, f := range []string{"go.mod", "main.go", "README.md"} {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected generated file %s to exist: %v", f, err)
		}
	}

	// Verify Go syntax validity with go vet or go build check
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("go vet failed on generated Go server: %v\nOutput: %s", err, string(out))
	}
}

func TestGenerateTypeScriptServer(t *testing.T) {
	specPath := filepath.Join("..", "..", "examples", "petstore_openapi3.yaml")
	spec, err := parser.DetectAndParse(specPath)
	if err != nil {
		t.Fatalf("failed to parse spec: %v", err)
	}

	hardened, _, err := hardener.Harden(spec, config.HardeningConfig{
		ReadOnly:             true,
		MaxParametersPerTool: 6,
		EnrichDescriptions:   true,
	})
	if err != nil {
		t.Fatalf("failed to harden spec: %v", err)
	}

	tmpDir, err := os.MkdirTemp("", "toolcraft-ts-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.DefaultConfig()
	cfg.Output.Language = "typescript"
	cfg.Output.OutputDir = tmpDir
	cfg.Output.ServerName = "test-petstore-ts-mcp"
	cfg.Hardening.ReadOnly = true

	gen, err := generator.GetGenerator("typescript")
	if err != nil {
		t.Fatalf("failed to get TypeScript generator: %v", err)
	}

	if err := gen.Generate(hardened, cfg); err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Verify generated files exist
	for _, f := range []string{"package.json", "tsconfig.json", "README.md", filepath.Join("src", "index.ts")} {
		path := filepath.Join(tmpDir, f)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("expected generated file %s to exist: %v", f, err)
		}
	}
}
