package cmd

import (
	"os"
	"path/filepath"
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
	for _, name := range []string{"personalize", "humanize", "quota", "prime"} {
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
