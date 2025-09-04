package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/toneclone/cli/internal/config"
	"github.com/toneclone/cli/pkg/client"
)

var (
	keyName        string
	baseURL        string
	setDefault     bool
	fromStdin      bool
	force          bool
	skipValidation bool
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication and API keys",
	Long: `Manage authentication and API keys for ToneClone CLI.

This command group allows you to:
- Login with an API key
- Logout and remove stored credentials  
- List configured API keys
- Switch between different API key profiles
- Check authentication status

Examples:
  toneclone auth login                     # Interactive login
  toneclone auth login --key tc_live_xxx   # Login with specific key
  toneclone auth login --from-stdin        # Read key from stdin (for CI/CD)
  toneclone auth logout                    # Remove default profile
  toneclone auth list                      # List all configured profiles
  toneclone auth status                    # Check current authentication
  toneclone auth switch --profile prod     # Switch to 'prod' profile`,
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with an API key",
	Long: `Login with a ToneClone API key.

You can provide the API key in several ways:
1. Interactive prompt (default)
2. --key flag
3. --from-stdin flag (useful for CI/CD)
4. TONECLONE_API_KEY environment variable

The API key will be validated before being saved to your configuration.

Examples:
  toneclone auth login
  toneclone auth login --key tc_live_abc123 --name production
  toneclone auth login --from-stdin --name ci-cd
  echo "tc_live_abc123" | toneclone auth login --from-stdin`,
	RunE: runLogin,
}

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout [profile-name]",
	Short: "Logout and remove stored credentials",
	Long: `Remove stored API key credentials.

If no profile name is provided, removes the default profile.
Use --all to remove all stored credentials.

Examples:
  toneclone auth logout                # Remove default profile
  toneclone auth logout production     # Remove 'production' profile
  toneclone auth logout --all          # Remove all profiles`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLogout,
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured API key profiles",
	Long: `List all configured API key profiles.

Shows profile names, base URLs, and which profile is currently default.

Example:
  toneclone auth list`,
	RunE: runList,
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check authentication status",
	Long: `Check current authentication status and validate API key.

Shows:
- Current profile being used
- API key prefix (redacted)
- Base URL
- Connection status
- User information (if available)

Example:
  toneclone auth status`,
	RunE: runStatus,
}

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch to a different API key profile",
	Long: `Switch the default API key profile.

Changes which profile is used by default when no --profile flag is specified.

Example:
  toneclone auth switch --profile production`,
	RunE: runSwitch,
}

func init() {
	rootCmd.AddCommand(authCmd)

	// Add subcommands
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(listCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(switchCmd)

	// Login flags
	loginCmd.Flags().StringVar(&keyName, "name", "", "name for this API key profile")
	loginCmd.Flags().StringVar(&baseURL, "base-url", "", "base URL for the API (default: https://api.toneclone.ai)")
	loginCmd.Flags().BoolVar(&setDefault, "set-default", true, "set as default profile")
	loginCmd.Flags().BoolVar(&fromStdin, "from-stdin", false, "read API key from stdin (useful for CI/CD)")
	loginCmd.Flags().BoolVar(&force, "force", false, "overwrite existing profile without confirmation")
	loginCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "skip API key validation (useful for development)")

	// Logout flags
	logoutCmd.Flags().Bool("all", false, "remove all profiles")

	// Switch flags
	switchCmd.Flags().String("profile", "", "profile name to switch to")
	switchCmd.MarkFlagRequired("profile")
}

