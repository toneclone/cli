package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/toneclone/cli/pkg/client"
)

// loginWithFlags runs `auth login --name work --skip-validation` with the API
// key taken from TONECLONE_API_KEY, restoring the package-level flags after.
func loginWithFlags(t *testing.T) error {
	t.Helper()
	origKeyName, origBaseURL, origSkip := keyName, baseURL, skipValidation
	origForce, origFromStdin, origSetDefault := force, fromStdin, setDefault
	t.Cleanup(func() {
		keyName, baseURL, skipValidation = origKeyName, origBaseURL, origSkip
		force, fromStdin, setDefault = origForce, origFromStdin, origSetDefault
	})
	keyName, baseURL, skipValidation, force, fromStdin, setDefault = "work", "", true, false, false, false
	return runLogin(loginCmd, nil)
}

// Ported from toneclone/toneclone 9b572b296. An existing config that fails to
// load or validate must never be replaced by a blank config on login: saving
// would silently delete every profile already stored in it.
func TestLoginDoesNotOverwriteUnloadableConfig(t *testing.T) {
	configPath := setupAuthTestHome(t)
	existing := `default_key: personal
keys:
  personal:
    key: tc_live_personal_key_123
    base_url: https://api.toneclone.ai
  legacy:
    key: not-a-valid-key-format
    base_url: https://api.toneclone.ai
`
	if err := os.WriteFile(configPath, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loginWithFlags(t); err == nil {
		t.Errorf("expected runLogin to fail on an invalid existing config")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if string(data) != existing {
		t.Errorf("existing config was overwritten:\n%s", data)
	}
}

// Ported from toneclone/toneclone 068acace0. A config file that exists but
// cannot be read (e.g. a write-only file) must abort login rather than be
// treated as an empty config.
func TestLoginDoesNotOverwriteUnreadableConfig(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("file permissions are not enforced for root")
	}
	configPath := setupAuthTestHome(t)
	existing := `default_key: personal
keys:
  personal:
    key: tc_live_personal_key_123
    base_url: https://api.toneclone.ai
`
	if err := os.WriteFile(configPath, []byte(existing), 0200); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(configPath, 0600) })

	if err := loginWithFlags(t); err == nil {
		t.Errorf("expected runLogin to fail on an unreadable existing config")
	}

	if err := os.Chmod(configPath, 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if string(data) != existing {
		t.Errorf("existing config was overwritten:\n%s", data)
	}
}

// A missing config file is still the normal first-login case.
func TestLoginCreatesConfigWhenMissing(t *testing.T) {
	configPath := setupAuthTestHome(t)

	if err := loginWithFlags(t); err != nil {
		t.Fatalf("runLogin with no config file: %v", err)
	}
	saved, data := readSavedConfig(t, configPath)
	if _, ok := saved.Keys["work"]; !ok {
		t.Errorf("expected 'work' profile to be saved:\n%s", data)
	}
}

// Ported from toneclone/toneclone d6091a87b (#790). Server timestamps can be
// slightly ahead of the local clock; they must not render as "-4 minutes ago".
func TestFormatTimeFutureTimestampIsNotNegative(t *testing.T) {
	got := formatTime(time.Now().Add(5 * time.Minute))
	if strings.Contains(got, "-") {
		t.Fatalf("formatTime(future) = %q, want no negative duration", got)
	}
}

func TestFormatTimeRecent(t *testing.T) {
	tests := []struct {
		name string
		age  time.Duration
		want string
	}{
		{"seconds ago", 10 * time.Second, "0 minutes ago"},
		{"minutes ago", 5*time.Minute + time.Second, "5 minutes ago"},
		{"hours ago", 3*time.Hour + time.Minute, "3 hours ago"},
		{"days ago", 2*24*time.Hour + time.Hour, "2 days ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatTime(time.Now().Add(-tt.age)); got != tt.want {
				t.Fatalf("formatTime(-%v) = %q, want %q", tt.age, got, tt.want)
			}
		})
	}
}

