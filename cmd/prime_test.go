package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPrimeGuideJSONIsValidAndPopulated(t *testing.T) {
	out, err := primeJSON(primeGuide())
	if err != nil {
		t.Fatal(err)
	}
	var g primeGuideDoc
	if err := json.Unmarshal([]byte(out), &g); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if g.Product == "" || len(g.Commands) == 0 || len(g.Workflow) == 0 {
		t.Error("prime guide is missing core fields")
	}
}

func TestPrimeDocumentsCoreCommands(t *testing.T) {
	names := map[string]bool{}
	for _, c := range primeGuide().Commands {
		names[c.Name] = true
	}
	for _, want := range []string{"write", "personalize", "humanize", "personas list", "knowledge list", "quota"} {
		if !names[want] {
			t.Errorf("expected prime to document command %q", want)
		}
	}
}

func TestPrimeTextMentionsKeyConcepts(t *testing.T) {
	text := primeText(primeGuide())
	for _, want := range []string{"persona", "knowledge card", "quota", "toneclone write", "--json"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected prime text to mention %q", want)
		}
	}
}

func TestPrimeDoesNotReferenceOldTerms(t *testing.T) {
	text := primeText(primeGuide())
	for _, bad := range []string{"generate text", "profiles list", "--profile"} {
		if strings.Contains(text, bad) {
			t.Errorf("prime text should not reference stale term %q", bad)
		}
	}
}