func runLogin(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load existing config
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.NewConfig()
	}

	// Get API key
	var apiKey string
	if fromStdin {
		apiKey, err = readAPIKeyFromStdin()
		if err != nil {
			return fmt.Errorf("failed to read API key from stdin: %w", err)
		}
	} else if envKey := os.Getenv("TONECLONE_API_KEY"); envKey != "" {
		apiKey = envKey
		fmt.Println("Using API key from TONECLONE_API_KEY environment variable")
	} else {
		apiKey, err = promptForAPIKey()
		if err != nil {
			return fmt.Errorf("failed to get API key: %w", err)
		}
	}

	// Determine profile name
	profileName := keyName
	if profileName == "" {
		if fromStdin {
			profileName = "default"
		} else {
			profileName, err = promptForProfileName()
			if err != nil {
				return fmt.Errorf("failed to get profile name: %w", err)
			}
		}
	}

	// Check if profile already exists
	if _, exists := cfg.Keys[profileName]; exists && !force {
		if !fromStdin {
			fmt.Printf("Profile '%s' already exists. Overwrite? (y/N): ", profileName)
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			if strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
				return fmt.Errorf("aborted")
			}
		} else {
			return fmt.Errorf("profile '%s' already exists, use --force to overwrite", profileName)
		}
	}

	// Determine base URL
	apiBaseURL := baseURL
	if apiBaseURL == "" {
		// Check environment variable first
		if envURL := os.Getenv("TONECLONE_BASE_URL"); envURL != "" {
			apiBaseURL = envURL
		} else {
			apiBaseURL = cfg.DefaultBaseURL
		}
	}

	// Validate API key (unless skipped)
	if !skipValidation {
		fmt.Print("Validating API key...")
		testClient := client.NewToneCloneClient(apiKey, client.WithBaseURL(apiBaseURL))

		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := testClient.ValidateConnection(ctx); err != nil {
			fmt.Println(" ✗")
			return fmt.Errorf("API key validation failed: %w", err)
		}
		fmt.Println(" ✓")

		// Get user info for confirmation
		user, err := testClient.WhoAmI(ctx)
		if err == nil {
			fmt.Printf("Successfully authenticated as: %s\n", user.Email)
		}
	} else {
		fmt.Println("⚠️  Skipping API key validation")
	}

	// Add to config
	cfg.AddKey(profileName, apiKey, apiBaseURL)

	if setDefault {
		cfg.DefaultKey = profileName
	}

	// Save config
	configPath, err := config.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	if err := cfg.SaveConfig(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✓ API key saved as profile '%s'\n", profileName)
	if setDefault {
		fmt.Printf("✓ Set '%s' as default profile\n", profileName)
	}

	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	removeAll, _ := cmd.Flags().GetBool("all")

	if removeAll {
		cfg.Keys = make(map[string]config.APIKeyConfig)
		cfg.DefaultKey = ""
		fmt.Println("Removed all API key profiles")
	} else {
		profileName := ""
		if len(args) > 0 {
			profileName = args[0]
		} else {
			profileName = cfg.DefaultKey
		}

		if profileName == "" {
			return fmt.Errorf("no profile specified and no default profile set")
		}

		if err := cfg.RemoveKey(profileName); err != nil {
			return err
		}

		fmt.Printf("Removed profile '%s'\n", profileName)
	}

	// Save config
	configPath, err := config.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	return cfg.SaveConfig(configPath)
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Keys) == 0 {
		fmt.Println("No API key profiles configured.")
		fmt.Println("Run 'toneclone auth login' to add one.")
		return nil
	}

	fmt.Println("Configured API key profiles:")
	fmt.Println()

	for name, keyConfig := range cfg.Keys {
		isDefault := name == cfg.DefaultKey
		defaultMarker := ""
		if isDefault {
			defaultMarker = " (default)"
		}

		// Redact API key for display
		redactedKey := redactAPIKey(keyConfig.Key)

		fmt.Printf("  %s%s\n", name, defaultMarker)
		fmt.Printf("    Key: %s\n", redactedKey)
		fmt.Printf("    URL: %s\n", keyConfig.BaseURL)
		fmt.Println()
	}

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	currentKeyName := cfg.GetCurrentKeyName()
	if currentKeyName == "" {
		fmt.Println("Not authenticated")
		fmt.Println("Run 'toneclone auth login' to authenticate")
		return nil
	}

	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		return err
	}

	fmt.Printf("Current profile: %s\n", currentKeyName)
	fmt.Printf("API key: %s\n", redactAPIKey(keyConfig.Key))
	fmt.Printf("Base URL: %s\n", keyConfig.BaseURL)

	// Test connection
	fmt.Print("Testing connection...")
	testClient := client.NewToneCloneClient(keyConfig.Key, client.WithBaseURL(keyConfig.BaseURL))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := testClient.ValidateConnection(ctx); err != nil {
		fmt.Println(" ✗")
		fmt.Printf("Connection failed: %v\n", err)
		return nil
	}
	fmt.Println(" ✓")

	// Get user info
	user, err := testClient.WhoAmI(ctx)
	if err != nil {
		fmt.Printf("Failed to get user info: %v\n", err)
		return nil
	}

	fmt.Printf("Authenticated as: %s\n", user.Email)
	if user.Plan != "" {
		fmt.Printf("Plan: %s\n", user.Plan)
	}

	return nil
}

func runSwitch(cmd *cobra.Command, args []string) error {
	profileName, _ := cmd.Flags().GetString("profile")

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.SetDefaultKey(profileName); err != nil {
		return err
	}

	// Save config
	configPath, err := config.GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	if err := cfg.SaveConfig(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Switched default profile to '%s'\n", profileName)
	return nil
}

// Helper functions

func readAPIKeyFromStdin() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("no input received")
	}

	apiKey := strings.TrimSpace(scanner.Text())
	if apiKey == "" {
		return "", fmt.Errorf("empty API key")
	}

	return apiKey, nil
}

func promptForAPIKey() (string, error) {
	fmt.Print("Enter your ToneClone API key: ")

	// Read password-style input (hidden)
	keyBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	fmt.Println() // Add newline after hidden input

	apiKey := strings.TrimSpace(string(keyBytes))
	if apiKey == "" {
		return "", fmt.Errorf("empty API key")
	}

	return apiKey, nil
}

func promptForProfileName() (string, error) {
	fmt.Print("Enter a name for this profile (default: default): ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	name := strings.TrimSpace(scanner.Text())
	if name == "" {
		name = "default"
	}

	return name, nil
}

func redactAPIKey(key string) string {
	if len(key) <= 12 {
		return "****"
	}

	// Find prefix end
	prefixEnd := 8
	if len(key) <= prefixEnd+4 {
		return key[:prefixEnd] + "****"
	}

	prefix := key[:prefixEnd]
	suffix := key[len(key)-4:]
	return prefix + "****" + suffix
}
