package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/internal/config"
	"github.com/toneclone/cli/pkg/client"
)

var (
	// Persona command flags
	personaFormat      string
	personaSort        string
	personaFilter      string
	personaInteractive bool
	personaName        string
	personaPresetID    string
	personaForce       bool
	personaConfirm     bool
)

// personasCmd represents the personas command
var personasCmd = &cobra.Command{
	Use:   "personas",
	Short: "Manage ToneClone personas",
	Long: `Manage ToneClone personas - create, list, update, and delete writing personas.

Personas define the writing style, tone, and characteristics that ToneClone uses 
when generating text. Each persona can be trained with your own writing samples
to create a unique voice.

Examples:
  toneclone personas list
  toneclone personas list --filter="professional"
  toneclone personas get persona-id
  toneclone personas create --name="Blog Writer"
  toneclone personas update persona-id --name="New Name"
  toneclone personas delete persona-id`,
}

// listPersonasCmd represents the list subcommand
var listPersonasCmd = &cobra.Command{
	Use:   "list",
	Short: "List all personas",
	Long: `List all personas associated with your account.

The list can be filtered by name or type, and sorted by various criteria.
By default, personas are sorted by last used date (most recent first).

Examples:
  toneclone personas list
  toneclone personas list --filter="professional"
  toneclone personas list --sort="name"
  toneclone personas list --format="json"`,
	RunE: runListPersonas,
}

// getPersonaCmd represents the get subcommand
var getPersonaCmd = &cobra.Command{
	Use:   "get <persona-id>",
	Short: "Get detailed information about a persona",
	Long: `Get detailed information about a specific persona.

Shows all metadata including training status, associated files, and usage statistics.

Examples:
  toneclone personas get persona-id
  toneclone personas get persona-id --format="json"`,
	Args: cobra.ExactArgs(1),
	RunE: runGetPersona,
}

// createPersonaCmd represents the create subcommand
var createPersonaCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new persona",
	Long: `Create a new persona with the specified name.

The persona type and other characteristics will be determined automatically
by the system based on your usage and training data.

Examples:
  toneclone personas create --name="Professional Writer"
  toneclone personas create --name="Casual Blogger" --preset="blogger"
  toneclone personas create --interactive`,
	RunE: runCreatePersona,
}

// updatePersonaCmd represents the update subcommand
var updatePersonaCmd = &cobra.Command{
	Use:   "update <persona-id>",
	Short: "Update an existing persona",
	Long: `Update the properties of an existing persona.

You can update the name and other properties of a persona.
Note: persona type is determined automatically by the system.

Examples:
  toneclone personas update persona-id --name="New Name"`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdatePersona,
}

// deletePersonaCmd represents the delete subcommand
var deletePersonaCmd = &cobra.Command{
	Use:   "delete <persona-id>",
	Short: "Delete a persona",
	Long: `Delete a persona permanently.

This action cannot be undone. All associated training data and files
will be disassociated (but not deleted).

Examples:
  toneclone personas delete persona-id
  toneclone personas delete persona-id --confirm`,
	Args: cobra.ExactArgs(1),
	RunE: runDeletePersona,
}

func init() {
	rootCmd.AddCommand(personasCmd)

	// Add subcommands
	personasCmd.AddCommand(listPersonasCmd)
	personasCmd.AddCommand(getPersonaCmd)
	personasCmd.AddCommand(createPersonaCmd)
	personasCmd.AddCommand(updatePersonaCmd)
	personasCmd.AddCommand(deletePersonaCmd)

	// List command flags
	listPersonasCmd.Flags().StringVar(&personaFormat, "format", "table", "output format: table, json")
	listPersonasCmd.Flags().StringVar(&personaSort, "sort", "last_used", "sort by: name, type, status, last_used, created")
	listPersonasCmd.Flags().StringVar(&personaFilter, "filter", "", "filter personas by name or type")

	// Get command flags
	getPersonaCmd.Flags().StringVar(&personaFormat, "format", "table", "output format: table, json")

	// Create command flags
	createPersonaCmd.Flags().StringVar(&personaName, "name", "", "persona name")
	createPersonaCmd.Flags().StringVar(&personaPresetID, "preset", "", "preset ID to use for persona creation")
	createPersonaCmd.Flags().BoolVar(&personaInteractive, "interactive", false, "interactive persona creation")
	createPersonaCmd.Flags().StringVar(&personaFormat, "format", "table", "output format: table, json")

	// Update command flags
	updatePersonaCmd.Flags().StringVar(&personaName, "name", "", "new persona name")
	updatePersonaCmd.Flags().StringVar(&personaFormat, "format", "table", "output format: table, json")

	// Delete command flags
	deletePersonaCmd.Flags().BoolVar(&personaConfirm, "confirm", false, "skip confirmation prompt")
}

