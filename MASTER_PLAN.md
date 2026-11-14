# toolcraft: Architecture and Engineering Master Plan
> Automated OpenAPI to Model Context Protocol (MCP) compiler, schema hardener, and runtime gateway.
> Target: High-performance single-binary Go executable (under 15MB RAM footprint) and enterprise tooling platform.

---

## 1. Executive Summary and Problem Statement

### 1.1 The Integration Bottleneck in Model Context Protocol (MCP)
Anthropic's Model Context Protocol (MCP) is the open standard connecting autonomous AI agents (Claude Code, Cursor, Antigravity, OpenCode) with external systems, APIs, and data sources.

Enterprise adoption faces three core operational obstacles:
1. Boilerplate Overhead: Enterprise systems maintain hundreds of REST endpoints documented in OpenAPI 3.0/3.1 or Swagger 2.0. Manually crafting MCP wrappers in TypeScript or Python requires substantial repetitive labor and ongoing maintenance.
2. Parameter Bloat and LLM Confusion: Enterprise APIs frequently expose 20 to 50 parameters per endpoint, including internal trace headers, legacy flags, and debugging fields. Presenting these unpruned schemas to an LLM inflates prompt token consumption and triggers argument hallucinations.
3. Security and Egress Exposure: Granting an autonomous agent direct access to internal endpoints without rate limits, secret masking, and read-only guards creates severe risks of unintended state mutations or credential leakage.

### 1.2 The toolcraft Solution
toolcraft is a single-binary compiler, schema optimizer, and runtime gateway implemented in Go:
- Universal Specification Ingestion: Ingests OpenAPI 3.0/3.1 (YAML and JSON), Swagger 2.0, Postman Collections v2.1, and raw cURL snippets.
- Dual-Target Code Generation:
  - Single-binary Go MCP server: Compiled into a standalone executable with zero runtime dependencies.
  - Modern TypeScript MCP package: Clean npm package using `@modelcontextprotocol/sdk` and strict Zod schemas.
- Schema Hardener and Token Optimizer:
  - Prunes low-value optional parameters and internal tracing headers.
  - Enriches technical summaries with structured instructions tailored for LLM tool selection.
  - Enforces JSON Schema and Zod type fidelity.
- Security Sandbox and Credential Shield:
  - Credential proxy injecting internal tokens (Bearer, API Key, Basic Auth) from local environment variables; models never receive raw secrets.
  - Configurable safety tiers: `--read-only` flag blocks mutating HTTP operations (POST, PUT, PATCH, DELETE).
  - Built-in token-bucket rate limiting per tool and endpoint.
- Direct MCP Runtime Gateway (`toolcraft serve`):
  - Exposes an OpenAPI specification directly as an active MCP server over stdio or SSE without requiring code generation or compilation steps.
- Interactive Terminal UI (TUI):
  - Fast keyboard-driven interface built with Bubbletea and Lipgloss for endpoint inspection, tool selection, schema tuning, and code generation.

---

## 2. High-Level System Architecture

```
       +--------------------------------------------------------+
       |             Enterprise API Specifications              |
       |     (OpenAPI 3.1 YAML/JSON, Postman, cURL, Swagger)    |
       +--------------------------------------------------------+
                                   |
                                   v
       +--------------------------------------------------------+
       |                   toolcraft CLI / TUI                  |
       |          (Single Go Binary, Sub-100ms Synthesis)       |
       +--------------------------------------------------------+
                                   |
                     +-------------+-------------+
                     |                           |
                     v                           v
     +-------------------------------+   +-------------------------------+
     |     Schema Hardening Engine   |   |   Security and Sandbox Layer  |
     | - Prunes redundant parameters |   | - Injects auth headers safely |
     | - Enriches tool descriptions  |   | - Enforces Read-Only guards   |
     | - Generates Zod/JSON-Schemas  |   | - Token-bucket rate limiting  |
     +-------------------------------+   +-------------------------------+
                     |                           |
                     +-------------+-------------+
                                   |
       +---------------------------+---------------------------+
       |                                                       |
       v                                                       v
+-----------------------------+               +-------------------------------+
| Target Code Generator (AST) |               | Direct MCP Gateway (Runtime)  |
| - Standalone Go MCP Server  |               | - stdio transport handler     |
| - TypeScript + Zod Package  |               | - SSE HTTP transport handler  |
+-----------------------------+               +-------------------------------+
       |                     |                                 |
       v                     v                                 v
+--------------+     +---------------+        +-------------------------------+
| Compiled Go  |     | NPM Package   |        | AI Agents (Claude Code, etc.) |
| (< 15MB RAM) |     | (Node.js ESM) |        | Connected via stdio or SSE    |
+--------------+     +---------------+        +-------------------------------+
```

