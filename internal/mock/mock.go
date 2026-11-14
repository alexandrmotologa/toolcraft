package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/toolcraft/internal/domain"
)

// MockServer provides an in-memory HTTP service returning synthetic responses matching API schemas.
type MockServer struct {
	spec    *domain.Spec
	server  *http.Server
	baseURL string
}

// NewMockServer constructs a mock HTTP service.
func NewMockServer(spec *domain.Spec) *MockServer {
	return &MockServer{spec: spec}
}

// Start launches the mock server on the requested host and port.
func (m *MockServer) Start(host string, port int) (string, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", m.handleRequest)

	addr := fmt.Sprintf("%s:%d", host, port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	actualPort := listener.Addr().(*net.TCPAddr).Port
	m.baseURL = fmt.Sprintf("http://127.0.0.1:%d", actualPort)

	m.server = &http.Server{
		Handler: mux,
	}

	go func() {
		if err := m.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Mock server error: %v", err)
		}
	}()

	return m.baseURL, nil
}

// Stop shuts down the mock server gracefully.
func (m *MockServer) Stop(ctx context.Context) error {
	if m.server != nil {
		return m.server.Shutdown(ctx)
	}
	return nil
}

// BaseURL returns the active listening URL.
func (m *MockServer) BaseURL() string {
	return m.baseURL
}

func (m *MockServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Path
	reqMethod := strings.ToUpper(r.Method)

	for _, tool := range m.spec.Tools {
		if strings.ToUpper(tool.Method) != reqMethod {
			continue
		}

		pathRegexStr := "^" + regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(tool.Path, `([^/]+)`) + "$"
		re, err := regexp.Compile(pathRegexStr)
		if err != nil {
			continue
		}

		if re.MatchString(reqPath) {
			m.respondForTool(w, &tool, reqPath)
			return
		}
	}

	// 404 fallback
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   "Not Found",
		"message": fmt.Sprintf("No mock endpoint matches %s %s", reqMethod, reqPath),
	})
}

func (m *MockServer) respondForTool(w http.ResponseWriter, tool *domain.ToolDefinition, path string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if tool.Method == "DELETE" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if tool.Method == "POST" {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	synthetic := generateSyntheticPayload(tool, path)
	json.NewEncoder(w).Encode(synthetic)
}

func generateSyntheticPayload(tool *domain.ToolDefinition, path string) interface{} {
	// If path ends with collection noun (e.g. /pets, /items, /invoices), generate array
	segments := strings.Split(strings.Trim(path, "/"), "/")
	lastSegment := segments[len(segments)-1]
	isCollection := !strings.Contains(lastSegment, "{") && strings.HasSuffix(lastSegment, "s") && tool.Method == "GET"

	if isCollection {
		return []map[string]interface{}{
			{
				"id":     "sample_01",
				"name":   "Primary " + strings.TrimSuffix(lastSegment, "s"),
				"status": "active",
			},
			{
				"id":     "sample_02",
				"name":   "Secondary " + strings.TrimSuffix(lastSegment, "s"),
				"status": "pending",
			},
		}
	}

	// Single record payload
	result := make(map[string]interface{})
	result["id"] = "mock_" + tool.ID
	result["status"] = "success"
	result["operation"] = tool.ID

	if tool.RequestBody != nil {
		for _, prop := range tool.RequestBody.Properties {
			result[prop.Name] = sampleValueForType(prop.Type, prop.Name)
		}
	}

	for _, p := range tool.Parameters {
		if p.In == "path" || p.Required {
			result[p.Name] = sampleValueForType(p.Type, p.Name)
		}
	}

	return result
}

func sampleValueForType(dataType, name string) interface{} {
	lowerName := strings.ToLower(name)
	switch strings.ToLower(dataType) {
	case "integer":
		if strings.Contains(lowerName, "quantity") || strings.Contains(lowerName, "count") {
			return 2
		}
		if strings.Contains(lowerName, "limit") {
			return 20
		}
		return 42
	case "number":
		return 99.95
	case "boolean":
		return true
	case "array":
		return []string{"sample_item_1", "sample_item_2"}
	default:
		if strings.Contains(lowerName, "email") {
			return "user@example.com"
		}
		if strings.Contains(lowerName, "id") {
			return "rec_99182a"
		}
		return "sample_" + name
	}
}
