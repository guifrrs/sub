package opensubtitles

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultBaseURL    = "https://www.opensubtitles.org"
	defaultUserAgent  = "sub-cli/0.1"
	defaultLanguage   = "eng"
	defaultMaxRetries = 2
	maxBodyBytes      = 16 << 20
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
	maxRetries int
	retryDelay time.Duration
}

func NewClient() *Client {
	return &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		userAgent:  defaultUserAgent,
		maxRetries: defaultMaxRetries,
		retryDelay: 250 * time.Millisecond,
	}
}

func NewClientWithHTTP(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		baseURL:    defaultBaseURL,
		httpClient: httpClient,
		userAgent:  defaultUserAgent,
		maxRetries: defaultMaxRetries,
		retryDelay: 250 * time.Millisecond,
	}
}

func (c *Client) get(ctx context.Context, endpoint string, query url.Values) ([]byte, error) {
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse request url: %w", err)
	}

	u.RawQuery = query.Encode()
	return c.doGetWithRetry(ctx, u.String())
}

func (c *Client) doGetWithRetry(ctx context.Context, requestURL string) ([]byte, error) {
	var lastErr error

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		body, shouldRetry, err := c.doGetOnce(ctx, requestURL)
		if err == nil {
			return body, nil
		}

		lastErr = err
		if !shouldRetry || attempt == c.maxRetries {
			break
		}

		delay := c.retryDelay * time.Duration(1<<attempt)
		if sleepErr := sleepWithContext(ctx, delay); sleepErr != nil {
			return nil, sleepErr
		}
	}

	return nil, lastErr
}

func (c *Client) doGetOnce(ctx context.Context, requestURL string) ([]byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, false, fmt.Errorf("request canceled: %w", ctx.Err())
		}
		return nil, true, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	if isTransientStatus(resp.StatusCode) {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, true, fmt.Errorf("transient status %d", resp.StatusCode)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, false, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return nil, true, fmt.Errorf("read response body: %w", err)
	}

	if len(body) > maxBodyBytes {
		return nil, false, fmt.Errorf("response body too large: limit %d bytes", maxBodyBytes)
	}

	return body, false, nil
}

func normalizeLanguage(language string) string {
	if language == "" {
		return defaultLanguage
	}

	return language
}

func isTransientStatus(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return true
	}

	return statusCode >= 500 && statusCode <= 599
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("request canceled during retry backoff: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