---

## 3. Core Modules and Specifications

### 3.1 Parser Engine (`internal/parser`)
The parser normalizes diverse input sources into a canonical intermediate representation (`spec.IntermediateSpec`):
- `OpenAPIParser`: Parses OpenAPI 3.0.x and 3.1.x definitions in YAML and JSON formats.
- `SwaggerParser`: Converts Swagger 2.0 definitions into normalized endpoints, parameters, and responses.
- `PostmanParser`: Converts Postman Collection v2.1 JSON files into structured endpoints and query models.
- `CurlParser`: Ingests cURL command strings, extracting HTTP method, URL, headers, and request body payload.

### 3.2 Schema Hardener (`internal/hardener`)
The hardener transforms raw API definitions into high-signal tool schemas:
- Parameter Pruning (`pruner.go`):
  - Filters parameters matching regular expression blacklists (for example: `.*_trace_id`, `internal_.*`, `X-.*`, `legacy_.*`).
  - Automatically removes internal debugging headers and proxy metadata.
  - Enforces a configurable cap on parameters per tool (default: 6), prioritizing required fields and primary filters over optional flags.
- Description Enhancement (`enhancer.go`):
  - Converts concise summaries into unambiguous instructions.
  - Injects operational guidance ("Use this tool to fetch... Returns...").
  - Strips prompt-injection vectors and control sequences from endpoint documentation.
- Type Validation (`validator.go`):
  - Maps primitive types and arrays to JSON Schema and Zod equivalents.
  - Validates default values and enum constraints.

### 3.3 Target Code Synthesizers (`internal/generator`)
1. Go Generator (`internal/generator/golang`):
   - Generates an idiomatic, single-binary MCP server in Go.
   - Includes standard library HTTP client with timeout, context propagation, and error handling.
   - Implements JSON-RPC 2.0 protocol over stdio.
2. TypeScript Generator (`internal/generator/typescript`):
   - Generates a production-ready npm package using `@modelcontextprotocol/sdk` and `zod`.
   - Includes `package.json`, `tsconfig.json`, and typed request handlers.

### 3.4 Runtime Gateway (`internal/runtime`)
- Provides immediate execution without an intermediate build step.
- Operates over stdio (JSON-RPC) or Server-Sent Events (SSE over HTTP).
- Includes dynamic request proxying, environment credential injection, and request rate limiting.

### 3.5 Terminal User Interface (`internal/tui`)
- Bubbletea and Lipgloss interactive application.
- Allows browsing endpoints grouped by path or tag.
- Provides interactive toggling of endpoints to include in the generated tool set.
- Displays live parameter comparison (raw vs hardened) and token reduction metrics.
- One-key action to compile, export, or launch a live server.

---

## 4. Configuration Schema (`toolcraft.yaml`)

