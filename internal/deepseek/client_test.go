package deepseek

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := New("test-key", srv.URL)
	client.Sleep = func(context.Context, time.Duration) error { return nil }
	return client
}

func TestSearchSuccessAndEmpty(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/v1/tools/search" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("authorization %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type %q", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"rerank_score":2.4,"title":"DeepSeek","url":"https://example.com","snippet":"short","summary":"long","content":"page body that is long","site_name":"Example","site_icon":null,"date":"2026-06-01T00:00:00+00:00","authority":3}]}`))
	})

	limit := 9
	out, text, err := client.SearchTool(context.Background(), SearchInput{
		Query:           "DeepSeek",
		NeedContent:     true,
		MaxContentChars: &limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("hits %d", hits.Load())
	}
	if len(out.Results) != 1 || !out.Results[0].ContentTruncated {
		t.Fatalf("output %+v", out.Results)
	}
	if !strings.Contains(out.Results[0].Content, "truncated at 9") {
		t.Fatalf("content %q", out.Results[0].Content)
	}
	if out.Results[0].SiteName == nil || *out.Results[0].SiteName != "Example" {
		t.Fatalf("site %+v", out.Results[0].SiteName)
	}
	if !strings.Contains(text, "https://example.com") {
		t.Fatalf("text %s", text)
	}

	empty := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	out, text, err = empty.SearchTool(context.Background(), SearchInput{Query: "nothing"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Results) != 0 || !strings.Contains(text, "No results found") {
		t.Fatalf("empty output %q %+v", text, out)
	}
}

func TestFetchSuccessTruncates(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tools/fetch" {
			t.Errorf("path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"title":"Example","content":"abcdefghij"}`))
	})
	limit := 4
	out, text, err := client.FetchTool(context.Background(), FetchInput{
		URL:      "https://example.com/a",
		MaxChars: &limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Truncated || out.OriginalChars != 10 || out.Title != "Example" {
		t.Fatalf("%+v", out)
	}
	if !strings.Contains(out.Content, "abcd") || !strings.Contains(out.Content, "truncated at 4") {
		t.Fatalf("content %q", out.Content)
	}
	if !strings.Contains(text, "Title: Example") || !strings.Contains(text, "URL: https://example.com/a") {
		t.Fatalf("text %s", text)
	}
}

func TestClientDoesNotRetry4xx(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key","type":"authentication_error","code":"invalid_api_key"}}`))
	})
	_, err := client.Search(context.Background(), SearchRequest{Query: "q", Count: 1})
	if err == nil {
		t.Fatal("expected error")
	}
	if hits.Load() != 1 {
		t.Fatalf("hits %d", hits.Load())
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type %T %v", err, err)
	}
	if apiErr.Status != http.StatusUnauthorized || !strings.Contains(err.Error(), "missing or invalid API key") {
		t.Fatalf("error %v", err)
	}

	hits.Store(0)
	bad := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"'count' must be between 1 and 50","type":"invalid_request_error","code":"invalid_request_error"}}`))
	})
	_, err = bad.Fetch(context.Background(), FetchRequest{URL: "https://example.com"})
	if err == nil || hits.Load() != 1 || !strings.Contains(err.Error(), "invalid request") {
		t.Fatalf("hits %d err %v", hits.Load(), err)
	}
}

func TestClientRetries429ThenSucceeds(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"slow down","code":"rate_limit"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"title":"ok","content":"body"}`))
	})
	out, err := client.Fetch(context.Background(), FetchRequest{URL: "https://example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 2 || out.Title != "ok" {
		t.Fatalf("hits %d out %+v", hits.Load(), out)
	}
}

func TestClientRetries5xx(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"message":"boom","code":"internal"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	if _, err := client.Search(context.Background(), SearchRequest{Query: "q", Count: 1}); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 3 {
		t.Fatalf("hits %d", hits.Load())
	}
}

func TestMissingAPIKey(t *testing.T) {
	client := New("", DefaultBaseURL)
	_, err := client.Fetch(context.Background(), FetchRequest{URL: "https://example.com"})
	if err == nil || !strings.Contains(err.Error(), "DEEPSEEK_API_KEY") {
		t.Fatal(err)
	}
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("DEEPSEEK_BASE_URL", "")
	if _, err := NewFromEnv(); err == nil {
		t.Fatal("expected missing key error")
	}
}
