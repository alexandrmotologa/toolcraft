# toolcraft CLI Reference Manual

This document details all command-line interface commands, flags, and environment variables.

---

## Global Options

The following flags apply to all subcommands:

| Flag | Short | Default | Description |
|---|---|---|---|
| `--config` | `-c` | `""` | Path to `toolcraft.yaml` configuration manifest. |
| `--verbose` | `-v` | `false` | Enable verbose logging output to stderr. |
| `--help` | `-h` | | Display help message for command. |

---

## Commands Summary

| Command | Description |
|---|---|
| `toolcraft generate` | Synthesize an MCP server into Go or TypeScript source code. |
| `toolcraft serve` | Run an immediate in-memory MCP gateway over stdio or SSE. |
| `toolcraft mock` | Run an in-memory synthetic mock API server matching OpenAPI schemas. |
| `toolcraft config` | Output a ready-to-copy client configuration snippet (Claude, Cursor, Antigravity). |
| `toolcraft install` | Automatically register an MCP server in the client application config file. |
| `toolcraft sanitize` | Export a pruned, LLM-optimized OpenAPI 3.1 specification (YAML/JSON). |
| `toolcraft catalog` | Generate an interactive Markdown or HTML catalog of all exposed tools. |
| `toolcraft inspect` | Analyze an API specification and display token reduction metrics. |
| `toolcraft init` | Initialize a baseline `toolcraft.yaml` configuration file. |
| `toolcraft tui` | Start the keyboard-driven interactive Terminal User Interface. |
| `toolcraft version` | Display binary version, commit hash, and build timestamp. |

---

## Detailed Command Specifications

### `toolcraft generate`

Synthesizes an MCP server from an input specification into Go or TypeScript source code.

```bash
toolcraft generate --spec <path|url> --lang <go|typescript> --out <dir> [flags]
```

Flags:
- `--spec, -s`: Path or URL to the API specification. Supported: OpenAPI 3.0/3.1 (YAML/JSON), Swagger 2.0 (JSON/YAML), Postman Collection v2.1, cURL snippets.
- `--lang, -l`: Target output language (`go`, `typescript`). Default: `go`.
- `--out, -o`: Destination directory. Default: `./dist/mcp-server`.
- `--server-name`: Canonical server name identifier. Default: `mcp-server`.
- `--read-only`: Disallow mutating HTTP methods (`POST`, `PUT`, `DELETE`).
- `--max-params`: Maximum parameters retained per tool. Default: `6`.
- `--auth-type`: Authentication scheme (`none`, `bearer`, `apikey`, `basic`).
- `--auth-env`: Environment variable name holding target credentials.
- `--rate-limit`: Maximum requests per minute per tool. Default: `120`.
- `--spec-header`: Custom header for remote spec download (e.g. `--spec-header "Authorization: Bearer <token>"`).

---

### `toolcraft serve`

Starts an in-memory MCP gateway exposing tools directly over stdio or SSE without writing files.

```bash
toolcraft serve --spec <path|url> [flags]
```

Flags:
- `--spec, -s`: Path or URL to the API specification.
- `--transport, -t`: MCP transport mode (`stdio`, `sse`). Default: `stdio`.
- `--port, -p`: Listening port for SSE transport. Default: `8080`.
- `--host`: Binding host for SSE transport. Default: `127.0.0.1`.
- `--read-only`: Rejects mutating operations at runtime.
- `--auth-env`: Environment variable containing authentication token.
- `--rate-limit`: Maximum requests per minute. Default: `120`.
- `--spec-header`: Custom header for remote spec download.

---

### `toolcraft mock`

Runs an in-memory mock API service generating schema-compliant synthetic JSON responses.

```bash
toolcraft mock --spec <path|url> [flags]
```

Flags:
- `--spec, -s`: Path or URL to the API specification.
- `--port, -p`: Listening HTTP port. Default: `8085`.
- `--host`: Binding host interface. Default: `127.0.0.1`.
- `--bridge-mcp`: Bridges MCP stdio transport directly to this mock server for fully offline agent testing.

---

### `toolcraft config`

Generates ready-to-copy JSON configuration blocks for popular AI agent clients.

```bash
toolcraft config --client <claude|cursor|antigravity> --spec <path> [flags]
```

Flags:
- `--client`: Target agent client (`claude`, `cursor`, `antigravity`). Default: `claude`.
- `--server-name`: MCP server identifier.
- `--binary`: Explicit executable path or command name.
- `--spec, -s`: Path to the specification file.
- `--auth-env`: Environment variable holding credentials.

---

### `toolcraft install`

Injects the MCP server entry directly into the host client application configuration file. Automatically creates a safety backup (`.bak`).

```bash
toolcraft install --client <claude|cursor|antigravity> --spec <path> [flags]
```

Flags:
- `--client`: Target agent client (`claude`, `cursor`, `antigravity`).
- `--server-name`: MCP server identifier.
- `--binary`: Explicit executable path.
- `--spec, -s`: Path to specification.
- `--auth-env`: Environment variable holding credentials.
- `--force, -f`: Overwrite corrupted target config if needed.

---

### `toolcraft sanitize`

Exports an OpenAPI 3.1 specification purified by the schema hardener (pruned parameters, enriched instructions).

```bash
toolcraft sanitize --spec <path> --out <path.yaml|json> [flags]
```

Flags:
- `--spec, -s`: Path to input specification.
- `--out, -o`: Output path ending in `.yaml`, `.yml`, or `.json`.
- `--max-params`: Parameter cap per tool. Default: `6`.
- `--read-only`: Discard mutating endpoints.

---

### `toolcraft catalog`

Generates a human and agent readable documentation catalog in Markdown or self-contained HTML.

```bash
toolcraft catalog --spec <path> --format <markdown|html> --out <path>
```

Flags:
- `--spec, -s`: Path to specification file.
- `--format, -f`: Output format (`markdown`, `html`). Default: `markdown`.
- `--out, -o`: Destination file path.

---

### `toolcraft inspect`

Analyzes an input specification and prints token optimization metrics.

```bash
toolcraft inspect --spec <path|url> [flags]
```

Flags:
- `--spec, -s`: Path or URL to specification file.
- `--json`: Output metrics as machine-readable JSON.
- `--max-params`: Parameter cap simulation limit. Default: `6`.

---

### `toolcraft init`

Generates a baseline `toolcraft.yaml` configuration manifest.

```bash
toolcraft init [--force]
```

---

### `toolcraft tui`

Launches the interactive Terminal User Interface.

```bash
toolcraft tui --spec <path>
```

Keyboard Shortcuts:
- `Up` / `Down` or `k` / `j`: Move cursor in tools list.
- `Space`: Toggle active tool inclusion.
- `a`: Select or deselect all tools.
- `Tab`: Switch pane focus.
- `x`: Execute test call for selected tool.
- `g`: Generate Go MCP server.
- `t`: Generate TypeScript MCP package.
- `q`: Quit.
