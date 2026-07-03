package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/toneclone/cli/pkg/client"
)

// jsonOutput is the global --json flag. When set, successful command output and
// errors are emitted as JSON so agents get structured, parseable results.
var jsonOutput bool

// errorEnvelope is the structured error shape emitted in --json mode so agents
// can branch on a stable code instead of parsing free text.
type errorEnvelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
	DocsURL   string `json:"docsUrl,omitempty"`
}

// JSON renders the envelope wrapped under a top-level "error" key.
func (e errorEnvelope) JSON() (string, error) {
	b, err := json.MarshalIndent(struct {
		Error errorEnvelope `json:"error"`
	}{Error: e}, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// classifyError maps a Go error into a stable, agent-friendly envelope. It
// unwraps wrapped errors (fmt.Errorf %w) via errors.As so typed client errors
// are classified even when a command has added context.
func classifyError(err error) errorEnvelope {
	if err == nil {
		return errorEnvelope{Code: "error", Message: "unknown error"}
	}

	var rateLimitErr *client.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return errorEnvelope{
			Code:      "rate_limited",
			Message:   rateLimitErr.Error(),
			Retryable: true,
			DocsURL:   docsURL,
		}
	}

	var errResp client.ErrorResponse
	if errors.As(err, &errResp) {
		return classifyErrorResponse(errResp)
	}

	// Fall back to message inspection for plain errors that lost their type.
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "rate limit"):
		return errorEnvelope{Code: "rate_limited", Message: msg, Retryable: true, DocsURL: docsURL}
	case strings.Contains(lower, "authentication required"),
		strings.Contains(lower, "no api key"),
		strings.Contains(lower, "unauthorized"):
		return errorEnvelope{Code: "auth_required", Message: msg, Retryable: false, DocsURL: docsURL}
	}

	return errorEnvelope{Code: "error", Message: msg}
}

func classifyErrorResponse(e client.ErrorResponse) errorEnvelope {
	code := e.Code
	switch code {
	case "paywall", "payment_required", "plan_limit":
		code = "paywall"
	case "":
		code = "error"
	}
	return errorEnvelope{Code: code, Message: e.Error(), Retryable: false, DocsURL: docsURL}
}

// renderCommandError prints a command error as a structured JSON envelope (when
// --json is set) or as a human-readable message, and is the single place errors
// reach the terminal so cobra's own printing stays silenced.
func renderCommandError(err error) {
	if jsonOutput {
		if out, jerr := classifyError(err).JSON(); jerr == nil {
			fmt.Fprintln(os.Stderr, out)
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}
