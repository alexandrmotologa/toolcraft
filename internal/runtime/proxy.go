package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alexandrmotologa/toolcraft/internal/config"
	"github.com/alexandrmotologa/toolcraft/internal/domain"
	"github.com/alexandrmotologa/toolcraft/internal/security"
)

// Proxy handles outbound HTTP execution with credential injection and safety guardrails.
type Proxy struct {
	client      *http.Client
	config      *config.Config
	rateLimiter *RateLimiter
}

// NewProxy creates a configured outbound proxy.
func NewProxy(cfg *config.Config) *Proxy {
	timeout := 30 * time.Second
	if cfg.Security.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.Security.TimeoutSeconds) * time.Second
	}

	return &Proxy{
		client: &http.Client{
			Timeout: timeout,
		},
		config:      cfg,
		rateLimiter: NewRateLimiter(cfg.Security.RateLimitRPM),
	}
}

// ExecuteTool calls the underlying API endpoint for a given tool definition and arguments.
func (p *Proxy) ExecuteTool(ctx context.Context, tool *domain.ToolDefinition, baseURL string, args map[string]interface{}) (string, error) {
	// Guard: check rate limiter
	if !p.rateLimiter.Allow() {
		return "", fmt.Errorf("rate limit exceeded (maximum %d requests per minute permitted)", p.config.Security.RateLimitRPM)
	}

	// Guard: check read-only mode
	if p.config.Hardening.ReadOnly && !tool.ReadOnly {
		return "", fmt.Errorf("operation blocked: server is configured in strict read-only mode, cannot execute %s", tool.Method)
	}

	if baseURL == "" {
		baseURL = os.Getenv("API_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	path := tool.Path
	queryParams := url.Values{}
	bodyMap := make(map[string]interface{})

	// Map inputs to path, query, or body
	for k, v := range args {
		placeholder := fmt.Sprintf("{%s}", k)
		if strings.Contains(path, placeholder) {
			path = strings.ReplaceAll(path, placeholder, url.PathEscape(fmt.Sprintf("%v", v)))
			continue
		}

		if tool.RequestBody != nil {
			bodyMap[k] = v
		} else {
			queryParams.Add(k, fmt.Sprintf("%v", v))
		}
	}

	targetURL := baseURL + path
	if len(queryParams) > 0 {
		targetURL += "?" + queryParams.Encode()
	}

	var reqBody io.Reader
	if len(bodyMap) > 0 {
		b, err := json.Marshal(bodyMap)
		if err != nil {
			return "", fmt.Errorf("failed to encode request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, tool.Method, targetURL, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("User-Agent", "toolcraft-mcp-gateway/1.0")
	req.Header.Set("Accept", "application/json")
	if len(bodyMap) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	// Credential shield injection
	p.injectCredentials(req, tool)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	sanitized, err := security.SanitizeResponseBody(respBytes, security.RedactorOptions{
		RedactPII:     p.config.Security.RedactPII,
		MaxArrayItems: p.config.Security.MaxArrayItems,
		MaxBytes:      p.config.Security.MaxResponseBytes,
	})
	if err == nil {
		return string(sanitized), nil
	}

	return string(respBytes), nil
}

func (p *Proxy) injectCredentials(req *http.Request, tool *domain.ToolDefinition) {
	authType := p.config.Security.AuthType
	if authType == "" || authType == "none" {
		authType = tool.Security.AuthType
	}

	envVar := p.config.Security.TokenEnvVar
	if envVar == "" {
		envVar = tool.Security.EnvVar
	}
	if envVar == "" {
		envVar = "API_KEY"
	}

	token := os.Getenv(envVar)
	if token == "" {
		return
	}

	switch strings.ToLower(authType) {
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+token)
	case "apikey":
		header := p.config.Security.APIKeyHeader
		if header == "" {
			header = tool.Security.HeaderName
		}
		if header == "" {
			header = "X-API-Key"
		}
		req.Header.Set(header, token)
	case "basic":
		req.Header.Set("Authorization", "Basic "+token)
	default:
		req.Header.Set("Authorization", "Bearer "+token)
	}
}
