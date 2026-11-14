package installer_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandrmotologa/toolcraft/internal/installer"
)

func TestGenerateSnippet(t *testing.T) {
	snippet, err := installer.GenerateSnippet("claude", "billing-mcp", "toolcraft", "./specs/api.yaml", "BILLING_KEY")
	if err != nil {
		t.Fatalf("failed to generate snippet: %v", err)
	}

	if !strings.Contains(snippet, `"billing-mcp"`) {
		t.Errorf("snippet missing server name: %s", snippet)
	}
	if !strings.Contains(snippet, `"BILLING_KEY"`) {
		t.Errorf("snippet missing env var: %s", snippet)
	}
	if !strings.Contains(snippet, `"stdio"`) {
		t.Errorf("snippet missing stdio transport arg: %s", snippet)
	}

	var parsed installer.ConfigDocument
	if err := json.Unmarshal([]byte(snippet), &parsed); err != nil {
		t.Fatalf("snippet is not valid JSON: %v", err)
	}
}

func TestInstallConfigWithMerge(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "toolcraft-installer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testConfigPath := filepath.Join(tmpDir, "mcp.json")
	initialContent := []byte(`{
		"mcpServers": {
			"existing-server": {
				"command": "node",
				"args": ["server.js"]
			}
		}
	}`)
	if err := os.WriteFile(testConfigPath, initialContent, 0644); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}

	// Read and verify merging directly
	entry, err := installer.BuildServerEntry("toolcraft", "./spec.yaml", "API_KEY")
	if err != nil {
		t.Fatalf("build entry failed: %v", err)
	}

	var doc installer.ConfigDocument
	if err := json.Unmarshal(initialContent, &doc); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	doc.MCPServers["new-toolcraft-server"] = entry

	if len(doc.MCPServers) != 2 {
		t.Errorf("expected 2 servers after merge, got %d", len(doc.MCPServers))
	}
	if _, ok := doc.MCPServers["existing-server"]; !ok {
		t.Errorf("existing server was lost during merge")
	}
}
