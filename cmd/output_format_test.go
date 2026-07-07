package cmd

import (
	"encoding/json"
	"os"
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
