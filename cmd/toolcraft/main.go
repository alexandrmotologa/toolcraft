package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/alexandrmotologa/toolcraft/internal/catalog"
	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/generator"
	"github.com/alexandrmotologa/toolcraft/internal/hardener"
	"github.com/alexandrmotologa/toolcraft/internal/installer"
	"github.com/alexandrmotologa/toolcraft/internal/mock"
	"github.com/alexandrmotologa/toolcraft/internal/parser"
	"github.com/alexandrmotologa/toolcraft/internal/runtime"
	"github.com/alexandrmotologa/toolcraft/internal/sanitizer"
	"github.com/alexandrmotologa/toolcraft/internal/tui"
	"github.com/alexandrmotologa/toolcraft/internal/version"
	"github.com/spf13/cobra"
)

var (
	cfgPath string
	verbose bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "toolcraft",
		Short: "Automated OpenAPI to Model Context Protocol (MCP) compiler and runtime gateway",
		Long: `toolcraft compiles OpenAPI 3.0/3.1, Swagger 2.0, Postman collections, and cURL
snippets into production-ready Model Context Protocol (MCP) servers in Go or TypeScript,
or acts as an immediate in-memory gateway with mock servers and client installers.`,
	}

	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "Path to toolcraft.yaml manifest")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	rootCmd.AddCommand(newGenerateCmd())
	rootCmd.AddCommand(newServeCmd())
	rootCmd.AddCommand(newInspectCmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newTUICmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newSanitizeCmd())
	rootCmd.AddCommand(newMockCmd())
	rootCmd.AddCommand(newCatalogCmd())
	rootCmd.AddCommand(newVersionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func loadEffectiveConfig(specFlag, langFlag, outFlag, serverNameFlag, authTypeFlag, authEnvFlag string, readOnlyFlag bool, maxParamsFlag, rateLimitFlag int) (*config.Config, error) {
	var cfg *config.Config
	if cfgPath != "" {
		loaded, err := config.Load(cfgPath)
		if err != nil {
			return nil, err
		}
		cfg = loaded
	} else {
		cfg = config.DefaultConfig()
	}

	if specFlag != "" {
		cfg.Input.SpecPath = specFlag
	}
	if langFlag != "" {
		cfg.Output.Language = langFlag
	}
	if outFlag != "" {
		cfg.Output.OutputDir = outFlag
	}
	if serverNameFlag != "" {
		cfg.Output.ServerName = serverNameFlag
	}
	if authTypeFlag != "" {
		cfg.Security.AuthType = authTypeFlag
	}
	if authEnvFlag != "" {
		cfg.Security.TokenEnvVar = authEnvFlag
	}
	if readOnlyFlag {
		cfg.Hardening.ReadOnly = true
	}
	if maxParamsFlag > 0 {
		cfg.Hardening.MaxParametersPerTool = maxParamsFlag
	}
	if rateLimitFlag > 0 {
		cfg.Security.RateLimitRPM = rateLimitFlag
	}

	if cfg.Input.SpecPath == "" {
		return nil, fmt.Errorf("no specification file provided; use --spec <path|url> or configure input.spec_path in toolcraft.yaml")
	}

	return cfg, nil
}

func newGenerateCmd() *cobra.Command {
	var (
		specFlag       string
		langFlag       string
		outFlag        string
		serverNameFlag string
		readOnlyFlag   bool
		authTypeFlag   string
		authEnvFlag    string
		rateLimitFlag  int
		maxParamsFlag  int
		headerFlag     []string
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Synthesize an MCP server from an API specification into Go or TypeScript code",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadEffectiveConfig(specFlag, langFlag, outFlag, serverNameFlag, authTypeFlag, authEnvFlag, readOnlyFlag, maxParamsFlag, rateLimitFlag)
			if err != nil {
				return err
			}

			headers := parseHeaderSlice(headerFlag)
			fmt.Fprintf(os.Stderr, "Ingesting specification from: %s\n", cfg.Input.SpecPath)
			spec, err := parser.DetectAndParseWithHeaders(cfg.Input.SpecPath, headers)
			if err != nil {
				return fmt.Errorf("parsing failed: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Hardening schemas (read-only=%v, max-params=%d)...\n", cfg.Hardening.ReadOnly, cfg.Hardening.MaxParametersPerTool)
			hardened, report, err := hardener.Harden(spec, cfg.Hardening)
			if err != nil {
				return fmt.Errorf("hardening failed: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Synthesizing %s MCP server into: %s\n", cfg.Output.Language, cfg.Output.OutputDir)
			gen, err := generator.GetGenerator(cfg.Output.Language)
			if err != nil {
				return err
			}

			if err := gen.Generate(hardened, cfg); err != nil {
				return fmt.Errorf("code synthesis failed: %w", err)
			}

			fmt.Printf("Generation complete.\n")
			fmt.Printf("  Tools Synthesized: %d\n", report.HardenedOperations)
			fmt.Printf("  Parameters Pruned: %d\n", report.PrunedParametersCount)
			fmt.Printf("  Token Reduction:   %.1f%%\n", report.ReductionPercent)
			fmt.Printf("  Output Location:   %s\n", cfg.Output.OutputDir)

			return nil
		},
	}

	cmd.Flags().StringVarP(&specFlag, "spec", "s", "", "Path or URL to API specification file")
	cmd.Flags().StringVarP(&langFlag, "lang", "l", "go", "Target programming language (go, typescript)")
	cmd.Flags().StringVarP(&outFlag, "out", "o", "./dist/mcp-server", "Output directory for synthesized code")
	cmd.Flags().StringVar(&serverNameFlag, "server-name", "mcp-server", "Server name identifier")
	cmd.Flags().BoolVar(&readOnlyFlag, "read-only", false, "Disallow mutating operations (POST, PUT, DELETE)")
	cmd.Flags().StringVar(&authTypeFlag, "auth-type", "none", "Authentication type (none, bearer, apikey, basic)")
	cmd.Flags().StringVar(&authEnvFlag, "auth-env", "", "Environment variable holding target API token")
	cmd.Flags().IntVar(&rateLimitFlag, "rate-limit", 120, "Rate limit in requests per minute")
	cmd.Flags().IntVar(&maxParamsFlag, "max-params", 6, "Maximum parameters retained per tool")
	cmd.Flags().StringSliceVar(&headerFlag, "spec-header", nil, "Custom headers for remote spec fetching (Name: Value)")

	return cmd
}

func newServeCmd() *cobra.Command {
	var (
		specFlag      string
		transportFlag string
		portFlag      int
		hostFlag      string
		readOnlyFlag  bool
		authEnvFlag   string
		rateLimitFlag int
		headerFlag    []string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start an in-memory MCP gateway exposing tools directly over stdio or SSE",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadEffectiveConfig(specFlag, "", "", "", "", authEnvFlag, readOnlyFlag, 0, rateLimitFlag)
			if err != nil {
				return err
			}

			headers := parseHeaderSlice(headerFlag)
			spec, err := parser.DetectAndParseWithHeaders(cfg.Input.SpecPath, headers)
			if err != nil {
				return fmt.Errorf("parsing failed: %w", err)
			}

			hardened, _, err := hardener.Harden(spec, cfg.Hardening)
			if err != nil {
				return fmt.Errorf("hardening failed: %w", err)
			}

			srv := runtime.NewServer(hardened, cfg)
			ctx := context.Background()

			switch transportFlag {
			case "stdio":
				return srv.ServeStdio(ctx, os.Stdin, os.Stdout)
			case "sse":
				return srv.ServeSSE(ctx, hostFlag, portFlag)
			default:
				return fmt.Errorf("unknown transport mode '%s'; supported: stdio, sse", transportFlag)
			}
		},
	}

	cmd.Flags().StringVarP(&specFlag, "spec", "s", "", "Path or URL to API specification file")
	cmd.Flags().StringVarP(&transportFlag, "transport", "t", "stdio", "MCP transport mode (stdio, sse)")
	cmd.Flags().IntVarP(&portFlag, "port", "p", 8080, "HTTP port for SSE transport")
	cmd.Flags().StringVar(&hostFlag, "host", "127.0.0.1", "Binding host for SSE transport")
	cmd.Flags().BoolVar(&readOnlyFlag, "read-only", false, "Disallow mutating operations")
	cmd.Flags().StringVar(&authEnvFlag, "auth-env", "", "Environment variable containing auth token")
	cmd.Flags().IntVar(&rateLimitFlag, "rate-limit", 120, "Requests per minute limit")
	cmd.Flags().StringSliceVar(&headerFlag, "spec-header", nil, "Custom headers for remote spec fetching")

	return cmd
}

func newInspectCmd() *cobra.Command {
	var (
		specFlag      string
		jsonOutput    bool
		maxParamsFlag int
		headerFlag    []string
	)

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Analyze an API specification and display token optimization metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			if specFlag == "" && cfgPath != "" {
				loaded, err := config.Load(cfgPath)
				if err == nil {
					specFlag = loaded.Input.SpecPath
				}
			}
			if specFlag == "" {
				return fmt.Errorf("missing specification path; pass --spec <path|url>")
			}

			headers := parseHeaderSlice(headerFlag)
			spec, err := parser.DetectAndParseWithHeaders(specFlag, headers)
			if err != nil {
				return fmt.Errorf("parsing failed: %w", err)
			}

			cfg := config.DefaultConfig().Hardening
			if maxParamsFlag > 0 {
				cfg.MaxParametersPerTool = maxParamsFlag
			}

			_, report, err := hardener.Harden(spec, cfg)
			if err != nil {
				return fmt.Errorf("hardening analysis failed: %w", err)
			}

			if jsonOutput {
				outBytes, _ := json.MarshalIndent(report, "", "  ")
				fmt.Println(string(outBytes))
				return nil
			}

			fmt.Printf("=== toolcraft Inspection Report ===\n")
			fmt.Printf("Specification Title:       %s\n", spec.Title)
			fmt.Printf("Specification Version:     %s\n", spec.Version)
			fmt.Printf("Total Endpoints:           %d\n", report.OriginalOperations)
			fmt.Printf("Available MCP Tools:       %d\n", report.HardenedOperations)
			fmt.Printf("Raw Parameter Count:       %d\n", report.OriginalParameters)
			fmt.Printf("Hardened Parameter Count:  %d\n", report.HardenedParameters)
			fmt.Printf("Pruned Redundant Params:   %d\n", report.PrunedParametersCount)
			fmt.Printf("Estimated Raw Context:     %d tokens\n", report.EstimatedRawTokens)
			fmt.Printf("Hardened Prompt Context:   %d tokens\n", report.EstimatedHardenedTokens)
			fmt.Printf("Token Reduction Savings:   %.1f%%\n", report.ReductionPercent)

			return nil
		},
	}

	cmd.Flags().StringVarP(&specFlag, "spec", "s", "", "Path or URL to API specification file")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output report as JSON")
	cmd.Flags().IntVar(&maxParamsFlag, "max-params", 6, "Maximum parameters retained per tool")
	cmd.Flags().StringSliceVar(&headerFlag, "spec-header", nil, "Custom headers for remote spec fetching")

	return cmd
}

func newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a baseline toolcraft.yaml configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "toolcraft.yaml"
			if _, err := os.Stat(target); err == nil && !force {
				return fmt.Errorf("toolcraft.yaml already exists; use --force to overwrite")
			}

			content := `version: "1"
input:
  spec_path: "./examples/petstore_openapi3.yaml"
  spec_type: "openapi_v3"

output:
  language: "go"
  output_dir: "./dist/mcp-server"
  server_name: "petstore-mcp"
  version: "1.0.0"

hardening:
  read_only: false
  max_parameters_per_tool: 6
  prune_parameters:
    - ".*_trace_id"
    - ".*_shard_.*"
    - "legacy_.*"
    - "X-.*"
  enrich_descriptions: true

security:
  auth_type: "none"
  rate_limit_rpm: 120
  timeout_seconds: 30
  redact_pii: true
  max_array_items: 10
  max_response_bytes: 51200
`
			if err := os.WriteFile(target, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", target, err)
			}

			fmt.Printf("Created baseline configuration at: %s\n", target)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing configuration file")
	return cmd
}

func newTUICmd() *cobra.Command {
	var specFlag string

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive Terminal User Interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadEffectiveConfig(specFlag, "", "", "", "", "", false, 0, 0)
			if err != nil {
				return err
			}

			spec, err := parser.DetectAndParse(cfg.Input.SpecPath)
			if err != nil {
				return fmt.Errorf("parsing failed: %w", err)
			}

			hardened, report, err := hardener.Harden(spec, cfg.Hardening)
			if err != nil {
				return fmt.Errorf("hardening failed: %w", err)
			}

			return tui.RunApp(hardened, report, cfg)
		},
	}

	cmd.Flags().StringVarP(&specFlag, "spec", "s", "", "Path to API specification file")
	return cmd
}

