<p align="center">
  <img src="docs/images/logo.png?raw=true" alt="toolcraft Logo" width="130" style="border-radius: 24px;" />
</p>

<h1 align="center">toolcraft</h1>

<p align="center">
  <strong>Automated OpenAPI to Model Context Protocol (MCP) compiler, schema hardener, and runtime gateway written in Go.</strong>
</p>

<p align="center">
  <a href="https://github.com/alexandrmotologa/toolcraft/actions/workflows/ci.yml"><img src="https://github.com/alexandrmotologa/toolcraft/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go" alt="Go Version" /></a>
  <a href="https://modelcontextprotocol.io"><img src="https://img.shields.io/badge/MCP-2024--11--05-7D56F4?style=flat" alt="MCP Protocol" /></a>
  <a href="https://github.com/alexandrmotologa/toolcraft/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-Apache_2.0-blue.svg" alt="License" /></a>
  <img src="https://img.shields.io/badge/RAM_Footprint-%3C_15MB-brightgreen" alt="RAM Footprint" />
</p>

---

## Overview

toolcraft converts OpenAPI 3.0/3.1, Swagger 2.0, Postman Collections, and cURL snippets into production-ready Model Context Protocol (MCP) servers. It generates standalone single-binary Go servers, typed TypeScript packages with Zod validation, acts directly as an in-memory MCP gateway over stdio and Server-Sent Events (SSE), provides synthetic schema mock servers, and registers servers directly into AI agent clients.

<p align="center">
  <img src="docs/images/toolcraft_demo.gif?raw=true" alt="toolcraft Workflow Demo" width="900" style="border-radius: 12px; box-shadow: 0 16px 32px rgba(0,0,0,0.4);" />
</p>

---

## Live Screenshots & Execution Telemetry

### Terminal Execution & Hardening Telemetry
Authentic CLI session executing schema inspection, token optimization analysis, Go single-binary generation in read-only mode, and automated registration into Claude Desktop.

<p align="center">
  <img src="docs/images/toolcraft_terminal_run.png?raw=true" alt="Terminal Execution Session" width="900" style="border-radius: 12px; box-shadow: 0 16px 32px rgba(0,0,0,0.4);" />
</p>

### Interactive Terminal User Interface (TUI)
Keyboard-driven Bubbletea dashboard enabling endpoint inspection, parameter toggle (`Space`), live test invocation (`x`), and instant code generation (`g`/`t`).

<p align="center">
  <img src="docs/images/toolcraft_tui.png?raw=true" alt="Terminal UI Dashboard" width="900" style="border-radius: 12px; box-shadow: 0 16px 32px rgba(0,0,0,0.4);" />
</p>

### Multi-Device Responsive Tools Catalog
Web documentation generated via `toolcraft catalog --format html`, featuring client-side real-time search, safety badges, and full responsive layout adaptation across Desktop (1440px) and Mobile (390px).

<p align="center">
  <img src="docs/images/toolcraft_catalog_desktop.png?raw=true" alt="Web Catalog Desktop View" width="620" style="border-radius: 10px; margin-right: 12px; vertical-align: top;" />
  <img src="docs/images/toolcraft_catalog_mobile.png?raw=true" alt="Web Catalog Mobile View" width="235" style="border-radius: 10px; vertical-align: top;" />
</p>

---

## Key Capabilities

- Universal Specification Ingestion: Reads local files and fetches remote URLs (HTTP/HTTPS with custom headers) for OpenAPI 3.0.x, OpenAPI 3.1.x, Swagger 2.0, Postman Collection v2.1, and raw cURL snippets.
- Dual-Target Code Generation:
  - Go Target: Produces idiomatic Go source code compiling into a portable, zero-dependency executable with under 15MB RAM footprint.
  - TypeScript Target: Produces an ESM npm package using `@modelcontextprotocol/sdk` and `zod` schema validation.
- Automated Schema Hardening:
  - Redundant parameter pruning: Strips internal trace IDs, debug flags, and proxy headers using regex rules.
  - Parameter capping: Constrains optional parameters per tool to reduce LLM prompt token consumption.
  - Description enhancement: Synthesizes actionable instructions and usage guidance for tool selection.
- Security Sandbox & Egress Guardrails:
  - Read-Only Mode: Disables mutating HTTP verbs (POST, PUT, PATCH, DELETE) to protect upstream APIs.
  - Credential Shield: Injects API keys and Bearer tokens from environment variables; credentials are never exposed to LLM contexts.
  - PII & Secret Redaction: Automatically sanitizes sensitive keys (password, token, credit card, JWT) in outbound responses.
  - Response Capping: Automatically summarizes oversized array payloads to protect LLM context windows.
  - Rate Limiting: Token-bucket rate limiter enforcing requests-per-minute thresholds.
- Direct Runtime Gateway:
  - Exposes any API specification as an active MCP server immediately via `toolcraft serve` without generating source files or compiling code.
- Offline Mock Server:
  - Generates schema-compliant synthetic JSON data via `toolcraft mock` with optional direct stdio bridging for offline agent evaluations.
