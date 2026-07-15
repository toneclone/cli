package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/toneclone/cli/pkg/client"
)

func TestUploadDirectoryFilesProvidesRecoveryCommandWhenAssociationFails(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		writeTrainingTestFile(t, dir, "first.txt", "first sample"),
		writeTrainingTestFile(t, dir, "second.txt", "second sample"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/files":
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("FormFile(file) error = %v", err)
			}
			file.Close()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(client.TrainingFile{FileID: "id-" + header.Filename, FileName: header.Filename})
		case r.Method == http.MethodPost && r.URL.Path == "/personas/persona-1/files":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"temporarily unavailable"}`))
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := client.NewToneCloneClient("test-key", client.WithBaseURL(server.URL))
	persona := &client.Persona{PersonaID: "persona-1", Name: "Writer"}
	err := uploadDirectoryFiles(context.Background(), api, persona, files, false)
	if err == nil {
		t.Fatal("uploadDirectoryFiles() error = nil, want association failure")
	}
	for _, want := range []string{
		"files remain uploaded but may be unassociated",
		"toneclone training associate",
		"id-first.txt,id-second.txt",
		"persona-1",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want recovery detail %q", err, want)
		}
	}
}

func TestUploadDirectoryFilesReturnsErrorAfterAssociatingSuccessfulUploads(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		writeTrainingTestFile(t, dir, "first.txt", "first sample"),
		writeTrainingTestFile(t, dir, "broken.txt", "broken sample"),
		writeTrainingTestFile(t, dir, "third.txt", "third sample"),
	}

	var uploadedNames []string
	var associatedIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/files":
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("FormFile(file) error = %v", err)
			}
			file.Close()
			uploadedNames = append(uploadedNames, header.Filename)
			w.Header().Set("Content-Type", "application/json")
			if header.Filename == "broken.txt" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error":"invalid sample"}`))
				return
			}
			json.NewEncoder(w).Encode(client.TrainingFile{FileID: "id-" + header.Filename, FileName: header.Filename})
		case r.Method == http.MethodPost && r.URL.Path == "/personas/persona-1/files":
			var request struct {
				FileIDs []string `json:"fileIds"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode association request: %v", err)
			}
			associatedIDs = request.FileIDs
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := client.NewToneCloneClient("test-key", client.WithBaseURL(server.URL))
	persona := &client.Persona{PersonaID: "persona-1", Name: "Writer"}
	err := uploadDirectoryFiles(context.Background(), api, persona, files, false)
	if err == nil || !strings.Contains(err.Error(), "1 of 3 files failed") {
		t.Fatalf("uploadDirectoryFiles() error = %v, want partial failure", err)
	}
	if got, want := fmt.Sprint(uploadedNames), "[first.txt broken.txt third.txt]"; got != want {
		t.Fatalf("attempted uploads = %s, want %s", got, want)
	}
	if got, want := fmt.Sprint(associatedIDs), "[id-first.txt id-third.txt]"; got != want {
		t.Fatalf("associated IDs = %s, want %s", got, want)
	}
}

func TestUploadDirectoryFilesPreservesEachFileAsASeparateSample(t *testing.T) {
	dir := t.TempDir()
	files := []string{
		writeTrainingTestFile(t, dir, "first.txt", "first sample"),
		writeTrainingTestFile(t, dir, "second.txt", "second sample"),
	}

	var uploadedNames []string
	var associatedIDs []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/files":
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("FormFile(file) error = %v", err)
			}
			file.Close()
			uploadedNames = append(uploadedNames, header.Filename)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(client.TrainingFile{FileID: "id-" + header.Filename, FileName: header.Filename})
		case r.Method == http.MethodPost && r.URL.Path == "/personas/persona-1/files":
			var request struct {
				FileIDs []string `json:"fileIds"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatalf("decode association request: %v", err)
			}
			associatedIDs = request.FileIDs
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL.Path), http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := client.NewToneCloneClient("test-key", client.WithBaseURL(server.URL))
	persona := &client.Persona{PersonaID: "persona-1", Name: "Writer"}
	if err := uploadDirectoryFiles(context.Background(), api, persona, files, false); err != nil {
		t.Fatalf("uploadDirectoryFiles() error = %v", err)
	}

	if got, want := fmt.Sprint(uploadedNames), "[first.txt second.txt]"; got != want {
		t.Fatalf("uploaded files = %s, want %s", got, want)
	}
	if got, want := fmt.Sprint(associatedIDs), "[id-first.txt id-second.txt]"; got != want {
		t.Fatalf("associated IDs = %s, want %s", got, want)
	}
}

func writeTrainingTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
