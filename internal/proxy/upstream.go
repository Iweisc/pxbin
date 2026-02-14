package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// UpstreamClient sends requests to an OpenAI-compatible upstream API.
type UpstreamClient struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

// NewUpstreamClient creates an UpstreamClient with a configured transport for
// connection pooling and keep-alive.
func NewUpstreamClient(baseURL, apiKey string) *UpstreamClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     0, // unlimited
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	return &UpstreamClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   0, // no global timeout; streaming can be long-lived
		},
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// Do sends a request to the upstream and returns the response. The caller is
// responsible for closing the response body.
func (c *UpstreamClient) Do(ctx context.Context, method, path string, body io.Reader, headers http.Header) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	// Copy any extra headers from the caller.
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	return c.client.Do(req)
}

// DoRaw sends a request to the upstream without setting Authorization: Bearer.
// The caller provides all necessary auth headers. The caller is responsible
// for closing the response body.
func (c *UpstreamClient) DoRaw(ctx context.Context, method, path string, body io.Reader, headers http.Header) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Set(k, v)
		}
	}

	return c.client.Do(req)
}
