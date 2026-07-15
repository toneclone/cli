package test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"
)

func TestPrimeJSONWritesMachineOutputToStdout(t *testing.T) {
	cmd := exec.Command("go", "run", "..", "prime", "--json")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("toneclone prime --json failed: %v\nstderr: %s", err, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatalf("expected JSON on stdout; stderr contained: %s", stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected quiet stderr, got: %s", stderr.String())
	}

	var output struct {
		Product string `json:"product"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("stdout was not valid JSON: %v\nstdout: %s", err, stdout.String())
	}
	if output.Product != "ToneClone CLI" {
		t.Fatalf("unexpected product: %q", output.Product)
	}
}
