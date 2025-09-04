package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/toneclone/cli/internal/config"
)

var (
	// Config command flags
	configFormat string
	configGlobal bool
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ToneClone CLI configuration",
	Long: `Manage ToneClone CLI configuration settings.

View, validate, and manage configuration files, API keys, and profiles.

Examples:
  toneclone config show
  toneclone config list
  toneclone config validate
  toneclone config path`,
}

// configShowCmd represents the show subcommand
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long: `Show the current ToneClone CLI configuration.

Displays the active configuration including API keys, profiles, and settings.

Examples:
  toneclone config show
  toneclone config show --format=json`,
	RunE: runConfigShow,
}

// configListCmd represents the list subcommand
var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configuration profiles",
	Long: `List all available configuration profiles.

Shows all configured API keys and their associated profiles.

Examples:
  toneclone config list
  toneclone config list --format=json`,
	RunE: runConfigList,
}

// configValidateCmd represents the validate subcommand
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration",
	Long: `Validate the ToneClone CLI configuration.

Checks configuration file syntax, API key validity, and profile settings.

Examples:
  toneclone config validate`,
	RunE: runConfigValidate,
}

// configPathCmd represents the path subcommand
var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Long: `Show the path to the ToneClone CLI configuration file.

Displays the location of the active configuration file.

Examples:
  toneclone config path`,
	RunE: runConfigPath,
}

// configInitCmd represents the init subcommand
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long: `Initialize a new ToneClone CLI configuration file.

Creates a new configuration file with default settings.

Examples:
  toneclone config init
  toneclone config init --global`,
	RunE: runConfigInit,
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Add subcommands
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)

	// Show command flags
	configShowCmd.Flags().StringVar(&configFormat, "format", "table", "output format: table, json")

	// List command flags
	configListCmd.Flags().StringVar(&configFormat, "format", "table", "output format: table, json")

	// Init command flags
	configInitCmd.Flags().BoolVar(&configGlobal, "global", false, "create global config file")
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Output configuration
	if configFormat == "json" {
		return outputConfigJSON(cfg)
	}

	return outputConfigTable(cfg)
}

func runConfigList(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Output API keys
	if configFormat == "json" {
		return outputKeysJSON(cfg.Keys)
	}

	return outputKeysTable(cfg.Keys)
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Validate current API key
	if cfg.DefaultKey != "" {
		keyConfig, err := cfg.GetCurrentKey()
		if err != nil {
			return fmt.Errorf("current API key validation failed: %w", err)
		}

		if keyConfig.Key == "" {
			return fmt.Errorf("current API key is empty")
		}

		if keyConfig.BaseURL == "" {
			return fmt.Errorf("current API key base URL is empty")
		}
	}

	fmt.Println("✓ Configuration is valid")
	fmt.Printf("  Config file: %s\n", viper.ConfigFileUsed())
	fmt.Printf("  Current key: %s\n", cfg.DefaultKey)
	fmt.Printf("  API keys: %d\n", len(cfg.Keys))

	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		configPath, _ = config.GetConfigPath()
	}
	fmt.Println(configPath)
	return nil
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	// Determine config file path
	var configPath string
	if configGlobal {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".toneclone.yaml")
	} else {
		configPath = ".toneclone.yaml"
	}

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists: %s", configPath)
	}

	// Create default configuration
	cfg := config.NewConfig()

	// Save configuration
	if err := cfg.SaveConfig(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Printf("✓ Configuration file created: %s\n", configPath)
	fmt.Println("  Add your first API key with: toneclone auth add")

	return nil
}

func outputConfigTable(cfg *config.Config) error {
	fmt.Printf("ToneClone CLI Configuration\n")
	fmt.Printf("===========================\n")
	fmt.Printf("Config File:  %s\n", viper.ConfigFileUsed())
	fmt.Printf("Current Key:  %s\n", cfg.DefaultKey)
	fmt.Printf("API Keys:     %d\n", len(cfg.Keys))

	if len(cfg.Keys) > 0 {
		fmt.Printf("\nAPI Keys:\n")

		// Create table writer
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		defer w.Flush()

		// Header
		fmt.Fprintln(w, "NAME\tBASE URL\tCURRENT")
		fmt.Fprintln(w, "----\t--------\t-------")

		// Data
		for name, keyConfig := range cfg.Keys {
			current := ""
			if name == cfg.DefaultKey {
				current = "✓"
			}

			fmt.Fprintf(w, "%s\t%s\t%s\n",
				name,
				keyConfig.BaseURL,
				current,
			)
		}
	}

	return nil
}

func outputConfigJSON(cfg *config.Config) error {
	// Create a sanitized version without API keys
	sanitized := map[string]interface{}{
		"config_file": viper.ConfigFileUsed(),
		"current_key": cfg.DefaultKey,
		"api_keys": func() map[string]interface{} {
			keys := make(map[string]interface{})
			for name, keyConfig := range cfg.Keys {
				keys[name] = map[string]interface{}{
					"base_url": keyConfig.BaseURL,
					"key":      "***REDACTED***",
				}
			}
			return keys
		}(),
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(sanitized)
}

func outputKeysTable(keys map[string]config.APIKeyConfig) error {
	if len(keys) == 0 {
		fmt.Println("No API keys configured.")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "NAME\tBASE URL\tKEY PREFIX")
	fmt.Fprintln(w, "----\t--------\t----------")

	// Data
	for name, keyConfig := range keys {
		keyPrefix := ""
		if len(keyConfig.Key) > 8 {
			keyPrefix = keyConfig.Key[:8] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			name,
			keyConfig.BaseURL,
			keyPrefix,
		)
	}

	return nil
}

func outputKeysJSON(keys map[string]config.APIKeyConfig) error {
	// Create a sanitized version without full API keys
	sanitized := make(map[string]interface{})
	for name, keyConfig := range keys {
		sanitized[name] = map[string]interface{}{
			"base_url": keyConfig.BaseURL,
			"key":      "***REDACTED***",
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"api_keys": sanitized,
		"count":    len(keys),
	})
}
