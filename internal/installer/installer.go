package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// SupportedClient identifies a host AI agent application.
type SupportedClient string

const (
	ClientClaude      SupportedClient = "claude"
	ClientCursor      SupportedClient = "cursor"
	ClientAntigravity SupportedClient = "antigravity"
)

// ServerEntry represents the standardized client configuration block for an MCP server.
type ServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// ConfigDocument represents the top-level MCP servers configuration JSON structure.
type ConfigDocument struct {
	MCPServers map[string]ServerEntry `json:"mcpServers"`
}

// GetClientConfigPath resolves the default filesystem path for a specific agent client.
func GetClientConfigPath(client string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	switch strings.ToLower(client) {
	case "claude", "claude-desktop":
		if runtime.GOOS == "windows" {
			appData := os.Getenv("APPDATA")
			if appData == "" {
				appData = filepath.Join(home, "AppData", "Roaming")
			}
			return filepath.Join(appData, "Claude", "claude_desktop_config.json"), nil
		} else if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
		} else {
			return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"), nil
		}

	case "cursor":
		return filepath.Join(".cursor", "mcp.json"), nil

	case "antigravity":
		return filepath.Join(home, ".gemini", "antigravity", "mcp_config.json"), nil

	default:
		return "", fmt.Errorf("unsupported client '%s'; supported: claude, cursor, antigravity", client)
	}
}

// BuildServerEntry constructs the server entry block for an MCP server.
func BuildServerEntry(binaryPath, specPath, envVar string) (ServerEntry, error) {
	absSpec, err := filepath.Abs(specPath)
	if err != nil {
		absSpec = specPath
	}

	execCmd := binaryPath
	if execCmd == "" {
		if exe, err := os.Executable(); err == nil {
			execCmd = exe
		} else {
			execCmd = "toolcraft"
		}
	}

	envMap := make(map[string]string)
	if envVar != "" {
		envMap[envVar] = fmt.Sprintf("${%s}", envVar)
	}

	return ServerEntry{
		Command: execCmd,
		Args:    []string{"serve", "--spec", absSpec, "--transport", "stdio"},
		Env:     envMap,
	}, nil
}

// GenerateSnippet returns the client-ready JSON configuration string.
func GenerateSnippet(client, serverName, binaryPath, specPath, envVar string) (string, error) {
	entry, err := BuildServerEntry(binaryPath, specPath, envVar)
	if err != nil {
		return "", err
	}

	doc := ConfigDocument{
		MCPServers: map[string]ServerEntry{
			serverName: entry,
		},
	}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize configuration snippet: %w", err)
	}
	return string(b), nil
}

// InstallConfig injects the server entry into the client's configuration file.
func InstallConfig(client, serverName, binaryPath, specPath, envVar string, force bool) (string, error) {
	targetPath, err := GetClientConfigPath(client)
	if err != nil {
		return "", err
	}

	entry, err := BuildServerEntry(binaryPath, specPath, envVar)
	if err != nil {
		return "", err
	}

	var doc ConfigDocument
	doc.MCPServers = make(map[string]ServerEntry)

	// If file exists, load and merge
	if raw, err := os.ReadFile(targetPath); err == nil {
		if err := json.Unmarshal(raw, &doc); err != nil {
			if !force {
				return "", fmt.Errorf("target configuration %s is invalid JSON; pass --force to overwrite", targetPath)
			}
			doc.MCPServers = make(map[string]ServerEntry)
		}
		if doc.MCPServers == nil {
			doc.MCPServers = make(map[string]ServerEntry)
		}

		// Create backup
		backupPath := targetPath + ".bak"
		_ = os.WriteFile(backupPath, raw, 0644)
	} else {
		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return "", fmt.Errorf("failed to create config directory %s: %w", filepath.Dir(targetPath), err)
		}
	}

	doc.MCPServers[serverName] = entry

	updatedJSON, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize updated configuration: %w", err)
	}

	if err := os.WriteFile(targetPath, updatedJSON, 0644); err != nil {
		return "", fmt.Errorf("failed to write configuration file %s: %w", targetPath, err)
	}

	return targetPath, nil
}
