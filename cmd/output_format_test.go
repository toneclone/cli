package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/toneclone/cli/pkg/client"
)

func TestWantsJSONFormatHonorsGlobalJSON(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() { jsonOutput = oldJSON })

	jsonOutput = true
	if !wantsJSONFormat("table") {
		t.Fatal("expected global --json to request JSON output")
	}
}

func TestWantsJSONFormatHonorsLegacyFormat(t *testing.T) {
	oldJSON := jsonOutput
	t.Cleanup(func() { jsonOutput = oldJSON })

	jsonOutput = false
	if !wantsJSONFormat("json") {
		t.Fatal("expected --format=json to request JSON output")
	}
	if wantsJSONFormat("table") {
		t.Fatal("expected table format without global --json to stay table output")
	}
}

func TestPersonaCreateHonorsGlobalJSON(t *testing.T) {
	oldJSON := jsonOutput
	oldFormat := personaFormat
	oldStdout := os.Stdout
	t.Cleanup(func() {
		jsonOutput = oldJSON
		personaFormat = oldFormat
		os.Stdout = oldStdout
	})

	jsonOutput = true
	personaFormat = "table"
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	if err := outputPersonaCreated(&client.Persona{PersonaID: "p1", Name: "Persona"}); err != nil {
		t.Fatal(err)
	}
	write.Close()
	var got map[string]interface{}
	if err := json.NewDecoder(read).Decode(&got); err != nil {
		t.Fatalf("expected JSON persona output: %v", err)
	}
	if got["personaId"] != "p1" {
		t.Fatalf("unexpected persona JSON: %+v", got)
	}
}

func TestPersonaDeleteJSONRequiresConfirmContract(t *testing.T) {
	oldJSON := jsonOutput
	oldFormat := personaFormat
	oldConfirm := personaConfirm
	oldStdout := os.Stdout
	t.Cleanup(func() {
		jsonOutput = oldJSON
		personaFormat = oldFormat
		personaConfirm = oldConfirm
		os.Stdout = oldStdout
	})
	deleteCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path != "/personas/p1" {
				t.Fatalf("unexpected get path: %s", r.URL.Path)
			}
			w.Write([]byte(`{"personaId":"p1","name":"Persona"}`))
		case http.MethodDelete:
			deleteCalled = true
			w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()
	t.Setenv("TONECLONE_API_KEY", "test_key")
	t.Setenv("TONECLONE_BASE_URL", server.URL)
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	jsonOutput = true
	personaFormat = "table"
	personaConfirm = false
	err = runDeletePersona(nil, []string{"p1"})
	write.Close()
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("expected JSON delete to require --confirm, got %v", err)
	}
	if deleteCalled {
		t.Fatal("delete should not be called without --confirm in JSON mode")
	}
	read.Close()
}

func TestPersonaDeleteHonorsGlobalJSONWithConfirm(t *testing.T) {
	oldJSON := jsonOutput
	oldFormat := personaFormat
	oldConfirm := personaConfirm
	oldStdout := os.Stdout
	t.Cleanup(func() {
		jsonOutput = oldJSON
		personaFormat = oldFormat
		personaConfirm = oldConfirm
		os.Stdout = oldStdout
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"personaId":"p1","name":"Persona"}`))
		case http.MethodDelete:
			w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected method: %s", r.Method)
		}
	}))
	defer server.Close()
	t.Setenv("TONECLONE_API_KEY", "test_key")
	t.Setenv("TONECLONE_BASE_URL", server.URL)
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = write
	jsonOutput = true
	personaFormat = "table"
	personaConfirm = true
	if err := runDeletePersona(nil, []string{"p1"}); err != nil {
		t.Fatal(err)
	}
	write.Close()
	var got map[string]interface{}
	if err := json.NewDecoder(read).Decode(&got); err != nil {
		t.Fatalf("expected JSON delete output: %v", err)
	}
	if got["deleted"] != true {
		t.Fatalf("unexpected delete JSON: %+v", got)
	}
}
