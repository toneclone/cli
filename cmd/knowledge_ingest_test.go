package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/toneclone/cli/pkg/client"
)

func TestKnowledgeSourceCommandsRegistered(t *testing.T) {
	for _, name := range []string{"create-from-url", "create-from-file", "sources"} {
		found := false
		for _, c := range knowledgeCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected knowledge %s command to be registered", name)
		}
	}
}

func TestKnowledgeCreateAndUpdateHonorGlobalJSON(t *testing.T) {
	oldJSON := jsonOutput
	oldFormat := knowledgeFormat
	oldStdout := os.Stdout
	t.Cleanup(func() {
		jsonOutput = oldJSON
		knowledgeFormat = oldFormat
		os.Stdout = oldStdout
	})

	jsonOutput = true
	knowledgeFormat = "table"
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	err = outputKnowledgeCardCreated(&client.KnowledgeCard{KnowledgeCardID: "card-1", Name: "Facts", Instructions: "Use facts."})
	write.Close()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]interface{}
	if err := json.NewDecoder(read).Decode(&got); err != nil {
		t.Fatalf("expected JSON output, got decode error %v", err)
	}
	if got["knowledgeCardId"] != "card-1" {
		t.Fatalf("unexpected JSON output: %+v", got)
	}
}

func TestKnowledgeIngestTextOutputIncludesSourceSummaryAndWarnings(t *testing.T) {
	oldJSON := jsonOutput
	oldFormat := knowledgeFormat
	oldStdout := os.Stdout
	t.Cleanup(func() {
		jsonOutput = oldJSON
		knowledgeFormat = oldFormat
		os.Stdout = oldStdout
	})

	jsonOutput = false
	knowledgeFormat = "table"
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	err = outputKnowledgeIngestResponse(&client.KnowledgeCardIngestResponse{
		KnowledgeCard: client.KnowledgeCard{KnowledgeCardID: "card-1", Name: "Facts"},
		Source:        client.KnowledgeCardSource{Type: "url", URL: "https://example.com", ExtractedCharCount: 123},
		Synthesis:     client.KnowledgeCardSynthesis{Summary: "A summary", KeyFacts: []string{"Fact"}, Warnings: []string{"Check this"}},
	})
	write.Close()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(read); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"Source: url https://example.com", "Summary:", "- Fact", "Warnings:", "- Check this"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output %q", want, out)
		}
	}
}

func TestValidateKnowledgeSourceURLRejectsUnsafeInputs(t *testing.T) {
	for _, raw := range []string{
		"ftp://example.com/file",
		"https://user:pass@example.com/private",
		"http://localhost/page",
		"http://127.0.0.1/page",
		"http://10.0.0.1/page",
	} {
		if err := validateKnowledgeSourceURL(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
	if err := validateKnowledgeSourceURL("https://example.com/page"); err != nil {
		t.Fatalf("expected public https URL to be allowed: %v", err)
	}
}

func TestSanitizeURLForOutputRedactsSensitiveParams(t *testing.T) {
	got := sanitizeURLForOutput("https://example.com/path?token=secret&ok=yes&signature=abc")
	if strings.Contains(got, "secret") || strings.Contains(got, "abc") {
		t.Fatalf("expected sensitive query params redacted, got %q", got)
	}
	if !strings.Contains(got, "ok=yes") {
		t.Fatalf("expected non-sensitive params preserved, got %q", got)
	}
}
