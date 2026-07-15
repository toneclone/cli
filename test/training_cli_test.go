package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/toneclone/cli/pkg/client"
)

func TestTrainingAddDirectoryUsesSupportedUploadFlow(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"email-one.txt": "First writing sample",
		"email-two.txt": "Second writing sample",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var uploadedNames []string
	var associatedIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/personas/persona-1":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(client.Persona{PersonaID: "persona-1", Name: "Email v2"})
		case r.Method == http.MethodPost && r.URL.Path == "/files":
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Errorf("FormFile(file) error = %v", err)
				http.Error(w, "missing file", http.StatusBadRequest)
				return
			}
			file.Close()
			if source := r.FormValue("source"); source != "cli" {
				t.Errorf("source = %q, want cli", source)
			}
			uploadedNames = append(uploadedNames, header.Filename)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(client.TrainingFile{FileID: "id-" + header.Filename, FileName: header.Filename})
		case r.Method == http.MethodPost && r.URL.Path == "/personas/persona-1/files":
			var request struct {
				FileIDs []string `json:"fileIds"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode association request: %v", err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			associatedIDs = request.FileIDs
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message":"associated"}`))
		default:
			http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	cmd := exec.Command("go", "run", "..", "training", "add", "--directory="+dir, "--persona=persona-1")
	cmd.Env = append(os.Environ(),
		"TONECLONE_API_KEY=test_key",
		"TONECLONE_BASE_URL="+server.URL,
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("training add failed: %v\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected quiet stderr, got: %s", stderr.String())
	}

	sort.Strings(uploadedNames)
	sort.Strings(associatedIDs)
	if got, want := strings.Join(uploadedNames, ","), "email-one.txt,email-two.txt"; got != want {
		t.Fatalf("uploaded files = %q, want %q", got, want)
	}
	if got, want := strings.Join(associatedIDs, ","), "id-email-one.txt,id-email-two.txt"; got != want {
		t.Fatalf("associated IDs = %q, want %q", got, want)
	}
}
