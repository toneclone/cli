package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTextVariantsSendsNAndParsesVariants(t *testing.T) {
	var gotReq map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotReq)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"variants": [
				{"content":"draft A","temperature":0.4,"index":0,"angle":{"title":"Bold","shortLabel":"bold","description":"d","approach":"a"}},
				{"content":"draft B","temperature":1.0,"index":1}
			],
			"anglePlan": {"strategyNote":"two angles","angles":[{"title":"Bold","shortLabel":"bold"}]},
			"done": true
		}`))
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	resp, err := tc.Generate.TextVariants(context.Background(), &GenerateTextRequest{Prompt: "hi", PersonaID: "p", N: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotReq["n"] != float64(2) {
		t.Errorf("expected n=2 sent, got %v", gotReq["n"])
	}
	if len(resp.Variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(resp.Variants))
	}
	if resp.Variants[0].Content != "draft A" || resp.Variants[0].Angle == nil || resp.Variants[0].Angle.ShortLabel != "bold" {
		t.Errorf("unexpected variant 0: %+v", resp.Variants[0])
	}
	if resp.AnglePlan == nil || resp.AnglePlan.StrategyNote != "two angles" {
		t.Errorf("expected angle plan, got %+v", resp.AnglePlan)
	}
}
