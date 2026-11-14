package mock_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandrmotologa/toolcraft/internal/mock"
	"github.com/alexandrmotologa/toolcraft/internal/parser"
)

func TestMockServerLifecycleAndEndpoints(t *testing.T) {
	specPath := filepath.Join("..", "..", "examples", "petstore_openapi3.yaml")
	spec, err := parser.DetectAndParse(specPath)
	if err != nil {
		t.Fatalf("failed to parse spec: %v", err)
	}

	srv := mock.NewMockServer(spec)
	baseURL, err := srv.Start("127.0.0.1", 0) // random free port
	if err != nil {
		t.Fatalf("failed to start mock server: %v", err)
	}
	defer srv.Stop(context.Background())

	client := &http.Client{Timeout: 5 * time.Second}

	// 1. Test GET /pets (collection -> array response)
	resp, err := client.Get(baseURL + "/pets")
	if err != nil {
		t.Fatalf("GET /pets failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	var items []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("failed to decode array: %v", err)
	}
	if len(items) == 0 {
		t.Errorf("expected mock array items, got empty")
	}

	// 2. Test GET /pets/123 (single item record)
	resp2, err := client.Get(baseURL + "/pets/123")
	if err != nil {
		t.Fatalf("GET /pets/123 failed: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp2.StatusCode)
	}

	var item map[string]interface{}
	if err := json.NewDecoder(resp2.Body).Decode(&item); err != nil {
		t.Fatalf("failed to decode item: %v", err)
	}
	if item["status"] != "success" {
		t.Errorf("expected status 'success', got %v", item["status"])
	}

	// 3. Test 404 for unknown route
	resp3, err := client.Get(baseURL + "/unknown/route")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp3.StatusCode)
	}
}
