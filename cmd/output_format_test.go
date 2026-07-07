package cmd

import "testing"

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
