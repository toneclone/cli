package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/pkg/client"
)

var (
	// Typos command flags
	typosFormat    string
	typosEnable    bool
	typosDisable   bool
	typosIntensity string
	typosRate      float64
	typosMaxChunk  int
	typosProtected []string
)

// Intensity presets
var intensityPresets = map[string]float64{
	"none":       0.0,
	"subtle":     0.005,
	"noticeable": 0.01,
	"high":       0.02,
}

// typosCmd represents the typos command group
var typosCmd = &cobra.Command{
	Use:   "typos",
	Short: "Manage typo imperfection settings",
	Long: `Configure random typo insertion to make generated output feel more human.

Controls include enabling/disabling, setting intensity or a custom rate,
limiting max typos per chunk, and protecting certain contexts from typos.

Examples:
  toneclone personas typos get my-persona
  toneclone personas typos set my-persona --enable --intensity subtle
  toneclone personas typos set my-persona --rate 0.008 --max-per-chunk 5`,
}

// typosGetCmd shows current typo settings
var typosGetCmd = &cobra.Command{
	Use:   "get <persona>",
	Short: "Show current typo settings",
	Long: `Display the current typo imperfection settings for a persona.

Examples:
  toneclone personas typos get my-persona
  toneclone personas typos get my-persona --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runTyposGet,
}

// typosSetCmd configures typo settings
var typosSetCmd = &cobra.Command{
	Use:   "set <persona>",
	Short: "Configure typo settings",
	Long: `Configure typo imperfection settings for a persona.

Intensity presets:
  none       - Rate 0.0 (disabled)
  subtle     - Rate 0.005 (0.5%)
  noticeable - Rate 0.01 (1.0%)
  high       - Rate 0.02 (2.0%)

The --rate flag overrides --intensity if both are provided.

Examples:
  toneclone personas typos set my-persona --enable --intensity subtle
  toneclone personas typos set my-persona --disable
  toneclone personas typos set my-persona --rate 0.008 --max-per-chunk 5
  toneclone personas typos set my-persona --protected urls,emails,code`,
	Args: cobra.ExactArgs(1),
	RunE: runTyposSet,
}

func init() {
	personasCmd.AddCommand(typosCmd)

	typosCmd.AddCommand(typosGetCmd)
	typosCmd.AddCommand(typosSetCmd)

	// Get flags
	typosGetCmd.Flags().StringVar(&typosFormat, "format", "table", "output format: table, json")

	// Set flags
	typosSetCmd.Flags().StringVar(&typosFormat, "format", "table", "output format: table, json")
	typosSetCmd.Flags().BoolVar(&typosEnable, "enable", false, "enable typos")
	typosSetCmd.Flags().BoolVar(&typosDisable, "disable", false, "disable typos")
	typosSetCmd.Flags().StringVar(&typosIntensity, "intensity", "", "preset intensity: none, subtle, noticeable, high")
	typosSetCmd.Flags().Float64Var(&typosRate, "rate", -1, "custom rate 0.0-0.02 (overrides intensity)")
	typosSetCmd.Flags().IntVar(&typosMaxChunk, "max-per-chunk", -1, "max typos per chunk, 0-100")
	typosSetCmd.Flags().StringSliceVar(&typosProtected, "protected", nil, "protected contexts (e.g., urls,emails,code)")
}

func runTyposGet(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	persona, err := validatePersona(ctx, apiClient, args[0])
	if err != nil {
		return fmt.Errorf("failed to resolve persona: %w", err)
	}

	if typosFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]interface{}{
			"personaId":     persona.PersonaID,
			"personaName":   persona.Name,
			"imperfections": persona.Imperfections,
		})
	}

	return outputTyposDetails(persona)
}

func runTyposSet(cmd *cobra.Command, args []string) error {
	if typosEnable && typosDisable {
		return fmt.Errorf("cannot use both --enable and --disable")
	}

	if typosIntensity != "" {
		if _, ok := intensityPresets[typosIntensity]; !ok {
			return fmt.Errorf("invalid intensity: %s (valid: none, subtle, noticeable, high)", typosIntensity)
		}
	}

	if typosRate != -1 && (typosRate < 0.0 || typosRate > 0.02) {
		return fmt.Errorf("rate must be between 0.0 and 0.02")
	}

	if typosMaxChunk != -1 && (typosMaxChunk < 0 || typosMaxChunk > 100) {
		return fmt.Errorf("max-per-chunk must be between 0 and 100")
	}

	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	persona, err := validatePersona(ctx, apiClient, args[0])
	if err != nil {
		return fmt.Errorf("failed to resolve persona: %w", err)
	}

	// Start from existing settings or defaults
	settings := persona.Imperfections
	if settings == nil {
		settings = &client.ImperfectionSettings{}
	}

	// Apply changes
	if typosEnable {
		settings.Enabled = true
	}
	if typosDisable {
		settings.Enabled = false
	}

	if typosRate != -1 {
		settings.Rate = typosRate
	} else if typosIntensity != "" {
		settings.Rate = intensityPresets[typosIntensity]
	}

	if typosMaxChunk != -1 {
		settings.MaxPerChunk = typosMaxChunk
	}

	if typosProtected != nil {
		settings.ProtectedContexts = typosProtected
	}

	// Update persona with new imperfection settings
	updatePersona := &client.Persona{
		Imperfections: settings,
	}

	updated, err := apiClient.Personas.Update(ctx, persona.PersonaID, updatePersona)
	if err != nil {
		return fmt.Errorf("failed to update typo settings: %w", err)
	}

	if typosFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]interface{}{
			"personaId":     updated.PersonaID,
			"personaName":   updated.Name,
			"imperfections": updated.Imperfections,
		})
	}

	fmt.Println("Typo settings updated successfully.")
	return outputTyposDetails(updated)
}

func outputTyposDetails(persona *client.Persona) error {
	fmt.Printf("Typo Settings for %q\n", persona.Name)
	fmt.Printf("===============================\n")

	if persona.Imperfections == nil {
		fmt.Printf("Enabled:            false\n")
		fmt.Printf("Intensity:          none\n")
		fmt.Printf("Rate:               0 (0.0%%)\n")
		fmt.Printf("Max per chunk:      0\n")
		fmt.Printf("Protected contexts: (none)\n")
		return nil
	}

	s := persona.Imperfections
	fmt.Printf("Enabled:            %t\n", s.Enabled)
	fmt.Printf("Intensity:          %s\n", rateToIntensity(s.Rate))
	fmt.Printf("Rate:               %g (%g%%)\n", s.Rate, s.Rate*100)
	fmt.Printf("Max per chunk:      %d\n", s.MaxPerChunk)

	if len(s.ProtectedContexts) > 0 {
		fmt.Printf("Protected contexts: %s\n", strings.Join(s.ProtectedContexts, ", "))
	} else {
		fmt.Printf("Protected contexts: (none)\n")
	}

	return nil
}

