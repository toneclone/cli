package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Constants for the API client
const (
	APIVersion     = "v1"
	DefaultBaseURL = "https://api.toneclone.ai"
	DefaultTimeout = 30 * time.Second
)

// Client represents the ToneClone API client
type Client struct {
	baseURL    string
	apiKey     string
	devAuthKey string
	httpClient *http.Client
	userAgent  string
	timeout    time.Duration
}

// ClientOption represents a configuration option for the client
type ClientOption func(*Client)

// NewClient creates a new ToneClone API client
func NewClient(apiKey string, options ...ClientOption) *Client {
	client := &Client{
		baseURL:    DefaultBaseURL,
		apiKey:     apiKey,
		devAuthKey: os.Getenv("TONECLONE_DEV_AUTH"),
		userAgent:  fmt.Sprintf("toneclone-cli/%s", APIVersion),
		timeout:    DefaultTimeout,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}

	// Apply options
	for _, option := range options {
		option(client)
	}

	return client
}

// WithBaseURL sets a custom base URL
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = strings.TrimSuffix(baseURL, "/")
	}
}

// WithTimeout sets a custom timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
		c.httpClient.Timeout = timeout
	}
}

// WithUserAgent sets a custom user agent
func WithUserAgent(userAgent string) ClientOption {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// APIResponse represents a generic API response
type APIResponse[T any] struct {
	Data   T      `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
	Status int    `json:"status,omitempty"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	ErrorMsg string `json:"error"`
	Message  string `json:"message,omitempty"`
	Code     string `json:"code,omitempty"`
}

// RateLimitError represents a rate limiting error with retry information
type RateLimitError struct {
	ErrorResponse
	RemainingRequests int
	ResetTime         time.Time
	RetryAfterSeconds int
}

func (e *RateLimitError) Error() string {
	if e.RetryAfterSeconds > 0 {
		return fmt.Sprintf("Rate limit exceeded. Try again in %d seconds", e.RetryAfterSeconds)
	}
	return fmt.Sprintf("Rate limit exceeded: %s", e.ErrorMsg)
}

func (e ErrorResponse) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.ErrorMsg, e.Message)
	}
	return e.ErrorMsg
}

// makeRequest performs an HTTP request to the API
func (c *Client) makeRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	u, err := c.requestURL(path)
	if err != nil {
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	// Prepare request body
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.applyCommonHeaders(req, "application/json")

	// Perform request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func (c *Client) requestURL(path string) (string, error) {
	// url.JoinPath treats '?' as path data, so split query strings supplied by
	// resource clients before joining the URL path.
	pathOnly, rawQuery, _ := strings.Cut(path, "?")
	u, err := url.JoinPath(c.baseURL, pathOnly)
	if err != nil {
		return "", err
	}
	if rawQuery != "" {
		u += "?" + rawQuery
	}
	return u, nil
}

func (c *Client) applyCommonHeaders(req *http.Request, contentType string) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.devAuthKey != "" {
		req.Header.Set("X-Dev-Auth", c.devAuthKey)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("TC-API-Version", APIVersion)
}

// doRequest performs a request and handles the response
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	resp, err := c.makeRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return handleResponse(resp, result)
}

func handleResponse(resp *http.Response, result interface{}) error {
	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var errorResp ErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err != nil {
			// If we can't parse the error response, return a generic error
			return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
		}

		// Handle rate limiting specifically
		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitErr := &RateLimitError{
				ErrorResponse: errorResp,
			}

			// Parse rate limiting headers
			if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
				if val, err := strconv.Atoi(remaining); err == nil {
					rateLimitErr.RemainingRequests = val
				}
			}

			if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
				if timestamp, err := strconv.ParseInt(reset, 10, 64); err == nil {
					rateLimitErr.ResetTime = time.Unix(timestamp, 0)
				}
			}

			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if val, err := strconv.Atoi(retryAfter); err == nil {
					rateLimitErr.RetryAfterSeconds = val
				}
			}

			return rateLimitErr
		}

		return errorResp
	}

	// Parse successful response
	if result != nil {
		// Handle empty response body case
		if len(respBody) == 0 {
			// For empty responses, we don't need to unmarshal anything
			return nil
		}
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	return c.doRequestWithRetry(ctx, "GET", path, nil, result)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequestWithRetry(ctx, "POST", path, body, result)
}

