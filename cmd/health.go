package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/internal/config"
	"github.com/toneclone/cli/pkg/client"
)

var (
	// Health command flags
	healthFormat  string
	healthVerbose bool
	healthTimeout int
)

// healthCmd represents the health command
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check ToneClone service health",
	Long: `Check the health and connectivity of ToneClone services.

Performs health checks on the API, authentication, and key services.
Use this command to diagnose connectivity issues or service problems.

Examples:
  toneclone health
  toneclone health --verbose
  toneclone health --format=json
  toneclone health --timeout=10`,
	RunE: runHealth,
}

// pingCmd represents the ping subcommand
var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping ToneClone API",
	Long: `Ping the ToneClone API to check basic connectivity.

This is a lightweight health check that doesn't require authentication.

Examples:
  toneclone ping
  toneclone ping --timeout=5`,
	RunE: runPing,
}

// healthStatusCmd represents the status subcommand
var healthStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check comprehensive service status",
	Long: `Check comprehensive status of ToneClone services.

Performs detailed health checks including API connectivity, authentication,
user account status, and service availability.

Examples:
  toneclone health status
  toneclone health status --verbose
  toneclone health status --format=json`,
	RunE: runHealthStatus,
}

type HealthCheck struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
	Details  interface{}   `json:"details,omitempty"`
}

type HealthResult struct {
	Overall   string        `json:"overall"`
	Timestamp time.Time     `json:"timestamp"`
	Checks    []HealthCheck `json:"checks"`
	Summary   string        `json:"summary"`
}

func init() {
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(pingCmd)

	// Add status as subcommand of health
	healthCmd.AddCommand(healthStatusCmd)

	// Health command flags
	healthCmd.Flags().StringVar(&healthFormat, "format", "table", "output format: table, json")
	healthCmd.Flags().BoolVar(&healthVerbose, "verbose", false, "verbose output")
	healthCmd.Flags().IntVar(&healthTimeout, "timeout", 10, "timeout in seconds")

	// Ping command flags
	pingCmd.Flags().IntVar(&healthTimeout, "timeout", 5, "timeout in seconds")

	// Status command flags
	healthStatusCmd.Flags().StringVar(&healthFormat, "format", "table", "output format: table, json")
	healthStatusCmd.Flags().BoolVar(&healthVerbose, "verbose", false, "verbose output")
	healthStatusCmd.Flags().IntVar(&healthTimeout, "timeout", 10, "timeout in seconds")
}

func runHealth(cmd *cobra.Command, args []string) error {
	return performHealthChecks(false)
}

func runPing(cmd *cobra.Command, args []string) error {
	return performPingCheck()
}

func runHealthStatus(cmd *cobra.Command, args []string) error {
	return performHealthChecks(true)
}

func performPingCheck() error {
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
		time.Duration(healthTimeout)*time.Second,
	)

	// Ping API
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(healthTimeout)*time.Second)
	defer cancel()

	err = apiClient.Ping(ctx)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("PING %s: FAILED (%v) - %s\n", keyConfig.BaseURL, duration, err.Error())
		return err
	}

	fmt.Printf("PING %s: OK (%v)\n", keyConfig.BaseURL, duration)
	return nil
}

func performHealthChecks(comprehensive bool) error {
	result := &HealthResult{
		Timestamp: time.Now(),
		Checks:    []HealthCheck{},
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		result.Overall = "CRITICAL"
		result.Summary = "Failed to load configuration"
		if healthFormat == "json" {
			return outputHealthJSON(result)
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get current API key
	keyConfig, err := cfg.GetCurrentKey()
	if err != nil {
		result.Overall = "CRITICAL"
		result.Summary = "Authentication required"
		if healthFormat == "json" {
			return outputHealthJSON(result)
		}
		return fmt.Errorf("authentication required: %w", err)
	}

	// Create API client
	apiClient := client.NewToneCloneClientFromConfig(
		keyConfig.BaseURL,
		keyConfig.Key,
		time.Duration(healthTimeout)*time.Second,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(healthTimeout)*time.Second)
	defer cancel()

	// Check 1: API Connectivity
	check1 := performCheck("API Connectivity", func() error {
		return apiClient.Ping(ctx)
	})
	result.Checks = append(result.Checks, check1)

	// Check 2: Authentication
	check2 := performCheck("Authentication", func() error {
		return apiClient.ValidateAPIKey(ctx)
	})
	result.Checks = append(result.Checks, check2)

	if comprehensive {
		// Check 3: User Account
		check3 := performCheck("User Account", func() error {
			_, err := apiClient.WhoAmI(ctx)
			return err
		})
		result.Checks = append(result.Checks, check3)

		// Check 4: Personas Service
		check4 := performCheck("Personas Service", func() error {
			_, err := apiClient.Personas.List(ctx)
			return err
		})
		result.Checks = append(result.Checks, check4)

		// Check 5: Knowledge Service
		check5 := performCheck("Knowledge Service", func() error {
			_, err := apiClient.Knowledge.List(ctx)
			return err
		})
		result.Checks = append(result.Checks, check5)
	}

	// Determine overall status
	result.Overall = "OK"
	failedChecks := 0
	for _, check := range result.Checks {
		if check.Status == "FAILED" {
			result.Overall = "CRITICAL"
			failedChecks++
		}
	}

	if failedChecks == 0 {
		result.Summary = "All systems operational"
	} else {
		result.Summary = fmt.Sprintf("%d/%d checks failed", failedChecks, len(result.Checks))
	}

	// Output results
	if healthFormat == "json" {
		return outputHealthJSON(result)
	}

	return outputHealthTable(result)
}

func performCheck(name string, checkFunc func() error) HealthCheck {
	start := time.Now()
	err := checkFunc()
	duration := time.Since(start)

	check := HealthCheck{
		Name:     name,
		Duration: duration,
	}

	if err != nil {
		check.Status = "FAILED"
		check.Error = err.Error()
	} else {
		check.Status = "OK"
	}

	return check
}

func outputHealthTable(result *HealthResult) error {
	// Overall status
	fmt.Printf("ToneClone Health Check\n")
	fmt.Printf("=====================\n")
	fmt.Printf("Overall Status: %s\n", result.Overall)
	fmt.Printf("Timestamp: %s\n", result.Timestamp.Format(time.RFC3339))
	fmt.Printf("Summary: %s\n\n", result.Summary)

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "CHECK\tSTATUS\tDURATION\tERROR")
	fmt.Fprintln(w, "-----\t------\t--------\t-----")

	// Data
	for _, check := range result.Checks {
		errorMsg := ""
		if check.Error != "" {
			errorMsg = check.Error
			if len(errorMsg) > 50 {
				errorMsg = errorMsg[:47] + "..."
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%v\t%s\n",
			check.Name,
			check.Status,
			check.Duration.Round(time.Millisecond),
			errorMsg,
		)
	}

	return nil
}

func outputHealthJSON(result *HealthResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
