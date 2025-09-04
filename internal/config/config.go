package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// APIKeyConfig represents configuration for a named API key
type APIKeyConfig struct {
	Key     string `yaml:"key" json:"key"`
	BaseURL string `yaml:"base_url" json:"base_url"`
	Timeout int    `yaml:"timeout,omitempty" json:"timeout,omitempty"` // seconds
}

// Config represents the complete CLI configuration
type Config struct {
	DefaultKey string                  `yaml:"default_key,omitempty" json:"default_key,omitempty"`
	Keys       map[string]APIKeyConfig `yaml:"keys" json:"keys"`

	// Global settings
	Verbose bool `yaml:"verbose,omitempty" json:"verbose,omitempty"`
	Debug   bool `yaml:"debug,omitempty" json:"debug,omitempty"`

	// Default values
	DefaultTimeout int    `yaml:"default_timeout,omitempty" json:"default_timeout,omitempty"`
	DefaultBaseURL string `yaml:"default_base_url,omitempty" json:"default_base_url,omitempty"`
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		Keys:           make(map[string]APIKeyConfig),
		DefaultTimeout: 30,
		DefaultBaseURL: "https://api.toneclone.ai",
	}
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig() (*Config, error) {
	config := NewConfig()

	// Get config file path
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		configPath, _ = GetConfigPath()
	}

	// Try to read config file directly
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal config: %w", err)
			}
		}
	}

	// Override with environment variables if present
	if apiKey := os.Getenv("TONECLONE_API_KEY"); apiKey != "" {
		// Create or update default key from environment
		if config.Keys == nil {
			config.Keys = make(map[string]APIKeyConfig)
		}

		envKeyName := "environment"
		config.Keys[envKeyName] = APIKeyConfig{
			Key:     apiKey,
			BaseURL: getEnvOrDefault("TONECLONE_BASE_URL", config.DefaultBaseURL),
		}

		// Set as default if no default is configured
		if config.DefaultKey == "" {
			config.DefaultKey = envKeyName
		}
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// SaveConfig saves the configuration to file
func (c *Config) SaveConfig(configPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	// Write to file with proper permissions
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetCurrentKey returns the configuration for the currently selected API key
func (c *Config) GetCurrentKey() (APIKeyConfig, error) {
	profile := viper.GetString("profile")
	keyName := profile

	// If no profile specified, use default
	if keyName == "" {
		keyName = c.DefaultKey
	}

	// If still no key name, check for environment variable
	if keyName == "" && os.Getenv("TONECLONE_API_KEY") != "" {
		keyName = "environment"
	}

	if keyName == "" {
		return APIKeyConfig{}, fmt.Errorf("no API key configured. Run 'toneclone auth login' or set TONECLONE_API_KEY environment variable")
	}

	keyConfig, exists := c.Keys[keyName]
	if !exists {
		return APIKeyConfig{}, fmt.Errorf("API key profile '%s' not found", keyName)
	}

	// Apply defaults
	if keyConfig.BaseURL == "" {
		keyConfig.BaseURL = c.DefaultBaseURL
	}
	if keyConfig.Timeout == 0 {
		keyConfig.Timeout = c.DefaultTimeout
	}

	return keyConfig, nil
}

// GetCurrentKeyName returns the name of the currently selected API key
func (c *Config) GetCurrentKeyName() string {
	profile := viper.GetString("profile")
	if profile != "" {
		return profile
	}

	if c.DefaultKey != "" {
		return c.DefaultKey
	}

	if os.Getenv("TONECLONE_API_KEY") != "" {
		return "environment"
	}

	return ""
}

// AddKey adds a new API key configuration
func (c *Config) AddKey(name string, key string, baseURL string) {
	if c.Keys == nil {
		c.Keys = make(map[string]APIKeyConfig)
	}

	if baseURL == "" {
		baseURL = c.DefaultBaseURL
	}

	c.Keys[name] = APIKeyConfig{
		Key:     key,
		BaseURL: baseURL,
		Timeout: c.DefaultTimeout,
	}

	// Set as default if it's the first key
	if c.DefaultKey == "" {
		c.DefaultKey = name
	}
}

// RemoveKey removes an API key configuration
func (c *Config) RemoveKey(name string) error {
	if _, exists := c.Keys[name]; !exists {
		return fmt.Errorf("API key profile '%s' not found", name)
	}

	delete(c.Keys, name)

	// If this was the default key, clear the default
	if c.DefaultKey == name {
		c.DefaultKey = ""
		// Set a new default if other keys exist
		for keyName := range c.Keys {
			c.DefaultKey = keyName
			break
		}
	}

	return nil
}

// ListKeys returns a list of configured API key names
func (c *Config) ListKeys() []string {
	var keys []string
	for name := range c.Keys {
		keys = append(keys, name)
	}
	return keys
}

// SetDefaultKey sets the default API key
func (c *Config) SetDefaultKey(name string) error {
	if _, exists := c.Keys[name]; !exists {
		return fmt.Errorf("API key profile '%s' not found", name)
	}

	c.DefaultKey = name
	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check that default key exists if specified
	if c.DefaultKey != "" {
		if _, exists := c.Keys[c.DefaultKey]; !exists {
			return fmt.Errorf("default key '%s' not found in configured keys", c.DefaultKey)
		}
	}

	// Validate each key configuration
	for name, keyConfig := range c.Keys {
		if keyConfig.Key == "" {
			return fmt.Errorf("API key for profile '%s' is empty", name)
		}

		// Validate key format
		if !isValidAPIKey(keyConfig.Key) {
			return fmt.Errorf("API key for profile '%s' has invalid format", name)
		}

		if keyConfig.BaseURL == "" {
			return fmt.Errorf("base URL for profile '%s' is empty", name)
		}

		if keyConfig.Timeout < 0 {
			return fmt.Errorf("timeout for profile '%s' cannot be negative", name)
		}
	}

	return nil
}

// GetConfigPath returns the default configuration file path
func GetConfigPath() (string, error) {
	// Check if config file is specified via flag
	if viper.ConfigFileUsed() != "" {
		return viper.ConfigFileUsed(), nil
	}

	// Default to ~/.toneclone.yaml
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".toneclone.yaml"), nil
}

// IsConfigured checks if the CLI is configured with at least one API key
func IsConfigured() bool {
	config, err := LoadConfig()
	if err != nil {
		return false
	}

	// Check if we have any API keys configured
	if len(config.Keys) > 0 {
		return true
	}

	// Check for environment variable
	return os.Getenv("TONECLONE_API_KEY") != ""
}

// Helper functions

func getEnvOrDefault(envKey, defaultValue string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	return defaultValue
}

func isValidAPIKey(key string) bool {
	// Validate ToneClone API key format
	return (len(key) > 10 &&
		(key[:8] == "tc_live_" || key[:8] == "tc_test_")) ||
		key == "test_key" // Allow test key for development
}