```yaml
version: "1"
input:
  spec_path: "./specs/billing_api_v3.yaml"
  spec_type: "openapi_v3" # openapi_v3, swagger_v2, postman_v2, curl

output:
  language: "go" # go, typescript
  output_dir: "./dist/billing-mcp-server"
  server_name: "billing-mcp"
  version: "1.0.0"

hardening:
  read_only: true
  prune_parameters:
    - ".*_trace_id"
    - "internal_.*"
    - "legacy_.*"
    - "X-.*"
  max_parameters_per_tool: 6
  enrich_descriptions: true

security:
  auth_type: "bearer" # bearer, apikey, basic, none
  token_env_var: "BILLING_API_TOKEN"
  api_key_header: "X-API-Key"
  rate_limit_rpm: 120
  timeout_seconds: 30
```

---

## 5. Repository Structure

```
toolcraft/
|-- .github/
|   `-- workflows/
|       `-- ci.yml
|-- cmd/
|   `-- toolcraft/
|       `-- main.go
|-- docs/
|   |-- architecture.md
|   |-- cli-reference.md
|   `-- spec-hardening.md
|-- examples/
|   |-- curl_sample.txt
|   |-- petstore_openapi3.yaml
|   |-- postman_collection_v2.json
|   `-- swagger_v2.json
|-- internal/
|   |-- config/
|   |   `-- config.go
|   |-- domain/
|   |   `-- models.go
|   |-- generator/
|   |   |-- golang/
|   |   |   `-- generator.go
|   |   `-- typescript/
|   |       `-- generator.go
|   |-- hardener/
|   |   |-- enhancer.go
|   |   |-- pruner.go
|   |   `-- validator.go
|   |-- parser/
|   |   |-- curl.go
|   |   |-- openapi.go
|   |   |-- parser.go
|   |   |-- postman.go
|   |   `-- swagger.go
|   |-- runtime/
|   |   |-- proxy.go
|   |   |-- ratelimit.go
|   |   |-- server.go
|   |   `-- stdio.go
|   `-- tui/
|       |-- app.go
|       |-- styles.go
|       `-- views.go
|-- go.mod
|-- go.sum
|-- LICENSE
|-- Makefile
|-- MASTER_PLAN.md
`-- README.md
```

---

## 6. Implementation Milestones

### Milestone 1: Foundation and Domain Models
- Initialize Go module and project directory layout.
- Define core domain models representing normalized endpoints, parameters, data types, and security schemes.
- Setup GitHub repository with proper topics, licensing, and CI automation.

### Milestone 2: Universal Parsers
- Implement parser for OpenAPI 3.0 and 3.1 specifications (YAML and JSON).
- Implement parser for Swagger 2.0 specifications.
- Implement parser for Postman Collection v2.1 format.
- Implement parser for raw cURL command snippets.
- Add comprehensive unit tests covering all formats with sample files.

### Milestone 3: Hardening and Optimization Engine
- Implement regex-based parameter pruning and redundant header elimination.
- Implement parameter cap and ranking heuristics.
- Implement description enhancer generating clear model instructions.
- Implement token reduction counter comparing raw vs optimized schemas.
- Add unit tests validating parameter filtering and schema integrity.

### Milestone 4: Dual Code Synthesizers
- Implement Go code generator outputting standalone, compilable Go MCP servers.
- Implement TypeScript code generator outputting npm package with Zod validation.
- Verify generated servers compile and validate input parameters properly.

### Milestone 5: Security Sandbox and Direct MCP Runtime
- Implement token-bucket rate limiter.
- Implement environment credential injection and header masking.
- Implement read-only enforcement blocking mutating operations.
- Implement direct `serve` command supporting stdio and SSE MCP transports.
- Add end-to-end protocol tests verifying `tools/list` and `tools/call`.

### Milestone 6: Terminal UI (TUI) and CLI Command Interface
- Build interactive TUI with Bubbletea and Lipgloss for endpoint inspection and selection.
- Implement CLI commands: `generate`, `serve`, `inspect`, `init`, `tui`, and `version`.
- Verify memory consumption remains under 15MB RAM.

### Milestone 7: Documentation and Final Polish
- Write factual, humanized documentation and guides.
- Verify zero placeholders, mockups, or hardcoded stubs.
