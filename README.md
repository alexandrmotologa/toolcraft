# toolcraft

Automated OpenAPI to Model Context Protocol (MCP) compiler, schema hardener, and runtime gateway written in Go.

toolcraft converts OpenAPI 3.0/3.1, Swagger 2.0, Postman Collections, and cURL snippets into production-ready Model Context Protocol (MCP) servers. It generates standalone single-binary Go servers, typed TypeScript packages with Zod validation, or acts directly as an in-memory MCP gateway over stdio and Server-Sent Events (SSE).

---

## Key Capabilities

- Universal Specification Support: Ingests OpenAPI 3.0.x, OpenAPI 3.1.x, Swagger 2.0, Postman Collection v2.1, and raw cURL command lines.
- Dual-Target Code Generation:
  - Go Target: Produces idiomatic, zero-dependency Go source code that compiles into a single binary (< 15MB RAM footprint).
  - TypeScript Target: Produces an ESM npm package using `@modelcontextprotocol/sdk` and `zod` schema validation.
- Automated Schema Hardening:
  - Redundant parameter pruning: Strips internal trace IDs, debug flags, and proxy headers using regex rules.
  - Parameter capping: Constrains optional parameters per tool to reduce LLM prompt token consumption.
  - Description enhancement: Synthesizes actionable instructions and usage guidance for tool selection.
- Security Sandbox:
  - Read-Only Mode: Disables mutating HTTP verbs (POST, PUT, PATCH, DELETE) to protect upstream APIs.
  - Credential Shield: Injects API keys and Bearer tokens from environment variables; credentials are never exposed to LLM contexts.
  - Rate Limiting: Token-bucket rate limiter enforcing requests-per-minute thresholds.
- Direct Runtime Gateway:
  - Exposes any API specification as an active MCP server immediately via `toolcraft serve` without generating source files or compiling code.
- Interactive Terminal UI:
  - Keyboard-driven terminal dashboard to inspect endpoints, review schema token reduction metrics, toggle active tools, and export configurations.

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

## Quick Start

### 1. Inspect an API Specification and View Hardening Metrics

Analyze an existing OpenAPI specification to verify token optimization metrics and pruned parameters:

```bash
toolcraft inspect --spec ./examples/petstore_openapi3.yaml
```

Output includes total operations, estimated raw prompt tokens, optimized prompt tokens, and parameter reduction percentage.

### 2. Generate a Standalone Go MCP Server

```bash
toolcraft generate \
  --spec ./examples/petstore_openapi3.yaml \
  --lang go \
  --out ./dist/petstore-mcp \
  --read-only \
  --auth-env PETSTORE_API_KEY
```

Build and test the generated Go server:

```bash
cd ./dist/petstore-mcp
go build -o petstore-mcp main.go
./petstore-mcp
```

### 3. Generate a TypeScript MCP Package

```bash
toolcraft generate \
  --spec ./examples/petstore_openapi3.yaml \
  --lang typescript \
  --out ./dist/petstore-mcp-ts \
  --read-only
```

Install dependencies and build the TypeScript package:

```bash
cd ./dist/petstore-mcp-ts
npm install
npm run build
```

### 4. Direct In-Memory MCP Gateway

Run an OpenAPI specification as an MCP server directly over stdio (compatible with Claude Code, Cursor, Antigravity):

```bash
toolcraft serve --spec ./examples/petstore_openapi3.yaml --transport stdio
```

Run as an SSE HTTP service:

```bash
toolcraft serve --spec ./examples/petstore_openapi3.yaml --transport sse --port 8080
```

### 5. Interactive Terminal UI

Launch the terminal interface to inspect paths, toggle tools, and configure generation parameters:

```bash
toolcraft tui --spec ./examples/petstore_openapi3.yaml
```

---

## Configuration File (`toolcraft.yaml`)

toolcraft can be configured using a declarative YAML manifest:

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
```

Generate using the configuration file:

```bash
toolcraft generate --config toolcraft.yaml
```

---

## Architecture Overview

```
                      +-----------------------------+
                      |   API Specification Input   |
                      | OpenAPI, Swagger, Postman   |
                      +-----------------------------+
                                     |
                                     v
                      +-----------------------------+
                      |     Intermediate Parser     |
                      |  Normalizes endpoints & ops |
                      +-----------------------------+
                                     |
                                     v
                      +-----------------------------+
                      |   Schema Hardening Engine   |
                      | Pruning, Capping, Enhancing |
                      +-----------------------------+
                                     |
             +-----------------------+-----------------------+
             |                                               |
             v                                               v
+-------------------------+                     +-------------------------+
|     Code Generators     |                     |   Direct Runtime Host   |
| Go (stdio) | TypeScript |                     | stdio / SSE MCP Server  |
+-------------------------+                     +-------------------------+
```

Detailed architectural specifications and design decisions are documented in [docs/architecture.md](docs/architecture.md).

---

## Schema Hardening Details

Enterprise APIs often expose dozens of optional fields and technical metadata headers that degrade LLM reliability. toolcraft addresses this through three stages:

1. Redundant Parameter Pruning: Discards parameters matching regex filters (e.g. `X-Request-Id`, `internal_tenant_hint`, `debug_mode`).
2. Priority Capping: Restricts optional parameters to the highest-utility inputs, ensuring prompt token consumption remains minimal.
3. Instruction Enrichment: Expands terse summaries (e.g. "Get invoices") into explicit instructions with required identifiers and domain context.

Further details are available in [docs/spec-hardening.md](docs/spec-hardening.md).

---

## CLI Reference

| Command | Description |
|---|---|
| `toolcraft generate` | Generates a Go or TypeScript MCP server from an API specification. |
| `toolcraft serve` | Runs an ad-hoc in-memory MCP server over stdio or SSE. |
| `toolcraft inspect` | Analyzes a specification, displaying token savings and operation counts. |
| `toolcraft init` | Creates a baseline `toolcraft.yaml` configuration file. |
| `toolcraft tui` | Starts the interactive terminal user interface. |
| `toolcraft version` | Prints version, commit hash, and build date. |

Full flag details are available in [docs/cli-reference.md](docs/cli-reference.md).

---

## Verification and Testing

Run test suites across all packages:

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
