package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHumanizeSendsModeAndText(t *testing.T) {
	var gotReq map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotReq)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":"cleaned text","done":true,"sessionId":"s1","reviewUrl":"https://app.toneclone.ai/writing-canvas/s1"}`))
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	resp, err := tc.Generate.Humanize(context.Background(), "raw text", "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotReq["mode"] != "humanize" {
		t.Errorf("expected mode humanize, got %v", gotReq["mode"])
	}
	if gotReq["text"] != "raw text" {
		t.Errorf("expected text 'raw text', got %v", gotReq["text"])
	}
	if gotReq["createSession"] != true {
		t.Errorf("expected createSession true, got %v", gotReq["createSession"])
	}
	if resp.Text != "cleaned text" || resp.ReviewURL == "" || resp.SessionID != "s1" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestTextDecodesReviewURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"content":"draft","done":true,"sessionId":"abc","reviewUrl":"https://app.toneclone.ai/writing-canvas/abc"}`))
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	resp, err := tc.Generate.Text(context.Background(), &GenerateTextRequest{Prompt: "hi", PersonaID: "p", CreateSession: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ReviewURL == "" || resp.SessionID != "abc" {
		t.Errorf("expected reviewUrl+sessionId, got %+v", resp)
	}
}
