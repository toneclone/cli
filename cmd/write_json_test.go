package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// newWriteTestServer serves a persona, two knowledge cards, and /query. When
// the query asks for n>1 it returns a drafts envelope; otherwise plain content.
func newWriteTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/personas/p1"):
			_, _ = w.Write([]byte(`{"personaId":"p1","name":"Writer"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/knowledge/k1"):
			_, _ = w.Write([]byte(`{"knowledgeCardId":"k1","name":"One"}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/knowledge/k2"):
			_, _ = w.Write([]byte(`{"knowledgeCardId":"k2","name":"Two"}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/query"):
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if n, _ := body["n"].(float64); n > 1 {
				_, _ = w.Write([]byte(`{"variants":[{"content":"draft one","index":0},{"content":"draft two","index":1}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"content":"hello world"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// runWriteCapture runs `write` against srv with the given knowledge/drafts/JSON
// settings and returns stdout.
func runWriteCapture(t *testing.T, knowledge string, drafts int, globalJSON bool, outputFlag string) string {
	t.Helper()
	srv := newWriteTestServer(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("TONECLONE_API_KEY", "tc_test_0123456789abcdef")
	t.Setenv("TONECLONE_BASE_URL", srv.URL)

	oldPersona, oldKnowledge, oldPrompt, oldFile := writePersona, writeKnowledge, writePrompt, writeFile
	oldOutput, oldDrafts, oldTimeout, oldVerbose, oldReview := writeOutput, writeDrafts, writeTimeout, writeVerbose, writeReviewLink
	oldJSON := jsonOutput
	t.Cleanup(func() {
		writePersona, writeKnowledge, writePrompt, writeFile = oldPersona, oldKnowledge, oldPrompt, oldFile
		writeOutput, writeDrafts, writeTimeout, writeVerbose, writeReviewLink = oldOutput, oldDrafts, oldTimeout, oldVerbose, oldReview
		jsonOutput = oldJSON
	})
	writePersona, writeKnowledge, writePrompt, writeFile = "p1", knowledge, "say hi", ""
	writeOutput, writeDrafts, writeTimeout, writeVerbose, writeReviewLink = outputFlag, drafts, 5, false, false
	jsonOutput = globalJSON

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	writeCmd.SetContext(context.Background())
	runErr := runWrite(writeCmd, nil)
	w.Close()
	os.Stdout = oldStdout
	out, _ := io.ReadAll(r)
	if runErr != nil {
		t.Fatalf("runWrite: %v", runErr)
	}
	return string(out)
}

// Regression guard for the monorepo fix "honor --json/--output json when
// generating with multiple profiles": every multi-item write path (multiple
// knowledge cards, multiple drafts) must emit JSON under --json and the hidden
// --output=json alias.
func TestWriteJSONOutputHonoredForMultiItemPaths(t *testing.T) {
	modes := []struct {
		name       string
		globalJSON bool
		output     string
	}{
		{"--json", true, ""},
		{"--output=json", false, "json"},
	}
	for _, mode := range modes {
		t.Run("single card "+mode.name, func(t *testing.T) {
			out := runWriteCapture(t, "k1", 1, mode.globalJSON, mode.output)
			assertWriteTextJSON(t, out)
		})
		t.Run("multiple cards "+mode.name, func(t *testing.T) {
			out := runWriteCapture(t, "k1,k2", 1, mode.globalJSON, mode.output)
			assertWriteTextJSON(t, out)
		})
		t.Run("multiple drafts "+mode.name, func(t *testing.T) {
			out := runWriteCapture(t, "k1,k2", 2, mode.globalJSON, mode.output)
			var parsed struct {
				Drafts []struct {
					Content string `json:"content"`
				} `json:"drafts"`
			}
			if err := json.Unmarshal([]byte(out), &parsed); err != nil {
				t.Fatalf("expected JSON drafts output, got %q: %v", out, err)
			}
			if len(parsed.Drafts) != 2 || parsed.Drafts[0].Content != "draft one" {
				t.Fatalf("unexpected drafts JSON: %q", out)
			}
		})
	}

	t.Run("multiple cards plain text without json", func(t *testing.T) {
		out := runWriteCapture(t, "k1,k2", 1, false, "")
		if out != "hello world\n" {
			t.Fatalf("expected plain text output, got %q", out)
		}
	})
}

func assertWriteTextJSON(t *testing.T, out string) {
	t.Helper()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", out, err)
	}
	if parsed["text"] != "hello world" {
		t.Fatalf("text = %v, want hello world", parsed["text"])
	}
}
