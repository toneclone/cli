package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewToneCloneClient(t *testing.T) {
	client := NewToneCloneClient("test_key")

	if client.Client == nil {
		t.Error("Expected base Client to be initialized")
	}

	if client.Personas == nil {
		t.Error("Expected Personas client to be initialized")
	}

	if client.Generate == nil {
		t.Error("Expected Generate client to be initialized")
	}
}

func TestNewToneCloneClientFromConfig(t *testing.T) {
	baseURL := "https://test.api.com"
	apiKey := "test_key"
	timeout := 5 * time.Second

	client := NewToneCloneClientFromConfig(baseURL, apiKey, timeout)

	if client.Client.baseURL != baseURL {
		t.Errorf("Expected base URL %s, got %s", baseURL, client.Client.baseURL)
	}

	if client.Client.apiKey != apiKey {
		t.Errorf("Expected API key %s, got %s", apiKey, client.Client.apiKey)
	}

	if client.Client.timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.Client.timeout)
	}
}

func TestWhoAmI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("Expected path /user, got %s", r.URL.Path)
		}

		user := User{
			UserID:    "user123",
			Email:     "test@example.com",
			Name:      "Test User",
			CreatedAt: time.Now(),
			Plan:      "pro",
		}

		json.NewEncoder(w).Encode(user)
	}))
	defer server.Close()

	client := NewToneCloneClient("test_key", WithBaseURL(server.URL))

	user, err := client.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if user.UserID != "user123" {
		t.Errorf("Expected UserID 'user123', got %s", user.UserID)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected Email 'test@example.com', got %s", user.Email)
	}

	if user.Plan != "pro" {
		t.Errorf("Expected Plan 'pro', got %s", user.Plan)
	}
}

func TestValidateConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ping":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok"}`))
		case "/user":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"valid": true}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewToneCloneClient("test_key", WithBaseURL(server.URL))

	err := client.ValidateConnection(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestValidateConnectionFailure(t *testing.T) {
	// Test server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "server error"}`))
	}))
	defer server.Close()

	client := NewToneCloneClient("test_key", WithBaseURL(server.URL))

	err := client.ValidateConnection(context.Background())
	if err == nil {
		t.Fatal("Expected error for server failure")
	}
}

func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ping" {
			t.Errorf("Expected path /ping, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	client := NewToneCloneClient("test_key", WithBaseURL(server.URL))

	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}
