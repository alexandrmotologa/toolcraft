# toolcraft Architecture Specification

This document details the internal design, data flow, and components of toolcraft.

---

## 1. Design Principles

1. Minimal Resource Consumption: The compiled binary must run with under 15MB resident set size (RSS) during code synthesis and runtime proxying.
2. Zero Dependency Deployment: The Go compiler target produces standalone single-binary executables with no external runtime dependencies.
3. Strict Defense in Depth: LLMs must never directly handle raw credentials. Authentication tokens, mTLS certificates, and internal headers are managed by the runtime sandbox layer.
4. Deterministic Output: Generated code and hardened schemas must be fully deterministic across invocations given the same input specification and configuration.

---

## 2. Pipeline Architecture

The toolcraft execution pipeline processes an API specification through four sequential stages:

```
[Input Spec] -> [Parser Engine] -> [Intermediate Model] -> [Hardener Engine] -> [Code Generator / Runtime]
```

### 2.1 Parser Stage (`internal/parser`)

The parser ingests heterogeneous inputs and translates them into a canonical `spec.IntermediateSpec`:

- `OpenAPIParser`: Handles OpenAPI 3.0 and 3.1 YAML and JSON files. Resolves local `$ref` pointers within the document, extracts server URLs, path items, operations, parameters (path, query, header, cookie), request bodies, and JSON schemas.
- `SwaggerParser`: Converts legacy Swagger 2.0 definitions. Normalizes Swagger parameter definitions (in query, path, header, formData) and maps definitions to unified schema types.
- `PostmanParser`: Ingests Postman Collection v2.1 exports. Extracts endpoints, URLs, query parameters, authorization headers, and sample JSON payloads.
- `CurlParser`: Parses raw cURL command lines. Extracts target URL, HTTP method, headers, and request body.

### 2.2 Intermediate Representation (`internal/domain`)

The intermediate representation isolates parsing from code generation:

```go
type ToolDefinition struct {
    ID          string                 // Canonical tool name (e.g., get_customer_invoices)
    Method      string                 // HTTP method (GET, POST, etc.)
    Path        string                 // Endpoint URL path template (/v1/customers/{id})
    Summary     string                 // Original summary
    Description string                 // Model-optimized instruction
    Parameters  []ParameterDefinition  // Hardened parameters
    RequestBody *RequestBodyDefinition // Optional request body schema
    Security    SecurityRequirement    // Auth scheme and requirements
    ReadOnly    bool                   // True if operation is non-mutating (GET/HEAD/OPTIONS)
}
```

### 2.3 Hardener Stage (`internal/hardener`)

The hardener transforms raw API models into compact, unambiguous schemas optimized for LLM tool selection:

1. Pruning:
   - Excludes headers and query fields matching configurable regular expressions (e.g. `.*_trace_id`, `internal_.*`, `X-.*`).
   - Removes internal routing and caching hints that offer no functional value to LLM agents.
2. Priority Capping:
   - Sorts parameters: required path parameters rank first, required query parameters second, high-value filters third, and optional flags last.
   - Truncates parameters exceeding the threshold `max_parameters_per_tool` (default: 6).
3. Instruction Enrichment:
   - Evaluates description quality. If empty or shorter than 20 characters, generates structured operational instructions specifying purpose, required parameters, and expected outcome.
   - Cleans formatting, removing control sequences and potential prompt-injection patterns.

### 2.4 Code Generator Stage (`internal/generator`)

#### Go Target (`internal/generator/golang`)
- Generates a standalone Go program implementing the Model Context Protocol stdio transport.
- Uses standard library `net/http` client with connection pooling, configurable timeouts, and context cancellation.
- Implements JSON-RPC 2.0 message handling for `initialize`, `notifications/initialized`, `tools/list`, and `tools/call`.
- Compiles with zero external CGO dependencies.

#### TypeScript Target (`internal/generator/typescript`)
- Generates an ESM project with `package.json`, `tsconfig.json`, and `src/index.ts`.
- Uses official `@modelcontextprotocol/sdk` for MCP stdio protocol compliance.
- Employs `zod` for runtime schema validation on every tool call.
- Uses native `fetch` for outbound HTTP requests with Bearer and API key injection.

### 2.5 Direct Runtime Gateway (`internal/runtime`)

For scenarios where writing code files is unnecessary, `toolcraft serve` acts directly as an in-memory MCP gateway:
- Reads specification into memory, executes hardening on the fly, and registers tools directly into an in-memory MCP server.
- Supports `stdio` transport for direct local process connection from Claude Code or Cursor.
- Supports `sse` transport over HTTP for remote and containerized agent connections.
- Houses the token-bucket rate limiter and environment credential proxy.

---

## 3. Security and Guardrails

### 3.1 Credential Shield
- Target API secrets (Bearer tokens, API keys) are configured through environment variable references (e.g. `BILLING_API_KEY`).
- Credentials are read at runtime by the local process or generated client and attached to outbound requests.
- Credentials never appear in tool schemas, JSON-RPC messages, or stdout logs.

### 3.2 Read-Only Enforcement
When `--read-only` is active:
- The parser/hardener flags mutating methods (`POST`, `PUT`, `PATCH`, `DELETE`).
- Code generators either omit mutating tools entirely or generate explicit rejection guards.
- The runtime gateway rejects any attempt to execute mutating operations with an MCP error response.

### 3.3 Rate Limiting
- Employs a token bucket algorithm per endpoint.
- Default limit is 120 requests per minute, configurable via CLI or YAML.
- Exceeding the rate limit returns a standard MCP error with code `-32000` and retry advice.
