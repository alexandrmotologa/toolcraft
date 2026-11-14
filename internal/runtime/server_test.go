package runtime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/hardener"
	"github.com/alexandrmotologa/toolcraft/internal/parser"
	"github.com/alexandrmotologa/toolcraft/internal/runtime"
)

func setupTestServer(t *testing.T, readOnly bool) (*runtime.Server, *httptest.Server) {
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if r.URL.Path == "/pets/123" && r.Method == "GET" {
			if auth != "Bearer test-secret-token" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"id":"123","name":"Buddy","status":"available"}`))
			return
		}

		if r.URL.Path == "/pets" && r.Method == "POST" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"124","name":"Max"}`))
			return
		}

		http.NotFound(w, r)
	}))

	specPath := filepath.Join("..", "..", "examples", "petstore_openapi3.yaml")
	spec, err := parser.DetectAndParse(specPath)
	if err != nil {
		t.Fatalf("failed to parse spec: %v", err)
	}
	spec.BaseURL = mockAPI.URL

	cfg := config.DefaultConfig()
	cfg.Hardening.ReadOnly = readOnly
	cfg.Security.AuthType = "bearer"
	cfg.Security.TokenEnvVar = "TEST_API_TOKEN"

	hardened, _, err := hardener.Harden(spec, cfg.Hardening)
	if err != nil {
		t.Fatalf("failed to harden spec: %v", err)
	}

	server := runtime.NewServer(hardened, cfg)
	return server, mockAPI
}

func TestServerInitializeAndToolsList(t *testing.T) {
	server, mockAPI := setupTestServer(t, false)
	defer mockAPI.Close()

	ctx := context.Background()

	// 1. Initialize
	initReq := &runtime.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	resp := server.HandleRPC(ctx, initReq)
	if resp == nil || resp.Error != nil {
		t.Fatalf("initialize failed: %+v", resp)
	}

	initResult, ok := resp.Result.(runtime.InitializeResult)
	if !ok {
		t.Fatalf("invalid result type for initialize: %T", resp.Result)
	}
	if initResult.ProtocolVersion != "2024-11-05" {
		t.Errorf("unexpected protocol version: %s", initResult.ProtocolVersion)
	}

	// 2. Tools list
	listReq := &runtime.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	listResp := server.HandleRPC(ctx, listReq)
	if listResp == nil || listResp.Error != nil {
		t.Fatalf("tools/list failed: %+v", listResp)
	}

	listResult, ok := listResp.Result.(runtime.ListToolsResult)
	if !ok {
		t.Fatalf("invalid result type for tools/list: %T", listResp.Result)
	}
	if len(listResult.Tools) == 0 {
		t.Errorf("expected registered tools, got 0")
	}
}

func TestServerToolCallSuccessAndAuth(t *testing.T) {
	server, mockAPI := setupTestServer(t, false)
	defer mockAPI.Close()

	os.Setenv("TEST_API_TOKEN", "test-secret-token")
	defer os.Unsetenv("TEST_API_TOKEN")

	ctx := context.Background()

	callParams := runtime.CallToolParams{
		Name: "get_pet_by_id",
		Arguments: map[string]interface{}{
			"petId": "123",
		},
	}
	paramBytes, _ := json.Marshal(callParams)

	callReq := &runtime.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  paramBytes,
	}

	resp := server.HandleRPC(ctx, callReq)
	if resp == nil || resp.Error != nil {
		t.Fatalf("tool call failed: %+v", resp)
	}

	callResult, ok := resp.Result.(runtime.CallToolResult)
	if !ok {
		t.Fatalf("invalid result type: %T", resp.Result)
	}

	if callResult.IsError {
		t.Fatalf("call reported error: %s", callResult.Content[0].Text)
	}

	if !strings.Contains(callResult.Content[0].Text, "Buddy") {
		t.Errorf("expected response to contain Buddy, got: %s", callResult.Content[0].Text)
	}
}

func TestServerReadOnlyGuardrail(t *testing.T) {
	server, mockAPI := setupTestServer(t, true) // Read-only enabled
	defer mockAPI.Close()

	ctx := context.Background()

	callParams := runtime.CallToolParams{
		Name: "create_pet",
		Arguments: map[string]interface{}{
			"name": "Fido",
		},
	}
	paramBytes, _ := json.Marshal(callParams)

	callReq := &runtime.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  paramBytes,
	}

	resp := server.HandleRPC(ctx, callReq)
	if resp == nil {
		t.Fatalf("expected response")
	}

	callResult, ok := resp.Result.(runtime.CallToolResult)
	if !ok {
		t.Fatalf("invalid result type: %T", resp.Result)
	}

	// Tool should not exist or be rejected in read-only mode
	if !callResult.IsError {
		t.Errorf("mutating operation should have failed in read-only mode")
	}
}

func TestServerServeStdioLoop(t *testing.T) {
	server, mockAPI := setupTestServer(t, false)
	defer mockAPI.Close()

	inBuf := bytes.NewBufferString(`{"jsonrpc":"2.0","id":10,"method":"initialize"}` + "\n")
	var outBuf bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := server.ServeStdio(ctx, inBuf, &outBuf)
	if err != nil {
		t.Fatalf("ServeStdio error: %v", err)
	}

	outStr := outBuf.String()
	if !strings.Contains(outStr, `"protocolVersion":"2024-11-05"`) {
		t.Errorf("expected output to contain protocol version, got: %s", outStr)
	}
}