// runUserSettingsCapture runs `user settings` against a fake API and returns
// what it printed to stdout.
func runUserSettingsCapture(t *testing.T, format string, globalJSON bool) (string, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userId":"u1","email":"a@example.com","plan":"pro"}`))
	}))
	t.Cleanup(srv.Close)

	t.Setenv("HOME", t.TempDir())
	t.Setenv("TONECLONE_API_KEY", "tc_test_0123456789abcdef")
	t.Setenv("TONECLONE_BASE_URL", srv.URL)

	oldFormat, oldJSON := userFormat, jsonOutput
	t.Cleanup(func() { userFormat, jsonOutput = oldFormat, oldJSON })
	userFormat, jsonOutput = format, globalJSON

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	os.Stdout = w
	runErr := runUserSettings(settingsCmd, nil)
	closeErr := w.Close()
	os.Stdout = oldStdout
	out, readErr := io.ReadAll(r)
	_ = r.Close()
	if closeErr != nil {
		t.Fatalf("closing stdout pipe: %v", closeErr)
	}
	if readErr != nil {
		t.Fatalf("reading stdout pipe: %v", readErr)
	}
	if runErr != nil {
		t.Fatalf("runUserSettings: %v", runErr)
	}
	return string(out), srv.URL
}

// Ported from toneclone/toneclone 101eb8ca3 and 91d79bee3. `user settings
// --format=json` is documented in the command help, so it must emit JSON with
// the same fields as the text output instead of being silently ignored.
func TestUserSettingsHonorsJSONFormat(t *testing.T) {
	for _, tc := range []struct {
		name       string
		format     string
		globalJSON bool
	}{
		{"--format=json", "json", false},
		{"global --json", "table", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, baseURL := runUserSettingsCapture(t, tc.format, tc.globalJSON)

			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(out), &parsed); err != nil {
				t.Fatalf("expected JSON output, got %q: %v", out, err)
			}
			if parsed["userId"] != "u1" {
				t.Errorf("userId = %v, want u1", parsed["userId"])
			}
			if parsed["baseUrl"] != baseURL {
				t.Errorf("baseUrl = %v, want %s", parsed["baseUrl"], baseURL)
			}
			if name, _ := parsed["apiKeyName"].(string); name == "" {
				t.Errorf("apiKeyName (active key profile) missing from JSON: %s", out)
			}
			if cf, _ := parsed["configFile"].(string); cf == "" {
				t.Errorf("configFile missing from JSON: %s", out)
			}
		})
	}
}

// Ported from toneclone/toneclone 62685b203. A monthly-quota 429 (upgrade_url,
// no Retry-After) from `write` must reach agents as a non-retryable paywall
// with the upgrade link, not as "rate_limited ... please wait".
func TestWriteQuota429IsPaywallNotRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"Monthly draft limit exceeded","upgrade_url":"https://app.example.com/billing"}`))
	}))
	t.Cleanup(srv.Close)

	c := client.NewToneCloneClientFromConfig(srv.URL, "tc_test_key", 10*time.Second)
	_, err := c.Generate.Text(context.Background(), &client.GenerateTextRequest{Prompt: "hi"})
	if err == nil {
		t.Fatal("expected an error")
	}
	env := classifyError(classifyRateLimit(err))
	if env.Code != "paywall" || env.Retryable {
		t.Errorf("envelope = %+v, want non-retryable paywall", env)
	}
	if !strings.Contains(env.Message, "https://app.example.com/billing") {
		t.Errorf("envelope message %q lacks the upgrade link", env.Message)
	}
}

func TestUserSettingsDefaultsToText(t *testing.T) {
	out, _ := runUserSettingsCapture(t, "table", false)
	if !strings.HasPrefix(out, "User Settings") {
		t.Fatalf("expected text output, got %q", out)
	}
}
