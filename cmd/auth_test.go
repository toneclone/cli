package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/toneclone/cli/internal/config"
)

// setupAuthTestHome points HOME and the viper config file at a temp dir and
// sets TONECLONE_API_KEY so the synthetic "environment" profile would appear
// in config.LoadConfig(). It returns the config file path.
func setupAuthTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TONECLONE_API_KEY", "tc_test_env_override_key")
	t.Setenv("TONECLONE_BASE_URL", "")

	configPath := filepath.Join(home, ".toneclone.yaml")
	oldConfigFile := viper.ConfigFileUsed()
	viper.SetConfigFile(configPath)
	t.Cleanup(func() {
		viper.SetConfigFile(oldConfigFile)
	})
	return configPath
}

func readSavedConfig(t *testing.T, configPath string) (*config.Config, []byte) {
	t.Helper()
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}
	var saved config.Config
	if err := yaml.Unmarshal(data, &saved); err != nil {
		t.Fatalf("failed to parse saved config: %v", err)
	}
	return &saved, data
}

func writeTwoProfileConfig(t *testing.T, configPath string) {
	t.Helper()
	cfg := config.NewConfig()
	cfg.AddKey("work", "tc_live_workprofilekey", "https://api.toneclone.ai")
	cfg.AddKey("personal", "tc_live_personalprofilekey", "https://api.toneclone.ai")
	cfg.DefaultKey = "work"
	if err := cfg.SaveConfig(configPath); err != nil {
		t.Fatalf("failed to seed config: %v", err)
	}
}

// An API key supplied through TONECLONE_API_KEY is a per-process override. It
// must never be persisted to the config file as a synthetic "environment"
// profile, and it must not displace the profile being saved as the default.
func TestLoginDoesNotPersistEnvironmentOverride(t *testing.T) {
	configPath := setupAuthTestHome(t)

	origKeyName, origBaseURL, origSkip := keyName, baseURL, skipValidation
	origForce, origFromStdin, origSetDefault := force, fromStdin, setDefault
	t.Cleanup(func() {
		keyName, baseURL, skipValidation = origKeyName, origBaseURL, origSkip
		force, fromStdin, setDefault = origForce, origFromStdin, origSetDefault
	})
	keyName, baseURL, skipValidation, force, fromStdin, setDefault = "work", "", true, false, false, false

	if err := runLogin(loginCmd, nil); err != nil {
		t.Fatalf("runLogin failed: %v", err)
	}

	saved, data := readSavedConfig(t, configPath)
	if _, ok := saved.Keys["environment"]; ok {
		t.Errorf("synthetic 'environment' profile was persisted to the config file:\n%s", data)
	}
	if _, ok := saved.Keys["work"]; !ok {
		t.Errorf("expected 'work' profile to be saved:\n%s", data)
	}
	if saved.DefaultKey != "work" {
		t.Errorf("expected default_key 'work' (first saved profile), got %q", saved.DefaultKey)
	}
}

func TestSwitchDoesNotPersistEnvironmentOverride(t *testing.T) {
	configPath := setupAuthTestHome(t)
	writeTwoProfileConfig(t, configPath)

	if err := switchCmd.Flags().Set("profile", "personal"); err != nil {
		t.Fatalf("failed to set --profile: %v", err)
	}
	t.Cleanup(func() { _ = switchCmd.Flags().Set("profile", "") })

	if err := runSwitch(switchCmd, nil); err != nil {
		t.Fatalf("runSwitch failed: %v", err)
	}

	saved, data := readSavedConfig(t, configPath)
	if _, ok := saved.Keys["environment"]; ok {
		t.Errorf("synthetic 'environment' profile was persisted to the config file:\n%s", data)
	}
	if saved.DefaultKey != "personal" {
		t.Errorf("expected default_key 'personal', got %q", saved.DefaultKey)
	}
}

func TestLogoutDoesNotPersistEnvironmentOverride(t *testing.T) {
	configPath := setupAuthTestHome(t)
	writeTwoProfileConfig(t, configPath)

	if err := runLogout(logoutCmd, []string{"work"}); err != nil {
		t.Fatalf("runLogout failed: %v", err)
	}

	saved, data := readSavedConfig(t, configPath)
	if _, ok := saved.Keys["environment"]; ok {
		t.Errorf("synthetic 'environment' profile was persisted to the config file:\n%s", data)
	}
	if _, ok := saved.Keys["work"]; ok {
		t.Errorf("expected 'work' profile to be removed:\n%s", data)
	}
	if saved.DefaultKey != "personal" {
		t.Errorf("expected remaining 'personal' profile to become default, got %q", saved.DefaultKey)
	}
}
