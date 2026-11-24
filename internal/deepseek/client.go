package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the DeepSeek API origin.
	DefaultBaseURL = "https://api.deepseek.com"
	searchPath     = "/v1/tools/search"
	fetchPath      = "/v1/tools/fetch"
	defaultTimeout = 60 * time.Second
	maxRetries     = 2
	maxBodyBytes   = 32 << 20
	maxRetryWait   = 30 * time.Second
)

// Client calls the DeepSeek Search and Fetch APIs.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	// MaxRetries is the number of extra attempts after the first request.
	// Retries apply to network errors and HTTP 408, 429, and 5xx.
	MaxRetries int
	// Sleep waits between retries. Tests replace it so they do not block.
	Sleep func(ctx context.Context, d time.Duration) error
}

// APIError is a non-success response from the DeepSeek API.
type APIError struct {
	Status  int
	Message string
	Code    string
}

func (e *APIError) Error() string {
	msg := strings.TrimSpace(e.Message)
	if msg == "" {
		msg = "request failed"
	}
	if e.Code != "" && !strings.Contains(msg, e.Code) {
		return fmt.Sprintf("deepseek api: HTTP %d %s: %s (%s)", e.Status, statusMeaning(e.Status), msg, e.Code)
	}
	return fmt.Sprintf("deepseek api: HTTP %d %s: %s", e.Status, statusMeaning(e.Status), msg)
}

// New builds a client. baseURL falls back to DefaultBaseURL.
func New(apiKey, baseURL string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL:    baseURL,
		APIKey:     strings.TrimSpace(apiKey),
		HTTPClient: &http.Client{Timeout: defaultTimeout},
		MaxRetries: maxRetries,
	}
}

// NewFromEnv reads DEEPSEEK_API_KEY and optional DEEPSEEK_BASE_URL.
func NewFromEnv() (*Client, error) {
	key := strings.TrimSpace(os.Getenv("DEEPSEEK_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY is not set")
	}
	return New(key, os.Getenv("DEEPSEEK_BASE_URL")), nil
}

// Search calls POST /v1/tools/search.
func (c *Client) Search(ctx context.Context, req SearchRequest) (SearchResponse, error) {
	var out SearchResponse
	if err := c.post(ctx, searchPath, req, &out); err != nil {
		return SearchResponse{}, err
	}
	if out.Results == nil {
		out.Results = []SearchResult{}
	}
	return out, nil
}

// Fetch calls POST /v1/tools/fetch.
func (c *Client) Fetch(ctx context.Context, req FetchRequest) (FetchResponse, error) {
	var out FetchResponse
	if err := c.post(ctx, fetchPath, req, &out); err != nil {
		return FetchResponse{}, err
	}
	return out, nil
}

func (c *Client) post(ctx context.Context, path string, payload any, dest any) error {
	if strings.TrimSpace(c.APIKey) == "" {
		return fmt.Errorf("DEEPSEEK_API_KEY is not set")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	var lastErr error
	attempts := c.MaxRetries
	if attempts < 0 {
		attempts = 0
	}
	for attempt := 0; attempt <= attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := c.http().Do(req)
		if err != nil {
			lastErr = fmt.Errorf("deepseek api: request failed: %w", err)
			if attempt == attempts {
				return lastErr
			}
			if err := c.sleep(ctx, retryAfter("", attempt)); err != nil {
				return err
			}
			continue
		}

		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("deepseek api: read response: %w", readErr)
			if attempt == attempts || !retryable(resp.StatusCode) {
				return lastErr
			}
			if err := c.sleep(ctx, retryAfter(resp.Header.Get("Retry-After"), attempt)); err != nil {
				return err
			}
			continue
		}
		if len(respBody) > maxBodyBytes {
			return fmt.Errorf("deepseek api: response exceeds %d bytes", maxBodyBytes)
		}

		if resp.StatusCode == http.StatusOK {
			if err := json.Unmarshal(respBody, dest); err != nil {
				return fmt.Errorf("deepseek api: decode response: %w", err)
			}
			return nil
		}

		apiErr := parseAPIError(resp.StatusCode, respBody)
		lastErr = apiErr
		if !retryable(resp.StatusCode) || attempt == attempts {
			return apiErr
		}
		if err := c.sleep(ctx, retryAfter(resp.Header.Get("Retry-After"), attempt)); err != nil {
			return err
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("deepseek api: request failed")
	}
	return lastErr
}

func (c *Client) http() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) sleep(ctx context.Context, d time.Duration) error {
	if c.Sleep != nil {
		return c.Sleep(ctx, d)
	}
	return sleepContext(ctx, d)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func retryable(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500
}

func retryAfter(header string, attempt int) time.Duration {
	fallback := time.Duration(200*(1<<attempt)) * time.Millisecond
	header = strings.TrimSpace(header)
	if header == "" {
		return fallback
	}
	if secs, err := strconv.Atoi(header); err == nil {
		if secs < 0 {
			return 0
		}
		d := time.Duration(secs) * time.Second
		if d > maxRetryWait {
			return maxRetryWait
		}
		return d
	}
	if when, err := http.ParseTime(header); err == nil {
		d := time.Until(when)
		if d < 0 {
			return 0
		}
		if d > maxRetryWait {
			return maxRetryWait
		}
		return d
	}
	return fallback
}

type apiErrorEnvelope struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func parseAPIError(status int, body []byte) *APIError {
	var env apiErrorEnvelope
	_ = json.Unmarshal(body, &env)
	msg := strings.TrimSpace(env.Error.Message)
	if msg == "" {
		msg = strings.TrimSpace(string(body))
		if utf8Len(msg) > 300 {
			msg = truncateRunes(msg, 300)
		}
	}
	code := env.Error.Code
	if code == "" {
		code = env.Error.Type
	}
	return &APIError{Status: status, Message: msg, Code: code}
}

func statusMeaning(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "invalid request"
	case http.StatusUnauthorized:
		return "missing or invalid API key"
	case http.StatusPaymentRequired:
		return "insufficient balance"
	case http.StatusForbidden:
		return "rejected by security policy"
	case http.StatusNotFound:
		return "page content unavailable"
	case http.StatusRequestTimeout:
		return "request timed out"
	case http.StatusRequestEntityTooLarge:
		return "request body too large"
	case http.StatusTooManyRequests:
		return "rate limit exceeded"
	case http.StatusInternalServerError:
		return "internal server error"
	case http.StatusGatewayTimeout:
		return "downstream timeout"
	default:
		if status >= 500 {
			return "server error"
		}
		return "request failed"
	}
}
