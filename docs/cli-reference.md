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

## Commands

### `toolcraft generate`

Synthesizes an MCP server from an input specification into Go or TypeScript source code.

```bash
toolcraft generate --spec <path> --lang <go|typescript> --out <dir> [flags]
```

Flags:
- `--spec, -s`: Path or URL to the API specification file. Supported formats: OpenAPI 3.0/3.1 (YAML/JSON), Swagger 2.0 (JSON/YAML), Postman Collection v2.1 (JSON), raw cURL text file.
- `--lang, -l`: Target output language. Accepted values: `go`, `typescript`. Default: `go`.
- `--out, -o`: Destination directory for generated code. Default: `./dist/mcp-server`.
- `--server-name`: Canonical server name identifier. Default: `mcp-server`.
- `--read-only`: Disables mutating HTTP operations (`POST`, `PUT`, `PATCH`, `DELETE`).
- `--max-params`: Maximum optional parameters retained per tool. Default: `6`.
- `--auth-type`: Authentication scheme. Accepted values: `none`, `bearer`, `apikey`, `basic`. Default: `none`.
- `--auth-env`: Environment variable name holding target credentials. Default: `API_KEY` or `API_TOKEN`.
- `--rate-limit`: Maximum requests per minute per tool. Default: `120`.

---

### `toolcraft serve`

Starts an in-memory MCP gateway exposing tools directly over stdio or SSE without writing files.

```bash
toolcraft serve --spec <path> [flags]
```

Flags:
- `--spec, -s`: Path or URL to the API specification file.
- `--transport, -t`: MCP transport mode. Accepted values: `stdio`, `sse`. Default: `stdio`.
- `--port, -p`: Listening port when running in SSE transport mode. Default: `8080`.
- `--host`: Binding host for SSE transport. Default: `127.0.0.1`.
- `--read-only`: Rejects mutating operations at runtime.
- `--auth-env`: Environment variable containing authentication token.
- `--rate-limit`: Maximum requests per minute. Default: `120`.

---

### `toolcraft inspect`

Analyzes an input specification, displaying operations, parameter counts, and token optimization metrics.

```bash
toolcraft inspect --spec <path> [flags]
```

Flags:
- `--spec, -s`: Path or URL to the API specification file.
- `--json`: Output inspection results as JSON for programmatic consumption.
- `--max-params`: Parameter cap applied during simulation. Default: `6`.

---

### `toolcraft init`

Generates a default `toolcraft.yaml` configuration file in the current directory.

```bash
toolcraft init [flags]
```

Flags:
- `--force, -f`: Overwrite existing configuration file if present.

---

### `toolcraft tui`

Launches the interactive Terminal User Interface for browsing endpoints, inspecting schemas, and generating servers.

```bash
toolcraft tui --spec <path>
```

Keyboard Shortcuts in TUI:
- `Up` / `Down` or `k` / `j`: Navigate endpoint list.
- `Space`: Toggle endpoint selection.
- `a`: Select or deselect all endpoints.
- `Tab`: Switch focus between endpoint list and detail preview pane.
- `g`: Open generation modal.
- `s`: Launch live serve mode.
- `q` / `Esc`: Quit application.

---

### `toolcraft version`

Displays binary version, git commit hash, and build timestamp.
