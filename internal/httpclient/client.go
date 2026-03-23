package httpclient

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// Client wraps an http.Client to provide rate limiting.
type Client struct {
	lastRequest        time.Time
	httpClient         *http.Client
	minRequestInterval time.Duration
	mu                 sync.Mutex
}

// NewClient creates a new rate-limited HTTP client.
func NewClient(httpClient *http.Client, minRequestInterval time.Duration) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     30 * time.Second,
				TLSHandshakeTimeout: 5 * time.Second,
			},
		}
	}
	return &Client{
		httpClient:         httpClient,
		minRequestInterval: minRequestInterval,
	}
}

// Do executes an HTTP request with rate-limiting. No retries - failures are returned immediately.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	c.mu.Lock()
	now := time.Now()
	nextAllowed := c.lastRequest.Add(c.minRequestInterval)
	var waitTime time.Duration
	if now.Before(nextAllowed) {
		waitTime = nextAllowed.Sub(now)
		c.lastRequest = nextAllowed
	} else {
		c.lastRequest = now
	}
	c.mu.Unlock()

	if waitTime > 0 {
		timer := time.NewTimer(waitTime)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	reqClone := req.Clone(ctx)
	return c.httpClient.Do(reqClone)
}

// GetUnderlyingClient returns the underlying *http.Client.
func (c *Client) GetUnderlyingClient() *http.Client {
	return c.httpClient
}
