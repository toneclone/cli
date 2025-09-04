package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/toneclone/cli/internal/config"
	"github.com/toneclone/cli/pkg/client"
)

var (
	// User command flags
	userFormat string
)

// userCmd represents the user command
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User account management",
	Long: `Manage your ToneClone user account and settings.

View account information, manage settings, and check your usage statistics.

Examples:
  toneclone user whoami
  toneclone user info
  toneclone user settings`,
}

// whoamiCmd represents the whoami subcommand
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display current user information",
	Long: `Display information about the currently authenticated user.

Shows user ID, email, plan, and account creation date.

Examples:
  toneclone user whoami
  toneclone user whoami --format=json`,
	RunE: runWhoami,
}

// infoCmd represents the info subcommand (alias for whoami)
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display detailed user information",
	Long: `Display detailed information about the currently authenticated user.

Shows user ID, email, plan, account creation date, and usage statistics.

Examples:
  toneclone user info
  toneclone user info --format=json`,
	RunE: runUserInfo,
}

// settingsCmd represents the settings subcommand
var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage user account settings",
	Long: `Manage your ToneClone user account settings.

View and update account preferences, default settings, and configurations.

Examples:
  toneclone user settings
  toneclone user settings --format=json`,
	RunE: runUserSettings,
}

func init() {
	rootCmd.AddCommand(userCmd)

	// Add subcommands
	userCmd.AddCommand(whoamiCmd)
	userCmd.AddCommand(infoCmd)
	userCmd.AddCommand(settingsCmd)

	// Whoami command flags
	whoamiCmd.Flags().StringVar(&userFormat, "format", "table", "output format: table, json")

	// Info command flags
	infoCmd.Flags().StringVar(&userFormat, "format", "table", "output format: table, json")

	// Settings command flags
	settingsCmd.Flags().StringVar(&userFormat, "format", "table", "output format: table, json")
}

func runWhoami(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current API key
	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// Create API client
	apiClient := client.NewToneCloneClientFromConfig(
		keyConfig.BaseURL,
		keyConfig.Key,
		30*time.Second,
	)

	// Get user info
	ctx := context.Background()
	user, err := apiClient.WhoAmI(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Output user info
	if userFormat == "json" {
		return outputUserJSON(user)
	}

	return outputUserTable(user)
}

func runUserInfo(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current API key
	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// Create API client
	apiClient := client.NewToneCloneClientFromConfig(
		keyConfig.BaseURL,
		keyConfig.Key,
		30*time.Second,
	)

	// Get user info
	ctx := context.Background()
	user, err := apiClient.WhoAmI(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Output detailed user info
	if userFormat == "json" {
		return outputUserJSON(user)
	}

	return outputUserDetails(user)
}

func runUserSettings(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current API key
	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		return fmt.Errorf("authentication required: %w", err)
	}

	// Create API client
	apiClient := client.NewToneCloneClientFromConfig(
		keyConfig.BaseURL,
		keyConfig.Key,
		30*time.Second,
	)

	// Get user info
	ctx := context.Background()
	user, err := apiClient.WhoAmI(ctx)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Display current settings
	fmt.Printf("User Settings\n")
	fmt.Printf("=============\n")
	fmt.Printf("User ID:      %s\n", user.UserID)
	fmt.Printf("Email:        %s\n", user.Email)
	fmt.Printf("Plan:         %s\n", user.Plan)
	fmt.Printf("API Key:      %s\n", cfg.GetCurrentKeyName())
	fmt.Printf("Base URL:     %s\n", keyConfig.BaseURL)
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		configPath, _ = config.GetConfigPath()
	}
	fmt.Printf("Config File:  %s\n", configPath)

	return nil
}

func outputUserTable(user *client.User) error {
	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "USER ID\tEMAIL\tPLAN\tCREATED")
	fmt.Fprintln(w, "-------\t-----\t----\t-------")

	// Data
	created := formatTime(user.CreatedAt)
	plan := user.Plan
	if plan == "" {
		plan = "Free"
	}

	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
		user.UserID,
		user.Email,
		plan,
		created,
	)

	return nil
}

func outputUserDetails(user *client.User) error {
	fmt.Printf("User Information\n")
	fmt.Printf("================\n")
	fmt.Printf("User ID:      %s\n", user.UserID)
	fmt.Printf("Email:        %s\n", user.Email)
	if user.Name != "" {
		fmt.Printf("Name:         %s\n", user.Name)
	}
	plan := user.Plan
	if plan == "" {
		plan = "Free"
	}
	fmt.Printf("Plan:         %s\n", plan)
	fmt.Printf("Created:      %s\n", formatTime(user.CreatedAt))

	return nil
}

func outputUserJSON(user *client.User) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(user)
}
