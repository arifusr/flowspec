package runtime

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPRequest represents a resolved HTTP request ready to send.
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    []byte
	Timeout time.Duration
}

// HTTPResponse represents the result of an HTTP call.
type HTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
	Size       int64
}

// HTTPClient sends HTTP requests.
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient creates a new HTTP client.
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Send executes an HTTP request and returns the response.
func (c *HTTPClient) Send(req *HTTPRequest) (*HTTPResponse, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		bodyReader = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, val := range req.Headers {
		httpReq.Header.Set(key, val)
	}

	start := time.Now()
	resp, err := c.client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
		Duration:   duration,
		Size:       int64(len(body)),
	}, nil
}

// BuildRequest constructs an HTTPRequest from AST data and variables.
func BuildRequest(method, rawURL string, headers map[string]string,
	queries map[string]string, bodyType string, bodyFields map[string]string,
	vars *Variables, timeout time.Duration) *HTTPRequest {

	resolvedURL := vars.Interpolate(rawURL)

	// Add query params
	if len(queries) > 0 {
		u, err := url.Parse(resolvedURL)
		if err == nil {
			q := u.Query()
			for k, v := range queries {
				q.Set(k, vars.Interpolate(v))
			}
			u.RawQuery = q.Encode()
			resolvedURL = u.String()
		}
	}

	resolvedHeaders := make(map[string]string)
	for k, v := range headers {
		resolvedHeaders[k] = vars.Interpolate(v)
	}

	var body []byte
	if bodyType == "json" && len(bodyFields) > 0 {
		jsonMap := make(map[string]interface{})
		for k, v := range bodyFields {
			resolved := vars.Interpolate(v)
			// Try to parse as number/bool
			jsonMap[k] = tryParseValue(resolved)
		}
		body, _ = json.Marshal(jsonMap)
		if _, exists := resolvedHeaders["Content-Type"]; !exists {
			resolvedHeaders["Content-Type"] = "application/json"
		}
	} else if bodyType == "form" && len(bodyFields) > 0 {
		form := url.Values{}
		for k, v := range bodyFields {
			form.Set(k, vars.Interpolate(v))
		}
		body = []byte(form.Encode())
		if _, exists := resolvedHeaders["Content-Type"]; !exists {
			resolvedHeaders["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}

	return &HTTPRequest{
		Method:  method,
		URL:     resolvedURL,
		Headers: resolvedHeaders,
		Body:    body,
		Timeout: timeout,
	}
}

func tryParseValue(s string) interface{} {
	// Only coerce explicit boolean keywords
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	// Check if value is a JSON array or object (from transform variables)
	trimmed := strings.TrimSpace(s)
	if (strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) ||
		(strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) {
		var parsed interface{}
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			return parsed
		}
	}
	// Everything else stays as string — no implicit numeric coercion.
	// If a value looks like a number but was written in quotes ("320301"),
	// it should remain a JSON string to match strict-typed backends.
	return s
}
