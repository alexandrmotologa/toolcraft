package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the toolcraft manifest configuration.
type Config struct {
	Version   string          `yaml:"version"`
	Input     InputConfig     `yaml:"input"`
	Output    OutputConfig    `yaml:"output"`
	Hardening HardeningConfig `yaml:"hardening"`
	Security  SecurityConfig  `yaml:"security"`
}

// InputConfig specifies input specification settings.
type InputConfig struct {
	SpecPath string `yaml:"spec_path"`
	SpecType string `yaml:"spec_type"` // "openapi_v3", "swagger_v2", "postman_v2", "curl", "auto"
}

// OutputConfig specifies code generation targets.
type OutputConfig struct {
	Language   string `yaml:"language"` // "go", "typescript"
	OutputDir  string `yaml:"output_dir"`
	ServerName string `yaml:"server_name"`
	Version    string `yaml:"version"`
}

// HardeningConfig specifies schema pruning and token optimization rules.
type HardeningConfig struct {
	ReadOnly             bool     `yaml:"read_only"`
	MaxParametersPerTool int      `yaml:"max_parameters_per_tool"`
	PruneParameters      []string `yaml:"prune_parameters"`
	EnrichDescriptions   bool     `yaml:"enrich_descriptions"`
}

// SecurityConfig defines authentication injection and rate limiting.
type SecurityConfig struct {
	AuthType       string `yaml:"auth_type"` // "none", "bearer", "apikey", "basic"
	TokenEnvVar    string `yaml:"token_env_var"`
	APIKeyHeader   string `yaml:"api_key_header"`
	APIKeyQuery    string `yaml:"api_key_query"`
	RateLimitRPM   int    `yaml:"rate_limit_rpm"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
}

// DefaultConfig returns a configuration struct populated with recommended defaults.
func DefaultConfig() *Config {
	return &Config{
		Version: "1",
		Input: InputConfig{
			SpecPath: "",
			SpecType: "auto",
		},
		Output: OutputConfig{
			Language:   "go",
			OutputDir:  "./dist/mcp-server",
			ServerName: "mcp-server",
			Version:    "1.0.0",
		},
		Hardening: HardeningConfig{
			ReadOnly:             false,
			MaxParametersPerTool: 6,
			PruneParameters: []string{
				".*_trace_id",
				".*_correlation_id",
				"^X-.*",
				"internal_.*",
				"legacy_.*",
				"debug_.*",
			},
			EnrichDescriptions: true,
		},
		Security: SecurityConfig{
			AuthType:       "none",
			RateLimitRPM:   120,
			TimeoutSeconds: 30,
		},
	}
}

// Load reads a YAML configuration file from disk, applying defaults for missing fields.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file: %w", err)
	}

	if cfg.Output.Language == "" {
		cfg.Output.Language = "go"
	}
	if cfg.Output.OutputDir == "" {
		cfg.Output.OutputDir = "./dist/mcp-server"
	}
	if cfg.Hardening.MaxParametersPerTool <= 0 {
		cfg.Hardening.MaxParametersPerTool = 6
	}
	if cfg.Security.TimeoutSeconds <= 0 {
		cfg.Security.TimeoutSeconds = 30
	}
	if cfg.Security.RateLimitRPM <= 0 {
		cfg.Security.RateLimitRPM = 120
	}

	return cfg, nil
}