// PostMultipart performs a POST request with a caller-provided multipart body.
func (c *Client) PostMultipart(ctx context.Context, path, contentType string, body []byte, result interface{}) error {
	return c.doRawRequestWithRetry(ctx, http.MethodPost, path, contentType, body, result)
}

func (c *Client) makeRawRequest(ctx context.Context, method, path, contentType string, body []byte) (*http.Response, error) {
	u, err := c.requestURL(path)
	if err != nil {
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.applyCommonHeaders(req, contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}

func (c *Client) doRawRequest(ctx context.Context, method, path, contentType string, body []byte, result interface{}) error {
	resp, err := c.makeRawRequest(ctx, method, path, contentType, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return handleResponse(resp, result)
}

func (c *Client) doRawRequestWithRetry(ctx context.Context, method, path, contentType string, body []byte, result interface{}) error {
	return withRateLimitRetry(ctx, func() error {
		return c.doRawRequest(ctx, method, path, contentType, body, result)
	})
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequestWithRetry(ctx, "PUT", path, body, result)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.doRequestWithRetry(ctx, "DELETE", path, nil, nil)
}

// Patch performs a PATCH request
func (c *Client) Patch(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.doRequest(ctx, "PATCH", path, body, result)
}

// Health checks the API health
func (c *Client) Health(ctx context.Context) error {
	// Use a simple GET request that doesn't require authentication
	resp, err := c.makeRequest(ctx, "GET", "/ping", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("health check failed with status %d", resp.StatusCode)
}

// ValidateAPIKey validates that the API key is working
func (c *Client) ValidateAPIKey(ctx context.Context) error {
	// Try to make a simple authenticated request
	return c.Get(ctx, "/user", nil)
}

// GetBaseURL returns the configured base URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// GetUserAgent returns the configured user agent
func (c *Client) GetUserAgent() string {
	return c.userAgent
}

// SetTimeout updates the client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// WithContext returns a non-nil context. Request timeouts are enforced by the
// underlying HTTP client; callers that need context cancellation should derive
// and cancel their own context.
func (c *Client) WithContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

// doRequestWithRetry performs a request with automatic retry for rate limits
func (c *Client) doRequestWithRetry(ctx context.Context, method, endpoint string, body interface{}, result interface{}) error {
	return withRateLimitRetry(ctx, func() error {
		return c.doRequest(ctx, method, endpoint, body, result)
	})
}

func withRateLimitRetry(ctx context.Context, do func() error) error {
	maxRetries := 3
	baseDelay := time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := do()

		// Check if it's a rate limit error
		if rateLimitErr, ok := err.(*RateLimitError); ok {
			// Don't retry on the last attempt
			if attempt == maxRetries-1 {
				return rateLimitErr
			}

			delay := retryDelay(rateLimitErr, attempt, baseDelay)

			// Wait before retrying
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue // Retry
			}
		}

		// If it's not a rate limit error, return immediately
		return err
	}

	return fmt.Errorf("max retries exceeded")
}

func retryDelay(rateLimitErr *RateLimitError, attempt int, baseDelay time.Duration) time.Duration {
	delay := baseDelay * time.Duration(1<<attempt)
	if rateLimitErr != nil && rateLimitErr.RetryAfterSeconds > 0 {
		delay = time.Duration(rateLimitErr.RetryAfterSeconds) * time.Second
	}
	if delay > 60*time.Second {
		return 60 * time.Second
	}
	return delay
}
