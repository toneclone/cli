package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Ported from toneclone/toneclone 62685b203. The backend's monthly draft and
// critique quotas answer 429 with an upgrade_url and no Retry-After. That is a
// plan limit, not transient rate limiting: it must not be retried, and the
// error must keep the quota message and upgrade link (and classify as a plan
// limit) instead of becoming a generic *RateLimitError.
func TestQuota429WithoutRetryAfterIsAPlanLimit(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"Monthly draft limit exceeded","details":{"monthly_limit":50},"upgrade_url":"https://app.example.com/billing"}`))
	}))
	defer server.Close()

	c := NewToneCloneClientFromConfig(server.URL, "tc_test_key", 10*time.Second)
	_, err := c.Generate.Text(context.Background(), &GenerateTextRequest{Prompt: "hi", PersonaID: "p1"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("quota 429 must not be retried; got %d calls", calls)
	}
	var rl *RateLimitError
	if errors.As(err, &rl) {
		t.Fatalf("quota 429 without Retry-After must not be a *RateLimitError: %v", err)
	}
	var apiErr ErrorResponse
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected an ErrorResponse, got %T: %v", err, err)
	}
	if apiErr.Code != "plan_limit" {
		t.Errorf("Code = %q, want plan_limit", apiErr.Code)
	}
	for _, want := range []string{"Monthly draft limit exceeded", "https://app.example.com/billing"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not include %q", err, want)
		}
	}
}

// A 429 from the rate limiters still carries Retry-After and stays a
// retryable *RateLimitError, even if its body happens to include upgrade_url.
func TestRateLimit429WithRetryAfterStaysRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"Rate limit exceeded. Please retry later.","upgrade_url":"https://app.example.com/billing"}`))
	}))
	defer server.Close()

	c := NewClient("tc_test_key", WithBaseURL(server.URL))
	err := c.doRequest(context.Background(), "POST", "/query", nil, nil)
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected *RateLimitError, got %T: %v", err, err)
	}
	if rl.RetryAfterSeconds != 3 {
		t.Errorf("RetryAfterSeconds = %d, want 3", rl.RetryAfterSeconds)
	}
}