func runListPersonas(cmd *cobra.Command, args []string) error {
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

	// Get personas (both user and built-in)
	ctx := context.Background()
	userPersonas, err := apiClient.Personas.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list user personas: %w", err)
	}

	builtInPersonas, err := apiClient.Personas.ListBuiltIn(ctx)
	if err != nil {
		return fmt.Errorf("failed to list built-in personas: %w", err)
	}

	// Mark built-in personas and combine lists
	for i := range builtInPersonas {
		builtInPersonas[i].IsBuiltIn = true
	}
	
	// Combine personas with user personas first
	personas := append(userPersonas, builtInPersonas...)

	// Filter personas
	if personaFilter != "" {
		personas = filterPersonas(personas, personaFilter)
	}

	// Sort personas
	sortPersonas(personas, personaSort)

	// Output personas
	if personaFormat == "json" {
		return outputPersonasJSON(personas)
	}

	return outputPersonasTable(personas)
}

func runGetPersona(cmd *cobra.Command, args []string) error {
	personaID := args[0]

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

	// Get persona (supports both name and ID)
	ctx := context.Background()
	persona, err := validatePersona(ctx, apiClient, personaID)
	if err != nil {
		return fmt.Errorf("failed to get persona: %w", err)
	}

	// Output persona
	if personaFormat == "json" {
		return outputPersonaJSON(persona)
	}

	return outputPersonaDetails(persona)
}

func runCreatePersona(cmd *cobra.Command, args []string) error {
	// Interactive mode
	if personaInteractive {
		return runInteractivePersonaCreation()
	}

	// Validate required flags
	if personaName == "" {
		return fmt.Errorf("persona name is required (use --name or --interactive)")
	}

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

	// Create persona
	persona := &client.Persona{
		Name: personaName,
	}

	ctx := context.Background()
	created, err := apiClient.Personas.Create(ctx, persona)
	if err != nil {
		return fmt.Errorf("failed to create persona: %w", err)
	}

	fmt.Printf("✓ Persona '%s' created successfully\n", created.Name)
	fmt.Printf("  ID: %s\n", created.PersonaID)
	fmt.Printf("  Type: %s\n", created.PersonaType)
	fmt.Printf("  Status: %s\n", created.Status)

	return nil
}

func runUpdatePersona(cmd *cobra.Command, args []string) error {
	personaID := args[0]

	// Check if any update flags are provided
	if personaName == "" {
		return fmt.Errorf("at least one update flag must be provided (--name)")
	}

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

	ctx := context.Background()

	// Get existing persona
	existing, err := validatePersona(ctx, apiClient, personaID)
	if err != nil {
		return fmt.Errorf("failed to get persona: %w", err)
	}

	// Update fields
	if personaName != "" {
		existing.Name = personaName
	}

	// Update persona
	updated, err := apiClient.Personas.Update(ctx, personaID, existing)
	if err != nil {
		return fmt.Errorf("failed to update persona: %w", err)
	}

	fmt.Printf("✓ Persona updated successfully\n")
	fmt.Printf("  Name: %s\n", updated.Name)
	fmt.Printf("  Type: %s\n", updated.PersonaType)
	fmt.Printf("  Status: %s\n", updated.Status)

	return nil
}

