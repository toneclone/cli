package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecipesClientCRUDAndSuggestPaths(t *testing.T) {
	var requests []struct {
		Method string
		Path   string
		Body   RecipeRequest
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body RecipeRequest
		if r.Body != nil {
			bytes, _ := io.ReadAll(r.Body)
			if len(bytes) > 0 {
				_ = json.Unmarshal(bytes, &body)
			}
		}
		requests = append(requests, struct {
			Method string
			Path   string
			Body   RecipeRequest
		}{r.Method, r.URL.Path, body})
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/recipes":
			if r.Method == http.MethodGet {
				w.Write([]byte(`[{"recipeId":"r1","command":"launch","name":"Launch","instruction":"Write launch copy","defaultDraftCount":2}]`))
				return
			}
			w.Write([]byte(`{"recipeId":"r2","command":"launch","name":"Launch","instruction":"Write launch copy","defaultDraftCount":2}`))
		case "/recipes/r1":
			w.Write([]byte(`{"recipeId":"r1","command":"launch","name":"Launch 2","instruction":"Write launch copy","defaultDraftCount":3}`))
		case "/recipes/suggest":
			w.Write([]byte(`{"command":"follow-up","name":"Follow Up","instruction":"Write a follow-up","defaultDraftCount":1}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	list, err := tc.Recipes.List(context.Background())
	if err != nil || len(list) != 1 || list[0].RecipeID != "r1" {
		t.Fatalf("unexpected list: %+v err=%v", list, err)
	}
	created, err := tc.Recipes.Create(context.Background(), &RecipeRequest{Command: "launch", Name: "Launch", Instruction: "Write launch copy", AngleHints: []string{"pain"}, DefaultDraftCount: 2})
	if err != nil || created.RecipeID != "r2" {
		t.Fatalf("unexpected create: %+v err=%v", created, err)
	}
	updated, err := tc.Recipes.Update(context.Background(), "r1", &RecipeRequest{Command: "launch", Name: "Launch 2", Instruction: "Write launch copy", DefaultDraftCount: 3})
	if err != nil || updated.DefaultDraftCount != 3 {
		t.Fatalf("unexpected update: %+v err=%v", updated, err)
	}
	suggested, err := tc.Recipes.Suggest(context.Background(), &RecipeSuggestionRequest{Prompt: "announce", Draft: "draft"})
	if err != nil || suggested.Command != "follow-up" {
		t.Fatalf("unexpected suggest: %+v err=%v", suggested, err)
	}

	want := []string{"GET /recipes", "POST /recipes", "PUT /recipes/r1", "POST /recipes/suggest"}
	for i, req := range requests {
		got := req.Method + " " + req.Path
		if got != want[i] {
			t.Errorf("request %d = %s, want %s", i, got, want[i])
		}
	}
	if requests[1].Body.AngleHints[0] != "pain" || requests[1].Body.DefaultDraftCount != 2 {
		t.Fatalf("create body missing fields: %+v", requests[1].Body)
	}
}
