package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/sertdev/pxbin/internal/resilience"
)

// UpstreamOpts configures resilience for upstream clients.
type UpstreamOpts struct {
	CBOpts    resilience.CircuitBreakerOpts
	RetryOpts resilience.RetryOpts
}

// UpstreamClient sends requests to an OpenAI-compatible upstream API.
type UpstreamClient struct {
	client    *http.Client
	baseURL   string
	apiKey    string
	cb        *resilience.CircuitBreaker
	retryOpts resilience.RetryOpts
}

// NewUpstreamClient creates an UpstreamClient with a configured transport for
// connection pooling and keep-alive, plus optional circuit breaker and retry.
func NewUpstreamClient(baseURL, apiKey string, opts *UpstreamOpts) *UpstreamClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     0, // unlimited
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  true, // avoid unnecessary decompress/recompress for passthrough
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	uc := &UpstreamClient{
		client: &http.Client{
			Transport: transport,
			Timeout:   0, // no global timeout; streaming can be long-lived
		},
		baseURL: baseURL,
		apiKey:  apiKey,
	}

	if opts != nil {
		uc.cb = resilience.NewCircuitBreaker(opts.CBOpts)
		uc.retryOpts = opts.RetryOpts
	}

	return uc
}

// Do sends a request to the upstream and returns the response. The caller is
// responsible for closing the response body. Uses circuit breaker and retry
// for connection errors on non-streaming requests.
func (c *UpstreamClient) Do(ctx context.Context, method, path string, body io.Reader, headers http.Header) (*http.Response, error) {
	return c.doRequest(ctx, method, path, body, headers, true)
}

// DoRaw sends a request to the upstream without setting Authorization: Bearer.
// The caller provides all necessary auth headers. Uses circuit breaker and retry.
func (c *UpstreamClient) DoRaw(ctx context.Context, method, path string, body io.Reader, headers http.Header) (*http.Response, error) {
	return c.doRequest(ctx, method, path, body, headers, false)
}

func (c *UpstreamClient) doRequest(ctx context.Context, method, path string, body io.Reader, headers http.Header, useBearer bool) (*http.Response, error) {
	// Check circuit breaker.
	var cbDone func(bool)
	if c.cb != nil {
		var err error
		cbDone, err = c.cb.Allow()
		if err != nil {
			return nil, fmt.Errorf("upstream unavailable: %w", err)
		}
	}

	// Body may be a ReadSeeker — needed for retry to re-read the body.
	bodySeeker, canRetry := body.(io.ReadSeeker)

	var resp *http.Response
	var lastErr error

	doOnce := func() error {
		url := c.baseURL + path
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		if useBearer {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}
		req.Header.Set("Content-Type", "application/json")

		for k, vals := range headers {
			for _, v := range vals {
				if useBearer {
					req.Header.Add(k, v)
				} else {
					req.Header.Set(k, v)
				}
			}
		}

		resp, err = c.client.Do(req)
		return err
	}

	// If retry is configured and body supports seeking, wrap in retry.
	if c.retryOpts.MaxAttempts > 1 && canRetry {
		lastErr = resilience.Do(ctx, c.retryOpts, func() error {
			if _, err := bodySeeker.Seek(0, io.SeekStart); err != nil {
				return err
			}
			return doOnce()
		})
	} else {
		lastErr = doOnce()
	}

	// Report to circuit breaker.
	if cbDone != nil {
		cbDone(lastErr == nil)
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return resp, nil
}
