package test

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"testing"
)

func TestPrimeWritesSuccessOutputToStdout(t *testing.T) {
	tests := []struct {
		name string
		args []string
		json bool
	}{
		{name: "text", args: []string{"prime"}},
		{name: "json", args: []string{"prime", "--json"}, json: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("go", append([]string{"run", ".."}, tt.args...)...)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				t.Fatalf("toneclone %v failed: %v\nstderr: %s", tt.args, err, stderr.String())
			}
			if stdout.Len() == 0 {
				t.Fatalf("expected output on stdout; stderr contained: %s", stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("expected quiet stderr, got: %s", stderr.String())
			}
			if !tt.json {
				return
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
		})
	}
}