func newConfigCmd() *cobra.Command {
	var (
		clientFlag     string
		serverNameFlag string
		binaryFlag     string
		specFlag       string
		authEnvFlag    string
	)

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Display ready-to-copy client configuration snippet (Claude, Cursor, Antigravity)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if specFlag == "" && cfgPath != "" {
				loaded, err := config.Load(cfgPath)
				if err == nil {
					specFlag = loaded.Input.SpecPath
				}
			}
			if specFlag == "" {
				return fmt.Errorf("missing specification path; pass --spec <path>")
			}
			if serverNameFlag == "" {
				serverNameFlag = "toolcraft-mcp-server"
			}

			snippet, err := installer.GenerateSnippet(clientFlag, serverNameFlag, binaryFlag, specFlag, authEnvFlag)
			if err != nil {
				return err
			}

			fmt.Println(snippet)
			return nil
		},
	}

	cmd.Flags().StringVar(&clientFlag, "client", "claude", "Target client agent (claude, cursor, antigravity)")
	cmd.Flags().StringVar(&serverNameFlag, "server-name", "mcp-server", "MCP server identifier")
	cmd.Flags().StringVar(&binaryFlag, "binary", "", "Explicit binary command name or path")
	cmd.Flags().StringVarP(&specFlag, "spec", "s", "", "Path to API specification file")
	cmd.Flags().StringVar(&authEnvFlag, "auth-env", "", "Environment variable holding auth token")

	return cmd
}

