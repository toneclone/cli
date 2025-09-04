package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test_key")

	if client.apiKey != "test_key" {
		t.Errorf("Expected API key 'test_key', got %s", client.apiKey)
	}

	if client.baseURL != DefaultBaseURL {
		t.Errorf("Expected base URL %s, got %s", DefaultBaseURL, client.baseURL)
	}

	if client.timeout != DefaultTimeout {
		t.Errorf("Expected timeout %v, got %v", DefaultTimeout, client.timeout)
	}
}

func TestClientWithOptions(t *testing.T) {
	customURL := "https://test.api.com"
	customTimeout := 5 * time.Second

	client := NewClient("test_key",
		WithBaseURL(customURL),
		WithTimeout(customTimeout),
	)

	if client.baseURL != customURL {
		t.Errorf("Expected base URL %s, got %s", customURL, client.baseURL)
	}

	if client.timeout != customTimeout {
		t.Errorf("Expected timeout %v, got %v", customTimeout, client.timeout)
	}
}

func TestClientHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test_key" {
			t.Errorf("Expected Authorization header 'Bearer test_key', got %s", auth)
		}

		// Check Content-Type header
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type header 'application/json', got %s", contentType)
		}

		// Check User-Agent header
		userAgent := r.Header.Get("User-Agent")
		if !strings.Contains(userAgent, "toneclone-cli") {
			t.Errorf("Expected User-Agent to contain 'toneclone-cli', got %s", userAgent)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "ok"}`))
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))

	var response map[string]string
	err := client.Get(context.Background(), "/test", &response)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/test" {
			t.Errorf("Expected path /test, got %s", r.URL.Path)
		}

		response := map[string]string{"message": "hello"}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))

	var response map[string]string
	err := client.Get(context.Background(), "/test", &response)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response["message"] != "hello" {
		t.Errorf("Expected message 'hello', got %s", response["message"])
	}
}

func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/test" {
			t.Errorf("Expected path /test, got %s", r.URL.Path)
		}

		// Check request body
		var requestBody map[string]string
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		if err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		if requestBody["name"] != "test" {
			t.Errorf("Expected name 'test', got %s", requestBody["name"])
		}

		response := map[string]string{"id": "123", "name": requestBody["name"]}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))

	requestData := map[string]string{"name": "test"}
	var response map[string]string

	err := client.Post(context.Background(), "/test", requestData, &response)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if response["id"] != "123" {
		t.Errorf("Expected id '123', got %s", response["id"])
	}

	if response["name"] != "test" {
		t.Errorf("Expected name 'test', got %s", response["name"])
	}
}

func TestClientErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found", "message": "Resource not found"}`))
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))

	var response map[string]string
	err := client.Get(context.Background(), "/notfound", &response)

	if err == nil {
		t.Fatal("Expected error for 404 response")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error to contain 'not found', got %s", err.Error())
	}
}

func TestClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "ok"}`))
	}))
	defer server.Close()

	// Create client with very short timeout
	client := NewClient("test_key",
		WithBaseURL(server.URL),
		WithTimeout(10*time.Millisecond),
	)

	var response map[string]string
	err := client.Get(context.Background(), "/test", &response)

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("Expected timeout/deadline error, got %s", err.Error())
	}
}

func TestHealthEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ping" {
			t.Errorf("Expected path /ping, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewClient("test_key", WithBaseURL(server.URL))

	err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("Expected path /user, got %s", r.URL.Path)
		}

		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth == "Bearer invalid_key" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "unauthorized"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"valid": true}`))
	}))
	defer server.Close()

	// Test valid key
	client := NewClient("test_key", WithBaseURL(server.URL))
	err := client.ValidateAPIKey(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error for valid key: %v", err)
	}

	// Test invalid key
	invalidClient := NewClient("invalid_key", WithBaseURL(server.URL))
	err = invalidClient.ValidateAPIKey(context.Background())
	if err == nil {
		t.Fatal("Expected error for invalid key")
	}
}