func runDeletePersona(cmd *cobra.Command, args []string) error {
	personaID := args[0]

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

	ctx := context.Background()

	// Get persona info for confirmation
	persona, err := validatePersona(ctx, apiClient, personaID)
	if err != nil {
		return fmt.Errorf("failed to get persona: %w", err)
	}

	// Confirm deletion
	if !personaConfirm {
		fmt.Printf("Are you sure you want to delete persona '%s' (%s)? [y/N]: ", persona.Name, persona.PersonaID)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete persona
	err = apiClient.Personas.Delete(ctx, personaID)
	if err != nil {
		return fmt.Errorf("failed to delete persona: %w", err)
	}

	fmt.Printf("✓ Persona '%s' deleted successfully\n", persona.Name)
	return nil
}

func runInteractivePersonaCreation() error {
	fmt.Println("Interactive Persona Creation")
	fmt.Println("============================")

	// Get persona name
	fmt.Print("Enter persona name: ")
	var name string
	fmt.Scanln(&name)
	if name == "" {
		return fmt.Errorf("persona name is required")
	}

	// Set the values for the create function
	personaName = name

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

	// Create persona
	persona := &client.Persona{
		Name: personaName,
	}

	ctx := context.Background()
	created, err := apiClient.Personas.Create(ctx, persona)
	if err != nil {
		return fmt.Errorf("failed to create persona: %w", err)
	}

	fmt.Printf("\n✓ Persona '%s' created successfully\n", created.Name)
	fmt.Printf("  ID: %s\n", created.PersonaID)
	fmt.Printf("  Type: %s\n", created.PersonaType)
	fmt.Printf("  Status: %s\n", created.Status)

	return nil
}

func filterPersonas(personas []client.Persona, filter string) []client.Persona {
	if filter == "" {
		return personas
	}

	var filtered []client.Persona
	filter = strings.ToLower(filter)

	for _, persona := range personas {
		if strings.Contains(strings.ToLower(persona.Name), filter) ||
			strings.Contains(strings.ToLower(persona.PersonaType), filter) ||
			strings.Contains(strings.ToLower(persona.Status), filter) {
			filtered = append(filtered, persona)
		}
	}

	return filtered
}

func sortPersonas(personas []client.Persona, sortBy string) {
	switch sortBy {
	case "name":
		sort.Slice(personas, func(i, j int) bool {
			return personas[i].Name < personas[j].Name
		})
	case "type":
		sort.Slice(personas, func(i, j int) bool {
			return personas[i].PersonaType < personas[j].PersonaType
		})
	case "status":
		sort.Slice(personas, func(i, j int) bool {
			return personas[i].Status < personas[j].Status
		})
	case "created":
		sort.Slice(personas, func(i, j int) bool {
			return personas[i].LastModifiedAt.Before(personas[j].LastModifiedAt)
		})
	case "last_used":
		fallthrough
	default:
		sort.Slice(personas, func(i, j int) bool {
			return personas[i].LastUsedAt.After(personas[j].LastUsedAt)
		})
	}
}

func outputPersonasTable(personas []client.Persona) error {
	if len(personas) == 0 {
		fmt.Println("No personas found.")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "NAME\tTYPE\tSTATUS\tTRAINING\tLAST USED\tSOURCE\tID")
	fmt.Fprintln(w, "----\t----\t------\t--------\t---------\t------\t--")

	// Rows
	for _, persona := range personas {
		lastUsed := formatTime(persona.LastUsedAt)
		source := "User"
		if persona.IsBuiltIn {
			source = "Built-in"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			persona.Name,
			persona.PersonaType,
			persona.Status,
			persona.TrainingStatus,
			lastUsed,
			source,
			persona.PersonaID,
		)
	}

	return nil
}

func outputPersonasJSON(personas []client.Persona) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"personas": personas,
		"count":    len(personas),
	})
}

func outputPersonaDetails(persona *client.Persona) error {
	fmt.Printf("Persona Details\n")
	fmt.Printf("===============\n")
	fmt.Printf("Name:             %s\n", persona.Name)
	fmt.Printf("ID:               %s\n", persona.PersonaID)
	fmt.Printf("Type:             %s\n", persona.PersonaType)
	fmt.Printf("Status:           %s\n", persona.Status)
	fmt.Printf("Training Status:  %s\n", persona.TrainingStatus)
	fmt.Printf("Voice Evolution:  %t\n", persona.VoiceEvolution)
	source := "User"
	if persona.IsBuiltIn {
		source = "Built-in"
	}
	fmt.Printf("Source:           %s\n", source)
	fmt.Printf("Last Used:        %s\n", formatTime(persona.LastUsedAt))
	fmt.Printf("Last Modified:    %s\n", formatTime(persona.LastModifiedAt))

	if persona.PromptDescription != "" {
		fmt.Printf("Description:      %s\n", persona.PromptDescription)
	}

	return nil
}

func outputPersonaJSON(persona *client.Persona) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(persona)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(diff.Hours()))
	} else if diff < 30*24*time.Hour {
		return fmt.Sprintf("%d days ago", int(diff.Hours()/24))
	} else {
		return t.Format("2006-01-02")
	}
}