func newInstallCmd() *cobra.Command {
	var (
		clientFlag     string
		serverNameFlag string
		binaryFlag     string
		specFlag       string
		authEnvFlag    string
		forceFlag      bool
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Inject an MCP server entry directly into an AI agent client's configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if specFlag == "" && cfgPath != "" {
				loaded, err := config.Load(cfgPath)
				if err == nil {
					specFlag = loaded.Input.SpecPath
				}
			}
			if specFlag == "" {
				return fmt.Errorf("missing specification path; pass --spec <path>")
			}
			if serverNameFlag == "" {
				serverNameFlag = "toolcraft-mcp-server"
			}

			targetFile, err := installer.InstallConfig(clientFlag, serverNameFlag, binaryFlag, specFlag, authEnvFlag, forceFlag)
			if err != nil {
				return fmt.Errorf("installation failed: %w", err)
			}

			fmt.Printf("Successfully registered '%s' into %s configuration:\n  %s\n", serverNameFlag, clientFlag, targetFile)
			return nil
		},
	}

	cmd.Flags().StringVar(&clientFlag, "client", "claude", "Target client agent (claude, cursor, antigravity)")
	cmd.Flags().StringVar(&serverNameFlag, "server-name", "mcp-server", "MCP server identifier")
	cmd.Flags().StringVar(&binaryFlag, "binary", "", "Explicit binary command name or path")
	cmd.Flags().StringVarP(&specFlag, "spec", "s", "", "Path to API specification file")
	cmd.Flags().StringVar(&authEnvFlag, "auth-env", "", "Environment variable holding auth token")
	cmd.Flags().BoolVarP(&forceFlag, "force", "f", false, "Overwrite corrupted client config file if present")

	return cmd
}

func newSanitizeCmd() *cobra.Command {
	var (
		specFlag      string
		outFlag       string
		maxParamsFlag int
		readOnlyFlag  bool
	)

	cmd := &cobra.Command{
		Use:   "sanitize",
		Short: "Export a pruned, LLM-optimized OpenAPI 3.1 specification file (YAML or JSON)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if specFlag == "" {
				return fmt.Errorf("missing specification path; pass --spec <path>")
			}
			if outFlag == "" {
				outFlag = "./hardened_api.yaml"
			}

			spec, err := parser.DetectAndParse(specFlag)
			if err != nil {
				return fmt.Errorf("parsing failed: %w", err)
			}

			cfg := config.DefaultConfig().Hardening
			if maxParamsFlag > 0 {
				cfg.MaxParametersPerTool = maxParamsFlag
			}
			if readOnlyFlag {
				cfg.ReadOnly = true
			}

			hardened, report, err := hardener.Harden(spec, cfg)
			if err != nil {
				return fmt.Errorf("hardening failed: %w", err)
			}

			if err := sanitizer.ExportSanitizedOpenAPI(hardened, outFlag); err != nil {
				return fmt.Errorf("sanitizer export failed: %w", err)
			}

			fmt.Printf("Successfully exported sanitized OpenAPI 3.1 spec:\n")
			fmt.Printf("  Destination:       %s\n", outFlag)
			fmt.Printf("  Tools Retained:    %d\n", report.HardenedOperations)
			fmt.Printf("  Parameters Pruned: %d\n", report.PrunedParametersCount)
			fmt.Printf("  Token Reduction:   %.1f%%\n", report.ReductionPercent)

			return nil
		},
	}

	cmd.Flags().StringVarP(&specFlag, "spec", "s", "", "Path to API specification file")
	cmd.Flags().StringVarP(&outFlag, "out", "o", "./hardened_api.yaml", "Destination file path (.yaml or .json)")
	cmd.Flags().IntVar(&maxParamsFlag, "max-params", 6, "Maximum parameters retained per tool")
	cmd.Flags().BoolVar(&readOnlyFlag, "read-only", false, "Disallow mutating operations")

	return cmd
}

