package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.Keys == nil {
		t.Error("Expected Keys map to be initialized")
	}

	if cfg.DefaultTimeout != 30 {
		t.Errorf("Expected DefaultTimeout to be 30, got %d", cfg.DefaultTimeout)
	}

	if cfg.DefaultBaseURL != "https://api.toneclone.ai" {
		t.Errorf("Expected DefaultBaseURL to be https://api.toneclone.ai, got %s", cfg.DefaultBaseURL)
	}
}

func TestAddKey(t *testing.T) {
	cfg := NewConfig()

	// Add first key
	cfg.AddKey("test", "tc_test_abc123", "https://test.api.com")

	if len(cfg.Keys) != 1 {
		t.Errorf("Expected 1 key, got %d", len(cfg.Keys))
	}

	key, exists := cfg.Keys["test"]
	if !exists {
		t.Error("Expected key 'test' to exist")
	}

	if key.Key != "tc_test_abc123" {
		t.Errorf("Expected key to be tc_test_abc123, got %s", key.Key)
	}

	if key.BaseURL != "https://test.api.com" {
		t.Errorf("Expected BaseURL to be https://test.api.com, got %s", key.BaseURL)
	}

	// Should be set as default since it's the first key
	if cfg.DefaultKey != "test" {
		t.Errorf("Expected DefaultKey to be 'test', got %s", cfg.DefaultKey)
	}

	// Add second key with empty base URL (should use default)
	cfg.AddKey("prod", "tc_live_xyz789", "")

	key2, exists := cfg.Keys["prod"]
	if !exists {
		t.Error("Expected key 'prod' to exist")
	}

	if key2.BaseURL != cfg.DefaultBaseURL {
		t.Errorf("Expected BaseURL to be default %s, got %s", cfg.DefaultBaseURL, key2.BaseURL)
	}

	// Default should still be the first key
	if cfg.DefaultKey != "test" {
		t.Errorf("Expected DefaultKey to remain 'test', got %s", cfg.DefaultKey)
	}
}

func TestRemoveKey(t *testing.T) {
	cfg := NewConfig()
	cfg.AddKey("test1", "tc_test_abc123", "")
	cfg.AddKey("test2", "tc_test_xyz789", "")

	// Remove non-default key
	err := cfg.RemoveKey("test2")
	if err != nil {
		t.Errorf("Unexpected error removing key: %v", err)
	}

	if len(cfg.Keys) != 1 {
		t.Errorf("Expected 1 key after removal, got %d", len(cfg.Keys))
	}

	if cfg.DefaultKey != "test1" {
		t.Errorf("Expected DefaultKey to remain 'test1', got %s", cfg.DefaultKey)
	}

	// Remove default key - should clear default and set new one
	err = cfg.RemoveKey("test1")
	if err != nil {
		t.Errorf("Unexpected error removing default key: %v", err)
	}

	if len(cfg.Keys) != 0 {
		t.Errorf("Expected 0 keys after removal, got %d", len(cfg.Keys))
	}

	if cfg.DefaultKey != "" {
		t.Errorf("Expected DefaultKey to be empty, got %s", cfg.DefaultKey)
	}

	// Try to remove non-existent key
	err = cfg.RemoveKey("nonexistent")
	if err == nil {
		t.Error("Expected error when removing non-existent key")
	}
}

func TestSetDefaultKey(t *testing.T) {
	cfg := NewConfig()
	cfg.AddKey("test1", "tc_test_abc123", "")
	cfg.AddKey("test2", "tc_test_xyz789", "")

	// Set existing key as default
	err := cfg.SetDefaultKey("test2")
	if err != nil {
		t.Errorf("Unexpected error setting default key: %v", err)
	}

	if cfg.DefaultKey != "test2" {
		t.Errorf("Expected DefaultKey to be 'test2', got %s", cfg.DefaultKey)
	}

	// Try to set non-existent key as default
	err = cfg.SetDefaultKey("nonexistent")
	if err == nil {
		t.Error("Expected error when setting non-existent key as default")
	}
}

