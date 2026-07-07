package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceTextPrefersTextOverFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.md")
	if err := os.WriteFile(path, []byte("file body"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := sourceText("inline", path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "inline" {
		t.Errorf("expected inline, got %q", got)
	}
}

func TestSourceTextReadsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "f.md")
	if err := os.WriteFile(path, []byte("file body"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := sourceText("", path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "file body" {
		t.Errorf("expected file body, got %q", got)
	}
}

func TestSourceTextMissingFile(t *testing.T) {
	if _, err := sourceText("", "/nonexistent/nope.md"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestVerbsRegistered(t *testing.T) {
	for _, name := range []string{"personalize", "humanize", "quota", "prime", "critique", "recipes"} {
		found := false
		for _, c := range rootCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q command registered on root", name)
		}
	}
}

func TestWriteHasDraftsFlag(t *testing.T) {
	f := writeCmd.Flags().Lookup("drafts")
	if f == nil {
		t.Fatal("expected --drafts flag on write")
	}
	if f.Shorthand != "n" {
		t.Errorf("expected -n shorthand, got %q", f.Shorthand)
	}
}

func TestWriteOutputFlagIsDeprecatedCompatibilityAlias(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() { jsonOutput = oldJSON })

	f := writeCmd.Flags().Lookup("output")
	if f == nil {
		t.Fatal("expected deprecated --output compatibility alias")
	}
	if !f.Hidden {
		t.Fatal("expected --output to be hidden from help")
	}
	if err := normalizeWriteOutput("json"); err != nil {
		t.Fatalf("expected --output json compatibility path, got %v", err)
	}
}

func TestValidateWriteDraftsRange(t *testing.T) {
	for _, drafts := range []int{1, 2, 5} {
		if err := validateWriteDrafts(drafts); err != nil {
			t.Errorf("validateWriteDrafts(%d) error = %v, want nil", drafts, err)
		}
	}
	for _, drafts := range []int{-1, 0, 6} {
		if err := validateWriteDrafts(drafts); err == nil {
			t.Errorf("validateWriteDrafts(%d) error = nil, want range error", drafts)
		}
	}
}

func TestWriteCommandRejectsInvalidDraftsBeforeAuth(t *testing.T) {
	oldDrafts := writeDrafts
	oldPersona := writePersona
	oldPrompt := writePrompt
	oldFlagValue := writeCmd.Flags().Lookup("drafts").Value.String()
	t.Cleanup(func() { writeDrafts = oldDrafts })
	t.Cleanup(func() {
		writePersona = oldPersona
		writePrompt = oldPrompt
		writeCmd.Flags().Set("drafts", oldFlagValue)
	})

	writePersona = "whatever"
	writePrompt = "hi"
	writeCmd.Flags().Set("drafts", "0")
	err := writeCmd.RunE(writeCmd, nil)
	if err == nil {
		t.Fatal("expected range error")
	}
	if !strings.Contains(err.Error(), "--drafts must be between 1 and 5") {
		t.Fatalf("expected drafts range error, got %v", err)
	}
	if strings.Contains(err.Error(), "authentication") {
		t.Fatalf("expected validation before auth, got %v", err)
	}
}

func TestWriteCommandParsesOutputAliasQuietlyBeforeDraftValidation(t *testing.T) {
	oldDrafts := writeDrafts
	oldPersona := writePersona
	oldPrompt := writePrompt
	oldOutput := writeOutput
	oldJSON := jsonOutput
	oldDraftFlag := writeCmd.Flags().Lookup("drafts").Value.String()
	oldOutputFlag := writeCmd.Flags().Lookup("output").Value.String()
	oldRootSilenceUsage := rootCmd.SilenceUsage
	oldRootSilenceErrors := rootCmd.SilenceErrors
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	rootCmd.SetErr(&stderr)
	rootCmd.SetOut(&stdout)
	rootCmd.SetArgs([]string{"write", "--persona", "whatever", "--prompt", "hi", "--drafts", "0", "--output", "json"})
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	t.Cleanup(func() {
		writeDrafts = oldDrafts
		writePersona = oldPersona
		writePrompt = oldPrompt
		writeOutput = oldOutput
		jsonOutput = oldJSON
		writeCmd.Flags().Set("drafts", oldDraftFlag)
		writeCmd.Flags().Set("output", oldOutputFlag)
		rootCmd.SetArgs(nil)
		rootCmd.SetErr(os.Stderr)
		rootCmd.SetOut(os.Stdout)
		rootCmd.SilenceUsage = oldRootSilenceUsage
		rootCmd.SilenceErrors = oldRootSilenceErrors
	})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected drafts range error")
	}
	if !strings.Contains(err.Error(), "--drafts must be between 1 and 5") {
		t.Fatalf("expected drafts range error, got %v", err)
	}
	if stderr.String() != "" {
		t.Fatalf("expected compatibility alias to be quiet on stderr, got %q", stderr.String())
	}
}

func TestExecuteRendersOutputAliasEarlyFailureAsJSON(t *testing.T) {
	oldArgs := os.Args
	oldJSON := jsonOutput
	oldRootSilenceUsage := rootCmd.SilenceUsage
	oldRootSilenceErrors := rootCmd.SilenceErrors
	rootCmd.SetArgs(nil)
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = stderrWriter
	t.Cleanup(func() {
		os.Args = oldArgs
		jsonOutput = oldJSON
		rootCmd.SetArgs(nil)
		rootCmd.SilenceUsage = oldRootSilenceUsage
		rootCmd.SilenceErrors = oldRootSilenceErrors
		os.Stderr = oldStderr
		stderrReader.Close()
		stderrWriter.Close()
	})

	os.Args = []string{"toneclone", "write", "--persona", "whatever", "--prompt", "hi", "--drafts", "0", "--output", "json"}
	err = Execute()
	stderrWriter.Close()
	stderrBytes, _ := io.ReadAll(stderrReader)
	stderr := string(stderrBytes)
	if err == nil {
		t.Fatal("expected drafts range error")
	}
	if !strings.Contains(stderr, `"message":"--drafts must be between 1 and 5"`) &&
		!strings.Contains(stderr, `"message": "--drafts must be between 1 and 5"`) {
		t.Fatalf("expected structured JSON error for output alias, got %q", stderr)
	}
	if strings.Contains(stderr, "Error:") {
		t.Fatalf("expected JSON error, got plain error output %q", stderr)
	}

	os.Args = []string{"toneclone", "write", "--persona", "whatever", "--prompt", "hi", "--drafts", "0"}
	jsonOutput = true
	stderrReader2, stderrWriter2, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = stderrWriter2
	err = Execute()
	stderrWriter2.Close()
	stderrBytes2, _ := io.ReadAll(stderrReader2)
	stderr2 := string(stderrBytes2)
	stderrReader2.Close()
	if err == nil {
		t.Fatal("expected drafts range error on second execution")
	}
	if strings.Contains(stderr2, `"message"`) {
		t.Fatalf("expected jsonOutput reset without alias/--json, got %q", stderr2)
	}
	if !strings.Contains(stderr2, "Error: --drafts must be between 1 and 5") {
		t.Fatalf("expected plain error after jsonOutput reset, got %q", stderr2)
	}
}

func TestWriteOutputAliasRequestsJSON(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"write", "--output", "json"}, true},
		{[]string{"write", "--output=json"}, true},
		{[]string{"--profile", "dev", "write", "--output", "json"}, true},
		{[]string{"write", "--output", "text"}, false},
		{[]string{"quota", "--output", "json"}, false},
		{[]string{"write"}, false},
	}
	for _, tt := range tests {
		if got := writeOutputAliasRequestsJSON(tt.args); got != tt.want {
			t.Errorf("writeOutputAliasRequestsJSON(%v) = %v, want %v", tt.args, got, tt.want)
		}
	}
}