func newMockCmd() *cobra.Command {
	var (
		specFlag   string
		portFlag   int
		hostFlag   string
		bridgeFlag bool
	)

	cmd := &cobra.Command{
		Use:   "mock",
		Short: "Run an in-memory mock API returning synthetic responses matching spec schemas",
		RunE: func(cmd *cobra.Command, args []string) error {
			if specFlag == "" {
				return fmt.Errorf("missing specification path; pass --spec <path>")
			}

			spec, err := parser.DetectAndParse(specFlag)
			if err != nil {
				return fmt.Errorf("parsing failed: %w", err)
			}

			mockServer := mock.NewMockServer(spec)
			baseURL, err := mockServer.Start(hostFlag, portFlag)
			if err != nil {
				return fmt.Errorf("mock server failed to start: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Mock API Server listening on: %s\n", baseURL)
			fmt.Fprintf(os.Stderr, "Endpoints available: %d\n", len(spec.Tools))

			if bridgeFlag {
				// Run MCP server in stdio mode pointing to this mock server
				spec.BaseURL = baseURL
				cfg := config.DefaultConfig()
				hardened, _, _ := hardener.Harden(spec, cfg.Hardening)
				mcpSrv := runtime.NewServer(hardened, cfg)
				fmt.Fprintf(os.Stderr, "MCP Gateway bridged over stdio connected to mock server\n")
				return mcpSrv.ServeStdio(context.Background(), os.Stdin, os.Stdout)
			}

			// Block until interrupt signal
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop mock server.\n")
			<-sigChan

			ctx, cancel := context.WithTimeout(context.Background(), 2)
			defer cancel()
			return mockServer.Stop(ctx)
		},
	}

	cmd.Flags().StringVarP(&specFlag, "spec", "s", "", "Path to API specification file")
	cmd.Flags().IntVarP(&portFlag, "port", "p", 8085, "Listening HTTP port")
	cmd.Flags().StringVar(&hostFlag, "host", "127.0.0.1", "Binding host interface")
	cmd.Flags().BoolVar(&bridgeFlag, "bridge-mcp", false, "Bridge MCP stdio transport to mock server")

	return cmd
}

func newCatalogCmd() *cobra.Command {
	var (
		specFlag   string
		formatFlag string
		outFlag    string
	)

	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Generate human and agent readable documentation catalog in Markdown or HTML",
		RunE: func(cmd *cobra.Command, args []string) error {
			if specFlag == "" {
				return fmt.Errorf("missing specification path; pass --spec <path>")
			}

			if outFlag == "" {
				if strings.ToLower(formatFlag) == "html" {
					outFlag = "./CATALOG.html"
				} else {
					outFlag = "./CATALOG.md"
				}
			}

			spec, err := parser.DetectAndParse(specFlag)
			if err != nil {
				return fmt.Errorf("parsing failed: %w", err)
			}

			cfg := config.DefaultConfig().Hardening
			hardened, _, err := hardener.Harden(spec, cfg)
			if err != nil {
				return fmt.Errorf("hardening failed: %w", err)
			}

			if err := catalog.ExportCatalog(hardened, formatFlag, outFlag); err != nil {
				return fmt.Errorf("catalog export failed: %w", err)
			}

			fmt.Printf("Successfully generated tool catalog (%s):\n  %s\n", formatFlag, outFlag)
			return nil
		},
	}

	cmd.Flags().StringVarP(&specFlag, "spec", "s", "", "Path to API specification file")
	cmd.Flags().StringVarP(&formatFlag, "format", "f", "markdown", "Catalog format (markdown, html)")
	cmd.Flags().StringVarP(&outFlag, "out", "o", "", "Destination file path")

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print toolcraft version and build information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("toolcraft v%s (commit: %s, built: %s)\n", version.Version, version.Commit, version.BuildDate)
		},
	}
}

func parseHeaderSlice(headers []string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	res := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			res[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return res
}
