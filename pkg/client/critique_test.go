package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCritiquePostsToDedicatedEndpoint(t *testing.T) {
	var gotReq CritiqueRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query/critique" {
			t.Errorf("expected path /query/critique, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"critiqueId":"c1","sessionId":"s1","strategyNote":"Tighten the close.","suggestions":[{"id":"sug1","type":"rewrite","category":"clarity","priority":1,"title":"Shorten","rationale":"Too long.","anchor":{"quote":"very long"},"replacement":"short"}]}`))
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	start, end := 4, 13
	resp, err := tc.Critique.Get(context.Background(), &CritiqueRequest{
		PersonaID:        "persona-1",
		KnowledgeCardIDs: []string{"card-1"},
		Document:         "a very long draft",
		Selection:        "very long",
		SelectionStart:   &start,
		SelectionEnd:     &end,
		Context:          "LinkedIn post",
		Categories:       []string{"clarity", "voice"},
		Intensity:        "deep",
		MaxSuggestions:   4,
		SessionID:        "s1",
		SourceVersionID:  "v1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotReq.PersonaID != "persona-1" || gotReq.Document != "a very long draft" || gotReq.MaxSuggestions != 4 {
		t.Fatalf("unexpected request: %+v", gotReq)
	}
	if gotReq.SelectionStart == nil || *gotReq.SelectionStart != 4 || gotReq.SelectionEnd == nil || *gotReq.SelectionEnd != 13 {
		t.Fatalf("expected selection offsets, got %+v/%+v", gotReq.SelectionStart, gotReq.SelectionEnd)
	}
	if resp.StrategyNote != "Tighten the close." || len(resp.Suggestions) != 1 || resp.Suggestions[0].Replacement != "short" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCritiqueHistoryUsesSessionQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query/critique/history" || r.URL.Query().Get("sessionId") != "session 1" {
			t.Errorf("unexpected history request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"items":[{"critiqueId":"c1","sessionId":"session 1","strategyNote":"Earlier pass.","suggestions":[]}]}`))
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	resp, err := tc.Critique.History(context.Background(), "session 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].CritiqueID != "c1" {
		t.Fatalf("unexpected history response: %+v", resp)
	}
}