- Client Auto-Configurator:
  - Generates configuration blocks or injects entries directly into Claude Desktop, Cursor, and Antigravity manifests (`toolcraft install`).
- Export & Documentation:
  - Purifies and exports hardened OpenAPI 3.1 files (`toolcraft sanitize`).
  - Generates interactive Markdown or HTML catalogs with live search (`toolcraft catalog`).

---

## Installation

### Prebuilt Binary

Download the latest binary for your operating system from the Releases page:

```bash
# macOS / Linux
curl -fsSL https://github.com/alexandrmotologa/toolcraft/releases/latest/download/toolcraft_linux_amd64 -o /usr/local/bin/toolcraft
chmod +x /usr/local/bin/toolcraft

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/alexandrmotologa/toolcraft/releases/latest/download/toolcraft_windows_amd64.exe -OutFile toolcraft.exe
```

### From Source

Requires Go 1.22 or later.

```bash
git clone https://github.com/alexandrmotologa/toolcraft.git
cd toolcraft
go install ./cmd/toolcraft
```

---

## Quick Start Workflows

### 1. Ingest Remote Specs and Inspect Token Savings

```bash
toolcraft inspect --spec https://api.petstore.example.com/v1/openapi.json
```

Output includes operation counts, original vs hardened parameters, and token reduction percentage.

### 2. Auto-Install MCP Server into Claude Desktop or Cursor

Generate configuration snippet:

```bash
toolcraft config --client claude --server-name petstore --spec ./examples/petstore_openapi3.yaml
```

Directly inject into the client's configuration file (with automatic backup):

```bash
toolcraft install --client claude --server-name petstore --spec ./examples/petstore_openapi3.yaml
```

### 3. Generate Standalone Go or TypeScript MCP Servers

Generate Go:

```bash
toolcraft generate \
  --spec ./examples/petstore_openapi3.yaml \
  --lang go \
  --out ./dist/petstore-mcp \
  --read-only
```

Generate TypeScript:

```bash
toolcraft generate \
  --spec ./examples/petstore_openapi3.yaml \
  --lang typescript \
  --out ./dist/petstore-mcp-ts
```

### 4. Run an In-Memory Mock Server for Offline Agent Testing

Start standalone HTTP mock server:

```bash
toolcraft mock --spec ./examples/petstore_openapi3.yaml --port 8085
```

Bridge MCP stdio directly to the mock server (zero external network required):

```bash
toolcraft mock --spec ./examples/petstore_openapi3.yaml --bridge-mcp
```

### 5. Export Hardened OpenAPI 3.1 Spec & Catalogs

Export purified OpenAPI 3.1:

```bash
toolcraft sanitize --spec ./examples/petstore_openapi3.yaml --out ./dist/hardened_api.yaml
```

Generate interactive HTML catalog with instant search:

```bash
toolcraft catalog --spec ./examples/petstore_openapi3.yaml --format html --out ./dist/catalog.html
```

---

## Configuration File (`toolcraft.yaml`)

```yaml
version: "1"
input:
  spec_path: "./specs/billing_api.yaml"
  spec_type: "openapi_v3"

output:
  language: "go" # "go" or "typescript"
  output_dir: "./dist/billing-mcp"
  server_name: "billing-mcp-server"
  version: "1.0.0"

hardening:
  read_only: true
  max_parameters_per_tool: 6
  prune_parameters:
    - ".*_trace_id"
    - "internal_.*"
    - "legacy_.*"
    - "X-.*"
  enrich_descriptions: true

security:
  auth_type: "bearer" # "bearer", "apikey", "basic", "none"
  token_env_var: "BILLING_API_TOKEN"
  api_key_header: "X-API-Key"
  rate_limit_rpm: 120
  timeout_seconds: 30
  redact_pii: true
  max_array_items: 10
  max_response_bytes: 51200
```

---

## CLI Reference

| Command | Description |
|---|---|
| `toolcraft generate` | Synthesizes an MCP server into Go or TypeScript code. |
| `toolcraft serve` | Starts an immediate in-memory MCP gateway (stdio or SSE). |
| `toolcraft mock` | Runs an in-memory schema mock server with optional MCP bridge. |
| `toolcraft config` | Prints ready-to-copy client configuration JSON. |
| `toolcraft install` | Automatically registers MCP server in client configuration. |
| `toolcraft sanitize` | Exports a pruned, LLM-optimized OpenAPI 3.1 specification. |
| `toolcraft catalog` | Generates a Markdown or HTML tool documentation catalog. |
| `toolcraft inspect` | Displays parameter reduction and token savings metrics. |
| `toolcraft init` | Initializes a baseline `toolcraft.yaml` configuration file. |
| `toolcraft tui` | Starts the interactive Terminal User Interface. |
| `toolcraft version` | Prints binary version, git commit, and build date. |

Full documentation is available in [docs/cli-reference.md](docs/cli-reference.md) and [docs/architecture.md](docs/architecture.md).

---

## Verification and Testing

Run test suites across all modules:

```bash
make test
```

Run test suite with race detector and coverage analysis:

```bash
make test-coverage
```

---

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.
