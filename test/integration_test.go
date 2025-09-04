package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/toneclone/cli/internal/config"
	"github.com/toneclone/cli/pkg/client"
)

// TestCLIIntegration tests the basic CLI integration flow
func TestCLIIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary config for testing
	tmpDir := t.TempDir()
	configPath := tmpDir + "/test-config.yaml"

	// Test configuration creation
	cfg := config.NewConfig()
	cfg.AddKey("test", "test_key", "https://api.toneclone.ai")

	err := cfg.SaveConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Test configuration loading
	loadedCfg, err := config.LoadConfig()
	if err != nil {
		// This is expected if no config exists, create a minimal one
		loadedCfg = config.NewConfig()
	}

	// Test client creation from config
	if len(loadedCfg.Keys) > 0 {
		currentKey, err := loadedCfg.GetCurrentKey()
		if err != nil {
			t.Fatalf("Failed to get current key: %v", err)
		}

		apiClient := client.NewToneCloneClientFromConfig(
			currentKey.BaseURL,
			currentKey.Key,
			time.Duration(currentKey.Timeout)*time.Second,
		)

		if apiClient == nil {
			t.Fatal("Failed to create API client")
		}

		// Test basic connectivity (will fail with test_key but should not crash)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// This will likely fail with authentication but should not crash
		_ = apiClient.ValidateConnection(ctx)
	}
}

// TestEnvironmentVariableAuth tests authentication via environment variables
func TestEnvironmentVariableAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set test environment variable
	originalKey := os.Getenv("TONECLONE_API_KEY")
	originalURL := os.Getenv("TONECLONE_BASE_URL")

	defer func() {
		// Restore original values
		if originalKey != "" {
			os.Setenv("TONECLONE_API_KEY", originalKey)
		} else {
			os.Unsetenv("TONECLONE_API_KEY")
		}

		if originalURL != "" {
			os.Setenv("TONECLONE_BASE_URL", originalURL)
		} else {
			os.Unsetenv("TONECLONE_BASE_URL")
		}
	}()

	// Set test values
	os.Setenv("TONECLONE_API_KEY", "tc_test_environment_key")
	os.Setenv("TONECLONE_BASE_URL", "https://test.api.toneclone.ai")

	// Load config with environment variables
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config with environment variables: %v", err)
	}

	// Check that environment key was loaded
	envKey, exists := cfg.Keys["environment"]
	if !exists {
		t.Fatal("Expected environment key to be loaded")
	}

	if envKey.Key != "tc_test_environment_key" {
		t.Errorf("Expected key from environment, got %s", envKey.Key)
	}

	if envKey.BaseURL != "https://test.api.toneclone.ai" {
		t.Errorf("Expected URL from environment, got %s", envKey.BaseURL)
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *config.Config
		expectError bool
	}{
		{
			name: "valid config",
			setupConfig: func() *config.Config {
				cfg := config.NewConfig()
				cfg.AddKey("test", "tc_test_valid_key", "https://api.test.com")
				return cfg
			},
			expectError: false,
		},
		{
			name: "invalid API key format",
			setupConfig: func() *config.Config {
				cfg := config.NewConfig()
				cfg.AddKey("test", "invalid_key_format", "https://api.test.com")
				return cfg
			},
			expectError: true,
		},
		{
			name: "empty API key",
			setupConfig: func() *config.Config {
				cfg := config.NewConfig()
				cfg.Keys = map[string]config.APIKeyConfig{
					"test": {
						Key:     "",
						BaseURL: "https://api.test.com",
					},
				}
				cfg.DefaultKey = "test"
				return cfg
			},
			expectError: true,
		},
		{
			name: "invalid default key",
			setupConfig: func() *config.Config {
				cfg := config.NewConfig()
				cfg.AddKey("test", "tc_test_valid_key", "https://api.test.com")
				cfg.DefaultKey = "nonexistent"
				return cfg
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setupConfig()
			err := cfg.Validate()

			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}
