package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cesargomez89/navidrums/internal/constants"
)

// Client wraps an http.Client to provide rate limiting and automatic retries.
type Client struct {
	httpClient *http.Client

	minRequestInterval time.Duration
	lastRequest        time.Time
	mu                 sync.Mutex
}

// NewClient creates a new rate-limited, retrying HTTP client.
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

// Do executes an HTTP request with rate-limiting and retries.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < constants.DefaultRetryCount; attempt++ {
		// Check context before claiming a time slot
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

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

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
		} else if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := parseRetryAfter(resp)
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("rate limited (status %d)", resp.StatusCode)

			backoffWait := time.Duration(attempt+1) * constants.DefaultRetryBase
			if retryAfter > backoffWait {
				backoffWait = retryAfter
			}
			if retryAfter > 0 {
				c.mu.Lock()
				next := time.Now().Add(retryAfter)
				if c.lastRequest.Before(next) {
					c.lastRequest = next
				}
				c.mu.Unlock()
			}

			backoffTimer := time.NewTimer(backoffWait)
			select {
			case <-ctx.Done():
				backoffTimer.Stop()
				return nil, ctx.Err()
			case <-backoffTimer.C:
			}
			continue
		} else {
			return resp, nil
		}

		backoffWait := time.Duration(attempt+1) * constants.DefaultRetryBase
		backoffTimer := time.NewTimer(backoffWait)
		select {
		case <-ctx.Done():
			backoffTimer.Stop()
			return nil, ctx.Err()
		case <-backoffTimer.C:
		}
	}
	return nil, lastErr
}

// GetUnderlyingClient returns the underlying *http.Client.
func (c *Client) GetUnderlyingClient() *http.Client {
	return c.httpClient
}

// parseRetryAfter reads a Retry-After header and returns the duration to wait.
func parseRetryAfter(resp *http.Response) time.Duration {
	ra := resp.Header.Get("Retry-After")
	if ra == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(ra); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(ra); err == nil {
		return time.Until(t)
	}
	return 0
}
