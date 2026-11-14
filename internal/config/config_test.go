package config_test

import (
	"path/filepath"
	"testing"

	"github.com/alexandrmotologa/toolcraft/internal/config"
)

func TestLoadConfig(t *testing.T) {
	cfgPath := filepath.Join("..", "..", "toolcraft.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("failed to load toolcraft.yaml: %v", err)
	}

	if cfg.Input.SpecPath == "" {
		t.Errorf("expected spec_path to be populated")
	}

	if cfg.Output.Language != "go" {
		t.Errorf("expected language 'go', got '%s'", cfg.Output.Language)
	}

	if cfg.Hardening.MaxParametersPerTool != 6 {
		t.Errorf("expected max_parameters_per_tool 6, got %d", cfg.Hardening.MaxParametersPerTool)
	}
}
