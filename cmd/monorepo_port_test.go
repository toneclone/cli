package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/toneclone/cli/pkg/client"
)

// Ported from toneclone/toneclone 9cc28e8de (#866). Each command's --timeout
// flag must hold its own documented default; they used to share one variable,
// so registering the later commands overwrote ping's default of 5 with 10.
func TestTimeoutFlagsKeepTheirOwnDefaults(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{"ping", "5"},
		{"health", "10"},
		{"status", "10"},
	} {
		cmd := pingCmd
		switch tc.name {
		case "health":
			cmd = healthCmd
		case "status":
			cmd = healthStatusCmd
		}
		if err := cmd.ParseFlags(nil); err != nil {
			t.Fatalf("%s: ParseFlags: %v", tc.name, err)
		}
		flag := cmd.Flags().Lookup("timeout")
		if flag == nil {
			t.Fatalf("%s: timeout flag not registered", tc.name)
		}
		if flag.DefValue != tc.want {
			t.Fatalf("%s: documented default = %s, want %s", tc.name, flag.DefValue, tc.want)
		}
		if got := flag.Value.String(); got != tc.want {
			t.Errorf("%s: effective timeout without --timeout = %s, want %s", tc.name, got, tc.want)
		}
	}
}

// captureStdout runs fn and returns everything it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// Ported from toneclone/toneclone f71d6cf5d (#758): byte slicing [:47] can
// split a multi-byte rune and emit invalid UTF-8.
func TestOutputKnowledgeTableTruncatesOnRuneBoundary(t *testing.T) {
	// Each curly quote is 3 bytes, so a byte-index cut at 47 lands mid-rune.
	instructions := strings.Repeat("“", 60)

	out := captureStdout(t, func() {
		if err := outputKnowledgeTable([]client.KnowledgeCard{{
			KnowledgeCardID: "k1",
			Name:            "Quotes",
			Instructions:    instructions,
		}}); err != nil {
			t.Fatal(err)
		}
	})

	if !utf8.ValidString(out) {
		t.Fatalf("knowledge table output contains invalid UTF-8: %q", out)
	}
	if want := strings.Repeat("“", 47) + "..."; !strings.Contains(out, want) {
		t.Fatalf("expected instructions truncated to 47 characters plus ellipsis, got %q", out)
	}
}

func TestOutputHealthTableTruncatesOnRuneBoundary(t *testing.T) {
	result := &HealthResult{
		Timestamp: time.Now(),
		Checks: []HealthCheck{{
			Name:     "api",
			Status:   "fail",
			Duration: time.Millisecond,
			Error:    strings.Repeat("é", 60),
		}},
	}

	out := captureStdout(t, func() {
		if err := outputHealthTable(result); err != nil {
			t.Fatal(err)
		}
	})

	if !utf8.ValidString(out) {
		t.Fatalf("health table output contains invalid UTF-8: %q", out)
	}
}

// withStdin replaces os.Stdin with a pipe holding input for the duration of the test.
func withStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	orig := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		_ = r.Close()
	})
}

// Ported from toneclone/toneclone 253e71b13 (#755): fmt.Scanln into a string
// captured only the first word of each line.
func TestInteractiveKnowledgeCardCreationKeepsFullLines(t *testing.T) {
	var got struct {
		Name         string `json:"name"`
		Instructions string `json:"instructions"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/knowledge" {
			_ = json.NewDecoder(r.Body).Decode(&got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"knowledgeCardId":"k1","name":"x","instructions":"y"}`))
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())
	t.Setenv("TONECLONE_API_KEY", "test_key")
	t.Setenv("TONECLONE_BASE_URL", srv.URL)
	origName, origInstr := knowledgeName, knowledgeInstructions
	t.Cleanup(func() { knowledgeName, knowledgeInstructions = origName, origInstr })

	withStdin(t, "Cold Outreach\nWrite short friendly emails.\nAlways end with a question.\n\n")

	captureStdout(t, func() {
		if err := runInteractiveKnowledgeCardCreation(); err != nil {
			t.Errorf("runInteractiveKnowledgeCardCreation: %v", err)
		}
	})

	if want := "Cold Outreach"; got.Name != want {
		t.Errorf("name = %q, want %q", got.Name, want)
	}
	if want := "Write short friendly emails.\nAlways end with a question."; got.Instructions != want {
		t.Errorf("instructions = %q, want %q", got.Instructions, want)
	}
}

func TestLineReaderReadsWholeLines(t *testing.T) {
	read := lineReader(strings.NewReader("Founder Voice\r\nsecond line\n"))
	if got := read(); got != "Founder Voice" {
		t.Errorf("first line = %q, want %q", got, "Founder Voice")
	}
	if got := read(); got != "second line" {
		t.Errorf("second line = %q, want %q", got, "second line")
	}
	if got := read(); got != "" {
		t.Errorf("at EOF = %q, want empty", got)
	}
}

// Ported from toneclone/toneclone ebd58d197 (#735): non-recursive
// `training add --directory` must find top-level files however the user
// spelled the path ("docs", "./docs", "docs/").
func TestCollectDirectoryTrainingFilesNonRecursive(t *testing.T) {
	parent := t.TempDir()
	const dirName = "docs"
	dir := filepath.Join(parent, dirName)
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "ignored.md"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(parent); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	sep := string(filepath.Separator)
	for _, root := range []string{dirName, "./" + dirName, dirName + sep, "./" + dirName + sep} {
		t.Run(root, func(t *testing.T) {
			files, err := collectDirectoryTrainingFiles(root, false)
			if err != nil {
				t.Fatalf("collectDirectoryTrainingFiles(%q, false): %v", root, err)
			}
			if len(files) != 1 || filepath.Base(files[0]) != "notes.md" {
				t.Fatalf("collectDirectoryTrainingFiles(%q, false) = %v, want only notes.md", root, files)
			}
		})
	}

	files, err := collectDirectoryTrainingFiles(dirName, true)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if len(files) != 2 {
		t.Fatalf("recursive walk = %v, want 2 files", files)
	}
}