func TestValidate(t *testing.T) {
	cfg := NewConfig()

	// Empty config should be valid
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected empty config to be valid, got error: %v", err)
	}

	// Add valid key
	cfg.AddKey("test", "tc_test_abc123", "https://api.test.com")
	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}

	// Test invalid default key
	cfg.DefaultKey = "nonexistent"
	err = cfg.Validate()
	if err == nil {
		t.Error("Expected error for invalid default key")
	}

	// Reset to valid state
	cfg.DefaultKey = "test"

	// Test empty API key
	cfg.Keys["test"] = APIKeyConfig{
		Key:     "",
		BaseURL: "https://api.test.com",
	}
	err = cfg.Validate()
	if err == nil {
		t.Error("Expected error for empty API key")
	}

	// Test invalid API key format
	cfg.Keys["test"] = APIKeyConfig{
		Key:     "invalid_key",
		BaseURL: "https://api.test.com",
	}
	err = cfg.Validate()
	if err == nil {
		t.Error("Expected error for invalid API key format")
	}

	// Test empty base URL
	cfg.Keys["test"] = APIKeyConfig{
		Key:     "tc_test_abc123",
		BaseURL: "",
	}
	err = cfg.Validate()
	if err == nil {
		t.Error("Expected error for empty base URL")
	}

	// Test negative timeout
	cfg.Keys["test"] = APIKeyConfig{
		Key:     "tc_test_abc123",
		BaseURL: "https://api.test.com",
		Timeout: -1,
	}
	err = cfg.Validate()
	if err == nil {
		t.Error("Expected error for negative timeout")
	}
}

func TestIsValidAPIKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"tc_live_abcdefgh123456789", true},
		{"tc_test_abcdefgh123456789", true},
		{"test_key", true}, // Special test key
		{"invalid_key", false},
		{"tc_live_sh", false}, // Too short
		{"", false},
		{"random_string", false},
	}

	for _, tt := range tests {
		result := isValidAPIKey(tt.key)
		if result != tt.valid {
			t.Errorf("isValidAPIKey(%s) = %v, expected %v", tt.key, result, tt.valid)
		}
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create test config
	cfg := NewConfig()
	cfg.AddKey("test", "tc_test_abc123", "https://test.api.com")
	cfg.Verbose = true
	cfg.Debug = false

	// Save config
	err := cfg.SaveConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Check that file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Check file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	// Check that file is readable only by owner (0600)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %v", info.Mode().Perm())
	}
}

func TestGetCurrentKey(t *testing.T) {
	cfg := NewConfig()
	cfg.AddKey("test", "tc_test_abc123", "https://test.api.com")
	cfg.AddKey("prod", "tc_live_xyz789", "https://api.toneclone.ai")
	cfg.DefaultKey = "test"

	// Test getting default key
	key, err := cfg.GetCurrentKey()
	if err != nil {
		t.Fatalf("Unexpected error getting current key: %v", err)
	}

	if key.Key != "tc_test_abc123" {
		t.Errorf("Expected key tc_test_abc123, got %s", key.Key)
	}

	// Test that defaults are applied
	cfg.Keys["test"] = APIKeyConfig{
		Key:     "tc_test_abc123",
		BaseURL: "",
		Timeout: 0,
	}

	key, err = cfg.GetCurrentKey()
	if err != nil {
		t.Fatalf("Unexpected error getting current key: %v", err)
	}

	if key.BaseURL != cfg.DefaultBaseURL {
		t.Errorf("Expected BaseURL to be default %s, got %s", cfg.DefaultBaseURL, key.BaseURL)
	}

	if key.Timeout != cfg.DefaultTimeout {
		t.Errorf("Expected Timeout to be default %d, got %d", cfg.DefaultTimeout, key.Timeout)
	}
}

func TestGetConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("Unexpected error getting config path: %v", err)
	}

	// Should end with .toneclone.yaml
	if !filepath.IsAbs(path) {
		t.Error("Expected absolute path")
	}

	if filepath.Base(path) != ".toneclone.yaml" {
		t.Errorf("Expected filename .toneclone.yaml, got %s", filepath.Base(path))
	}
}
