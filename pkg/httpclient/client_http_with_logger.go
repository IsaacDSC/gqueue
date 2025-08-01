package httpclient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/IsaacDSC/webhook/pkg/ctxlogger"
)

type HTTPClientTransport struct {
	Transport http.RoundTripper
	ctx       context.Context
}

func NewHTTPClientTransport(ctx context.Context, transport http.RoundTripper) *HTTPClientTransport {
	if transport == nil {
		transport = &http.Transport{
			DisableKeepAlives:   false,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     90 * time.Second,
		}
	}
	return &HTTPClientTransport{
		Transport: transport,
		ctx:       ctx,
	}
}

func (t *HTTPClientTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	logger := ctxlogger.GetLogger(t.ctx)

	var reqBody string
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			logger.Error("Failed to read request body", "error", err)
		} else {
			reqBody = string(bodyBytes)
		}

		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	logger.Info("HTTP client request started",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", req.Header,
		"body", reqBody,
	)

	resp, err := t.Transport.RoundTrip(req)

	elapsed := time.Since(start)

	if err != nil {
		logger.Error("HTTP client request failed",
			"method", req.Method,
			"url", req.URL.String(),
			"error", err.Error(),
			"elapsed_time", elapsed,
		)
		return nil, err
	}

	var respBody string
	if resp.Body != nil {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error("Failed to read response body", "error", err)
		} else {
			respBody = string(bodyBytes)
		}

		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	logger.Info("HTTP client request completed",
		"method", req.Method,
		"url", req.URL.String(),
		"status_code", resp.StatusCode,
		"status", resp.Status,
		"response_headers", resp.Header,
		"response_body", respBody,
		"elapsed_time", elapsed,
	)

	return resp, nil
}

func NewHTTPClientWithLogging(ctx context.Context) *http.Client {
	transport := NewHTTPClientTransport(ctx, nil)
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}
