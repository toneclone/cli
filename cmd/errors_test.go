package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/toneclone/cli/pkg/client"
)

func TestClassifyErrorCodes(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantCode      string
		wantRetryable bool
	}{
		{"nil", nil, "error", false},
		{"generic", errors.New("boom"), "error", false},
		{"auth message", errors.New("authentication required: no API key configured"), "auth_required", false},
		{"rate limit message", errors.New("Rate limit exceeded. Try again in 5 seconds"), "rate_limited", true},
		{"typed rate limit", &client.RateLimitError{RetryAfterSeconds: 3}, "rate_limited", true},
		{"typed paywall", client.ErrorResponse{ErrorMsg: "nope", Code: "paywall"}, "paywall", false},
		{"typed plan_limit", client.ErrorResponse{ErrorMsg: "nope", Code: "plan_limit"}, "paywall", false},
		{"typed unknown code", client.ErrorResponse{ErrorMsg: "nope", Code: ""}, "error", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := classifyError(tt.err)
			if env.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", env.Code, tt.wantCode)
			}
			if env.Retryable != tt.wantRetryable {
				t.Errorf("retryable = %v, want %v", env.Retryable, tt.wantRetryable)
			}
		})
	}
}

func TestClassifyErrorUnwrapsWrapped(t *testing.T) {
	rl := fmt.Errorf("text generation failed: %w", &client.RateLimitError{RetryAfterSeconds: 3})
	if env := classifyError(rl); env.Code != "rate_limited" || !env.Retryable {
		t.Errorf("wrapped rate limit: got %+v", env)
	}

	pw := fmt.Errorf("failed to get quota: %w", client.ErrorResponse{ErrorMsg: "x", Code: "paywall"})
	if env := classifyError(pw); env.Code != "paywall" || env.Retryable {
		t.Errorf("wrapped paywall: got %+v", env)
	}
}

func TestErrorEnvelopeJSONShape(t *testing.T) {
	out, err := errorEnvelope{Code: "paywall", Message: "m", Retryable: false, DocsURL: "https://toneclone.ai"}.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Error struct {
			Code      string `json:"code"`
			Message   string `json:"message"`
			Retryable bool   `json:"retryable"`
			DocsURL   string `json:"docsUrl"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if parsed.Error.Code != "paywall" || parsed.Error.DocsURL != "https://toneclone.ai" {
		t.Errorf("unexpected envelope: %s", out)
	}
}
