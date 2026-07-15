package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUploadFileDoesNotRetryServerErrorsWithoutIdempotency(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"temporarily unavailable"}`))
	}))
	defer server.Close()

	api := NewClient("test-key", WithBaseURL(server.URL))
	_, err := NewTrainingClient(api).UploadFile(context.Background(), strings.NewReader("sample text"), "sample.txt")
	if err == nil {
		t.Fatal("UploadFile() error = nil, want server error")
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want 1 to avoid duplicate file creation", requests)
	}
}

func TestUploadFileRetriesRateLimits(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TrainingFile{FileID: "file-after-retry", FileName: "sample.txt"})
	}))
	defer server.Close()

	api := NewClient("test-key", WithBaseURL(server.URL))
	file, err := NewTrainingClient(api).UploadFile(context.Background(), strings.NewReader("sample text"), "sample.txt")
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if file.FileID != "file-after-retry" || requests != 2 {
		t.Fatalf("file = %+v, requests = %d; want successful second request", file, requests)
	}
}

func TestUploadFileUsesSupportedSingleFileEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/files" {
			t.Fatalf("request = %s %s, want POST /files", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		if got := r.FormValue("source"); got != "cli" {
			t.Fatalf("source = %q, want cli", got)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile(file) error = %v", err)
		}
		file.Close()
		if header.Filename != "sample.txt" {
			t.Fatalf("filename = %q, want sample.txt", header.Filename)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TrainingFile{FileID: "file-1", FileName: header.Filename})
	}))
	defer server.Close()

	api := NewClient("test-key", WithBaseURL(server.URL))
	file, err := NewTrainingClient(api).UploadFile(context.Background(), strings.NewReader("sample text"), "sample.txt")
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if file.FileID != "file-1" {
		t.Fatalf("FileID = %q, want file-1", file.FileID)
	}
}
