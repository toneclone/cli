package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKnowledgeCreateFromURLPostsURLAndHint(t *testing.T) {
	var gotReq map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/knowledge/from-url" {
			t.Errorf("expected /knowledge/from-url, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"knowledgeCard":{"knowledgeCardId":"card-1","name":"Product FAQ","instructions":"Use these facts."},"source":{"sourceId":"src-1","knowledgeCardId":"card-1","type":"url","displayName":"example.com","url":"https://example.com","status":"ready","extractedCharCount":123},"synthesis":{"summary":"Summary","keyFacts":["Fact"],"warnings":["Check facts"]}}`))
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	resp, err := tc.Knowledge.CreateFromURL(context.Background(), "https://example.com", "focus on pricing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotReq["url"] != "https://example.com" || gotReq["instructionsHint"] != "focus on pricing" {
		t.Fatalf("unexpected request body: %+v", gotReq)
	}
	if resp.KnowledgeCard.KnowledgeCardID != "card-1" || resp.Source.Type != "url" || resp.Synthesis.Summary != "Summary" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestKnowledgeCreateFromFileUploadsMultipartFileAndHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/knowledge/from-file" {
			t.Errorf("expected /knowledge/from-file, got %s", r.URL.Path)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Fatalf("expected multipart content type, got %s", r.Header.Get("Content-Type"))
		}
		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		var gotFile, gotHint string
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			data, _ := io.ReadAll(part)
			switch part.FormName() {
			case "file":
				gotFile = string(data)
				if part.FileName() != "source.md" {
					t.Errorf("expected filename source.md, got %q", part.FileName())
				}
			case "instructionsHint":
				gotHint = string(data)
			}
		}
		if gotFile != "# Facts\nUse these facts." || gotHint != "extract durable facts" {
			t.Fatalf("unexpected multipart payload file=%q hint=%q", gotFile, gotHint)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"knowledgeCard":{"knowledgeCardId":"card-2","name":"File Facts","instructions":"Use file facts."},"source":{"sourceId":"src-2","knowledgeCardId":"card-2","type":"file","displayName":"source.md","filename":"source.md","status":"ready","sizeBytes":24}}`))
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	resp, err := tc.Knowledge.CreateFromFile(context.Background(), "source.md", strings.NewReader("# Facts\nUse these facts."), "extract durable facts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.KnowledgeCard.KnowledgeCardID != "card-2" || resp.Source.Type != "file" || resp.Source.Filename != "source.md" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestKnowledgeSourcesUsesEscapedCardID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/knowledge/card%2F1/sources" {
			t.Errorf("unexpected path %s escaped=%s", r.URL.Path, r.URL.EscapedPath())
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"sourceId":"src-1","knowledgeCardId":"card/1","type":"url","displayName":"example.com","status":"ready"}]`))
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	sources, err := tc.Knowledge.Sources(context.Background(), "card/1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sources) != 1 || sources[0].SourceID != "src-1" {
		t.Fatalf("unexpected sources: %+v", sources)
	}
}

func TestKnowledgeResourceMethodsEscapePathIDs(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.EscapedPath())
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`[]`))
		case http.MethodPut:
			w.Write([]byte(`{"knowledgeCardId":"card/1","name":"n","instructions":"i"}`))
		case http.MethodPost:
			w.Write([]byte(`{}`))
		case http.MethodDelete:
			w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	ctx := context.Background()
	_, _ = tc.Knowledge.Update(ctx, "card/1", &KnowledgeCard{Name: "n", Instructions: "i"})
	_ = tc.Knowledge.Delete(ctx, "card/1")
	_ = tc.Knowledge.AssociateWithPersona(ctx, "card/1", "persona/1")
	_ = tc.Knowledge.DisassociateFromPersona(ctx, "card/1", "persona/1")
	_, _ = tc.Knowledge.GetPersonaKnowledge(ctx, "persona/1")

	want := []string{
		"PUT /knowledge/card%2F1",
		"DELETE /knowledge/card%2F1",
		"POST /personas/persona%2F1/knowledge",
		"DELETE /personas/persona%2F1/knowledge",
		"GET /personas/persona%2F1/knowledge",
	}
	for i, expected := range want {
		if i >= len(paths) || paths[i] != expected {
			t.Fatalf("path %d = %q, want %q (all paths: %v)", i, paths[i], expected, paths)
		}
	}
}

func TestPersonaResourceMethodsEscapePathIDs(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.EscapedPath())
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			if strings.HasSuffix(r.URL.EscapedPath(), "/files") {
				w.Write([]byte(`{"files":[]}`))
			} else {
				w.Write([]byte(`{"personaId":"persona/1","name":"n"}`))
			}
		case http.MethodPut:
			w.Write([]byte(`{"personaId":"persona/1","name":"n"}`))
		case http.MethodPost, http.MethodDelete:
			w.Write([]byte(`{}`))
		}
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	ctx := context.Background()
	_, _ = tc.Personas.Get(ctx, "persona/1")
	_, _ = tc.Personas.Update(ctx, "persona/1", &Persona{Name: "n"})
	_ = tc.Personas.Delete(ctx, "persona/1")
	_, _ = tc.Personas.ListFiles(ctx, "persona/1")
	_ = tc.Personas.AssociateFiles(ctx, "persona/1", []string{"file/1"})
	_ = tc.Personas.DisassociateFiles(ctx, "persona/1", []string{"file/1"})

	want := []string{
		"GET /personas/persona%2F1",
		"PUT /personas/persona%2F1",
		"DELETE /personas/persona%2F1",
		"GET /personas/persona%2F1/files",
		"POST /personas/persona%2F1/files",
		"DELETE /personas/persona%2F1/files",
	}
	for i, expected := range want {
		if i >= len(paths) || paths[i] != expected {
			t.Fatalf("path %d = %q, want %q (all paths: %v)", i, paths[i], expected, paths)
		}
	}
}
