package runtime

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

// Server represents an active in-memory Model Context Protocol runtime host.
type Server struct {
	spec      *domain.Spec
	config    *config.Config
	proxy     *Proxy
	toolsMap  map[string]domain.ToolDefinition
	mu        sync.RWMutex
	sessions  map[string]chan string
	sessionMu sync.Mutex
}

// NewServer initializes a runtime MCP server.
func NewServer(spec *domain.Spec, cfg *config.Config) *Server {
	toolsMap := make(map[string]domain.ToolDefinition)
	for _, t := range spec.Tools {
		toolsMap[t.ID] = t
	}

	return &Server{
		spec:     spec,
		config:   cfg,
		proxy:    NewProxy(cfg),
		toolsMap: toolsMap,
		sessions: make(map[string]chan string),
	}
}

// JSON-RPC 2.0 structures
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    map[string]any `json:"capabilities"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type ListToolsResult struct {
	Tools []ToolInfo `json:"tools"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// HandleRPC processes an incoming JSON-RPC request and returns the corresponding response.
func (s *Server) HandleRPC(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		serverName := s.config.Output.ServerName
		if serverName == "" {
			serverName = "toolcraft-mcp-gateway"
		}
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: "2024-11-05",
				Capabilities: map[string]any{
					"tools": map[string]any{},
				},
				ServerInfo: ServerInfo{
					Name:    serverName,
					Version: s.config.Output.Version,
				},
			},
		}

	case "notifications/initialized":
		return nil // Notifications require no response

	case "tools/list":
		s.mu.RLock()
		tools := make([]ToolInfo, 0, len(s.spec.Tools))
		for _, t := range s.spec.Tools {
			properties := make(map[string]interface{})
			var required []string

			for _, p := range t.Parameters {
				properties[p.Name] = map[string]interface{}{
					"type":        p.Type,
					"description": p.Description,
				}
				if p.Required {
					required = append(required, p.Name)
				}
			}

			if t.RequestBody != nil {
				for _, p := range t.RequestBody.Properties {
					properties[p.Name] = map[string]interface{}{
						"type":        p.Type,
						"description": p.Description,
					}
					if p.Required {
						required = append(required, p.Name)
					}
				}
			}

			tools = append(tools, ToolInfo{
				Name:        t.ID,
				Description: t.Description,
				InputSchema: map[string]interface{}{
					"type":       "object",
					"properties": properties,
					"required":   required,
				},
			})
		}
		s.mu.RUnlock()

		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ListToolsResult{
				Tools: tools,
			},
		}

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &RPCError{
					Code:    -32602,
					Message: "Invalid parameters format",
				},
			}
		}

		s.mu.RLock()
		tool, exists := s.toolsMap[params.Name]
		s.mu.RUnlock()

		if !exists {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: CallToolResult{
					Content: []ContentItem{
						{
							Type: "text",
							Text: fmt.Sprintf("Error: tool '%s' not found", params.Name),
						},
					},
					IsError: true,
				},
			}
		}

		res, err := s.proxy.ExecuteTool(ctx, &tool, s.spec.BaseURL, params.Arguments)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: CallToolResult{
					Content: []ContentItem{
						{
							Type: "text",
							Text: fmt.Sprintf("Error: %s", err.Error()),
						},
					},
					IsError: true,
				},
			}
		}

		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{
					{
						Type: "text",
						Text: res,
					},
				},
			},
		}

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

// ServeStdio executes the server loop reading from in and writing to out.
func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			errResp := JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &RPCError{
					Code:    -32700,
					Message: "Parse error",
				},
			}
			data, _ := json.Marshal(errResp)
			fmt.Fprintln(out, string(data))
			continue
		}

		resp := s.HandleRPC(ctx, &req)
		if resp != nil {
			data, _ := json.Marshal(resp)
			fmt.Fprintln(out, string(data))
		}
	}
	return scanner.Err()
}

// ServeSSE starts an HTTP server providing Server-Sent Events transport for MCP.
func (s *Server) ServeSSE(ctx context.Context, host string, port int) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		sessionID := fmt.Sprintf("session-%d", timeNowUnixNano())
		msgChan := make(chan string, 16)

		s.sessionMu.Lock()
		s.sessions[sessionID] = msgChan
		s.sessionMu.Unlock()

		defer func() {
			s.sessionMu.Lock()
			delete(s.sessions, sessionID)
			s.sessionMu.Unlock()
		}()

		// Send endpoint event to client
		fmt.Fprintf(w, "event: endpoint\ndata: /message?sessionId=%s\n\n", sessionID)
		flusher.Flush()

		notify := r.Context().Done()
		for {
			select {
			case <-notify:
				return
			case <-ctx.Done():
				return
			case msg := <-msgChan:
				fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
				flusher.Flush()
			}
		}
	})

	mux.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			http.Error(w, "Invalid JSON-RPC request", http.StatusBadRequest)
			return
		}

		resp := s.HandleRPC(r.Context(), &req)
		if resp == nil {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		respBytes, _ := json.Marshal(resp)
		sessionID := r.URL.Query().Get("sessionId")

		s.sessionMu.Lock()
		ch, exists := s.sessions[sessionID]
		s.sessionMu.Unlock()

		if exists && ch != nil {
			select {
			case ch <- string(respBytes):
			default:
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(respBytes)
	})

	addr := fmt.Sprintf("%s:%d", host, port)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	log.Printf("MCP SSE server listening on http://%s\n", addr)
	return srv.ListenAndServe()
}

func timeNowUnixNano() int64 {
	return time.Now().UnixNano()
}
