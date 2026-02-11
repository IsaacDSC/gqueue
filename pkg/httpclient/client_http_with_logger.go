package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/IsaacDSC/clienthttp"

	"github.com/IsaacDSC/gqueue/pkg/ctxlogger"
)

// LoggerAuditor implements clienthttp.Auditor using ctxlogger
type LoggerAuditor struct {
	ctx context.Context
}

func (a *LoggerAuditor) Log(ctx context.Context, req *clienthttp.AuditRequest, resp *clienthttp.AuditResponse) {
	logger := ctxlogger.GetLogger(a.ctx)

	logger.Info("HTTP client request started",
		"method", req.Method,
		"url", req.URL,
		"headers", req.Headers,
		"body", string(req.Body),
	)

	logger.Info("HTTP client request completed",
		"method", req.Method,
		"url", req.URL,
		"status_code", resp.StatusCode,
		"status", http.StatusText(resp.StatusCode),
		"response_headers", resp.Headers,
		"response_body", string(resp.Body),
	)
}

// ClientHTTPTransport implements http.RoundTripper using clienthttp
type ClientHTTPTransport struct {
	client *clienthttp.Client
	ctx    context.Context
}

func (t *ClientHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Read body if present
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	// Build headers map
	headers := make(map[string]string)
	for key := range req.Header {
		headers[key] = req.Header.Get(key)
	}

	// Build request options
	opts := make([]clienthttp.RequestOption, 0)
	if len(headers) > 0 {
		opts = append(opts, clienthttp.WithHeaders(headers))
	}

	// Execute request using clienthttp
	resp, err := t.client.Do(req.Context(), req.Method, req.URL.String(), body, opts...)
	if err != nil {
		return nil, err
	}

	// Convert to http.Response
	httpResp := &http.Response{
		StatusCode: resp.StatusCode,
		Status:     fmt.Sprintf("%d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
		Header:     resp.Headers,
		Body:       io.NopCloser(bytes.NewReader(resp.Body)),
		Request:    req,
	}

	return httpResp, nil
}

// NewHTTPClientWithLogging creates a new HTTP client with logging using clienthttp
func NewHTTPClientWithLogging(ctx context.Context) *http.Client {
	auditor := &LoggerAuditor{ctx: ctx}

	client, err := clienthttp.New("",
		clienthttp.WithTimeout(30*time.Second),
		clienthttp.WithMaxIdleConns(100),
		clienthttp.WithMaxIdleConnsPerHost(2),
		clienthttp.WithIdleConnTimeout(90*time.Second),
		clienthttp.WithAuditor(auditor),
	)
	if err != nil {
		// Fallback to standard http.Client
		return &http.Client{Timeout: 30 * time.Second}
	}

	return &http.Client{
		Transport: &ClientHTTPTransport{
			client: client,
			ctx:    ctx,
		},
		Timeout: 30 * time.Second,
	}
}

// HTTPClientTransport kept for backward compatibility (deprecated)
type HTTPClientTransport struct {
	Transport http.RoundTripper
	ctx       context.Context
}

// NewHTTPClientTransport creates a new HTTPClientTransport (deprecated, use NewHTTPClientWithLogging)
func NewHTTPClientTransport(ctx context.Context, transport http.RoundTripper) *HTTPClientTransport {
	return &HTTPClientTransport{
		Transport: transport,
		ctx:       ctx,
	}
}

func (t *HTTPClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Delegate to new implementation
	client := NewHTTPClientWithLogging(t.ctx)
	return client.Transport.RoundTrip(req)
}
