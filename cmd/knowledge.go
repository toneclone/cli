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
	// Knowledge command flags
	knowledgeFormat       string
	knowledgeSort         string
	knowledgeFilter       string
	knowledgeInteractive  bool
	knowledgeName         string
	knowledgeInstructions string
	knowledgeAppend       string
	knowledgeConfirm      bool
	knowledgePersona      string
)

// knowledgeCmd represents the knowledge command
var knowledgeCmd = &cobra.Command{
	Use:   "knowledge",
	Short: "Manage ToneClone knowledge cards",
	Long: `Manage ToneClone knowledge cards - create, list, update, and delete writing knowledge cards.

Knowledge cards define writing instructions and context that can be used with personas
to customize the writing style and format for specific use cases.

Examples:
  toneclone knowledge list
  toneclone knowledge list --filter="email"
  toneclone knowledge get "Email Template"
  toneclone knowledge create --name="Email" --instructions="Write professional emails"
  toneclone knowledge update "Email Template" --name="New Name"
  toneclone knowledge delete "Email Template"
  toneclone knowledge associate --knowledge="Email Template" --persona=Professional`,
}

// listKnowledgeCmd represents the list subcommand
var listKnowledgeCmd = &cobra.Command{
	Use:   "list",
	Short: "List all knowledge cards",
	Long: `List all knowledge cards associated with your account.

The list can be filtered by name and sorted by various criteria.
By default, knowledge cards are sorted by creation date (most recent first).

Examples:
  toneclone knowledge list
  toneclone knowledge list --filter="email"
  toneclone knowledge list --sort="name"
  toneclone knowledge list --format="json"`,
	RunE: runListKnowledge,
}

// getKnowledgeCardCmd represents the get subcommand
var getKnowledgeCardCmd = &cobra.Command{
	Use:   "get <knowledge-card-name-or-id>",
	Short: "Get detailed information about a knowledge card",
	Long: `Get detailed information about a specific knowledge card by name or ID.

Shows all metadata including instructions, creation date, and usage information.

Examples:
  toneclone knowledge get "Email Template"
  toneclone knowledge get knowledge-card-id
  toneclone knowledge get "Email Template" --format="json"`,
	Args: cobra.ExactArgs(1),
	RunE: runGetKnowledgeCard,
}

// createKnowledgeCardCmd represents the create subcommand
var createKnowledgeCardCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new knowledge card",
	Long: `Create a new knowledge card with the specified name and instructions.

Knowledge cards define writing instructions and context that can be used with personas
to customize the writing style and format for specific use cases.

Examples:
  toneclone knowledge create --name="Email" --instructions="Write professional emails"
  toneclone knowledge create --name="Blog Post" --instructions="Write engaging blog posts"
  toneclone knowledge create --interactive`,
	RunE: runCreateKnowledgeCard,
}

// updateKnowledgeCardCmd represents the update subcommand
var updateKnowledgeCardCmd = &cobra.Command{
	Use:   "update <knowledge-card-name-or-id>",
	Short: "Update an existing knowledge card",
	Long: `Update the properties of an existing knowledge card by name or ID.

You can update the name and instructions of a knowledge card, or append text to existing instructions.

Examples:
  toneclone knowledge update "Email Template" --name="New Name"
  toneclone knowledge update knowledge-card-id --instructions="New instructions"
  toneclone knowledge update "Email Template" --append=" Also include examples."`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdateKnowledgeCard,
}

// deleteKnowledgeCardCmd represents the delete subcommand
var deleteKnowledgeCardCmd = &cobra.Command{
	Use:   "delete <knowledge-card-name-or-id>",
	Short: "Delete a knowledge card",
	Long: `Delete a knowledge card permanently by name or ID.

This action cannot be undone. The knowledge card will be disassociated from all personas.

Examples:
  toneclone knowledge delete "Email Template"
  toneclone knowledge delete knowledge-card-id
  toneclone knowledge delete "Email Template" --confirm`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteKnowledgeCard,
}

// associateKnowledgeCmd represents the associate subcommand
var associateKnowledgeCmd = &cobra.Command{
	Use:   "associate",
	Short: "Associate a knowledge card with a persona",
	Long: `Associate a knowledge card with a persona for use in text generation.

The knowledge card will be available when generating text with the specified persona.
Both knowledge card and persona can be specified by name or ID.

Examples:
  toneclone knowledge associate --knowledge="Email Template" --persona=Professional
  toneclone knowledge associate --knowledge=knowledge-card-id --persona=persona-id`,
	RunE: runAssociateKnowledgeCard,
}

// disassociateKnowledgeCmd represents the disassociate subcommand
var disassociateKnowledgeCmd = &cobra.Command{
	Use:   "disassociate",
	Short: "Disassociate a knowledge card from a persona",
	Long: `Disassociate a knowledge card from a persona.

The knowledge card will no longer be available when generating text with the specified persona.
Both knowledge card and persona can be specified by name or ID.

Examples:
  toneclone knowledge disassociate --knowledge="Email Template" --persona=Professional
  toneclone knowledge disassociate --knowledge=knowledge-card-id --persona=persona-id`,
	RunE: runDisassociateKnowledgeCard,
}

func init() {
	rootCmd.AddCommand(knowledgeCmd)

	// Add subcommands
	knowledgeCmd.AddCommand(listKnowledgeCmd)
	knowledgeCmd.AddCommand(getKnowledgeCardCmd)
	knowledgeCmd.AddCommand(createKnowledgeCardCmd)
	knowledgeCmd.AddCommand(updateKnowledgeCardCmd)
	knowledgeCmd.AddCommand(deleteKnowledgeCardCmd)
	knowledgeCmd.AddCommand(associateKnowledgeCmd)
	knowledgeCmd.AddCommand(disassociateKnowledgeCmd)

	// List command flags
	listKnowledgeCmd.Flags().StringVar(&knowledgeFormat, "format", "table", "output format: table, json")
	listKnowledgeCmd.Flags().StringVar(&knowledgeSort, "sort", "created", "sort by: name, created, updated")
	listKnowledgeCmd.Flags().StringVar(&knowledgeFilter, "filter", "", "filter knowledge cards by name")

	// Get command flags
	getKnowledgeCardCmd.Flags().StringVar(&knowledgeFormat, "format", "table", "output format: table, json")

	// Create command flags
	createKnowledgeCardCmd.Flags().StringVar(&knowledgeName, "name", "", "knowledge card name")
	createKnowledgeCardCmd.Flags().StringVar(&knowledgeInstructions, "instructions", "", "knowledge card instructions")
	createKnowledgeCardCmd.Flags().BoolVar(&knowledgeInteractive, "interactive", false, "interactive knowledge card creation")
	createKnowledgeCardCmd.Flags().StringVar(&knowledgeFormat, "format", "table", "output format: table, json")

	// Update command flags
	updateKnowledgeCardCmd.Flags().StringVar(&knowledgeName, "name", "", "new knowledge card name")
	updateKnowledgeCardCmd.Flags().StringVar(&knowledgeInstructions, "instructions", "", "new knowledge card instructions")
	updateKnowledgeCardCmd.Flags().StringVar(&knowledgeAppend, "append", "", "append text to existing instructions")
	updateKnowledgeCardCmd.Flags().StringVar(&knowledgeFormat, "format", "table", "output format: table, json")

	// Delete command flags
	deleteKnowledgeCardCmd.Flags().BoolVar(&knowledgeConfirm, "confirm", false, "skip confirmation prompt")

	// Associate command flags
	associateKnowledgeCmd.Flags().StringVar(&knowledgePersona, "persona", "", "persona name or ID")
	associateKnowledgeCmd.Flags().StringVar(&knowledgeName, "knowledge", "", "knowledge card name or ID to associate")
	associateKnowledgeCmd.MarkFlagRequired("persona")
	associateKnowledgeCmd.MarkFlagRequired("knowledge")

	// Disassociate command flags
	disassociateKnowledgeCmd.Flags().StringVar(&knowledgePersona, "persona", "", "persona name or ID")
	disassociateKnowledgeCmd.Flags().StringVar(&knowledgeName, "knowledge", "", "knowledge card name or ID to disassociate")
	disassociateKnowledgeCmd.MarkFlagRequired("persona")
	disassociateKnowledgeCmd.MarkFlagRequired("knowledge")
}

func runListKnowledge(cmd *cobra.Command, args []string) error {
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

	// Get knowledge
	ctx := context.Background()
	knowledge, err := apiClient.Knowledge.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list knowledge: %w", err)
	}

	// Filter knowledge
	if knowledgeFilter != "" {
		knowledge = filterKnowledge(knowledge, knowledgeFilter)
	}

	// Sort knowledge
	sortKnowledge(knowledge, knowledgeSort)

	// Output knowledge
	if knowledgeFormat == "json" {
		return outputKnowledgeJSON(knowledge)
	}

	return outputKnowledgeTable(knowledge)
}

func runGetKnowledgeCard(cmd *cobra.Command, args []string) error {
	knowledgeInput := args[0]

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

	// Validate and get knowledge card by ID or name
	ctx := context.Background()
	knowledgeCard, err := validateKnowledgeCard(ctx, apiClient, knowledgeInput)
	if err != nil {
		return fmt.Errorf("knowledge card validation failed: %w", err)
	}

	// Output knowledge card
	if knowledgeFormat == "json" {
		return outputKnowledgeCardJSON(knowledgeCard)
	}

	return outputKnowledgeCardDetails(knowledgeCard)
}

func runCreateKnowledgeCard(cmd *cobra.Command, args []string) error {
	// Interactive mode
	if knowledgeInteractive {
		return runInteractiveKnowledgeCardCreation()
	}

	// Validate required flags
	if knowledgeName == "" {
		return fmt.Errorf("knowledge card name is required (use --name or --interactive)")
	}
	if knowledgeInstructions == "" {
		return fmt.Errorf("knowledge card instructions are required (use --instructions or --interactive)")
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

	// Create knowledge card
	knowledgeCard := &client.KnowledgeCard{
		Name:         knowledgeName,
		Instructions: knowledgeInstructions,
	}

	ctx := context.Background()
	created, err := apiClient.Knowledge.Create(ctx, knowledgeCard)
	if err != nil {
		return fmt.Errorf("failed to create knowledge card: %w", err)
	}

	fmt.Printf("✓ Knowledge card '%s' created successfully\n", created.Name)
	fmt.Printf("  ID: %s\n", created.KnowledgeCardID)
	fmt.Printf("  Instructions: %s\n", created.Instructions)

	return nil
}

func runUpdateKnowledgeCard(cmd *cobra.Command, args []string) error {
	knowledgeInput := args[0]

	// Check if any update flags are provided
	if knowledgeName == "" && knowledgeInstructions == "" && knowledgeAppend == "" {
		return fmt.Errorf("at least one update flag must be provided (--name, --instructions, or --append)")
	}

	// Validate that --instructions and --append are not used together
	if knowledgeInstructions != "" && knowledgeAppend != "" {
		return fmt.Errorf("--instructions and --append cannot be used together")
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

	// Validate and get existing knowledge card by ID or name
	existing, err := validateKnowledgeCard(ctx, apiClient, knowledgeInput)
	if err != nil {
		return fmt.Errorf("knowledge card validation failed: %w", err)
	}

	// Update fields
	if knowledgeName != "" {
		existing.Name = knowledgeName
	}
	if knowledgeInstructions != "" {
		existing.Instructions = knowledgeInstructions
	}
	if knowledgeAppend != "" {
		existing.Instructions = existing.Instructions + knowledgeAppend
	}

	// Update knowledge card
	updated, err := apiClient.Knowledge.Update(ctx, existing.KnowledgeCardID, existing)
	if err != nil {
		return fmt.Errorf("failed to update knowledge card: %w", err)
	}

	fmt.Printf("✓ Knowledge card updated successfully\n")
	fmt.Printf("  Name: %s\n", updated.Name)
	fmt.Printf("  Instructions: %s\n", updated.Instructions)

	return nil
}

func runDeleteKnowledgeCard(cmd *cobra.Command, args []string) error {
	knowledgeInput := args[0]

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

	// Validate and get knowledge card by ID or name
	knowledgeCard, err := validateKnowledgeCard(ctx, apiClient, knowledgeInput)
	if err != nil {
		return fmt.Errorf("knowledge card validation failed: %w", err)
	}

	// Confirm deletion
	if !knowledgeConfirm {
		fmt.Printf("Are you sure you want to delete knowledge card '%s' (%s)? [y/N]: ", knowledgeCard.Name, knowledgeCard.KnowledgeCardID)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete knowledge card
	err = apiClient.Knowledge.Delete(ctx, knowledgeCard.KnowledgeCardID)
	if err != nil {
		return fmt.Errorf("failed to delete knowledge card: %w", err)
	}

	fmt.Printf("✓ Knowledge card '%s' deleted successfully\n", knowledgeCard.Name)
	return nil
}

func runAssociateKnowledgeCard(cmd *cobra.Command, args []string) error {
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

	// Validate persona
	persona, err := validatePersona(ctx, apiClient, knowledgePersona)
	if err != nil {
		return fmt.Errorf("persona validation failed: %w", err)
	}

	// Validate knowledge card
	knowledgeCard, err := validateKnowledgeCard(ctx, apiClient, knowledgeName)
	if err != nil {
		return fmt.Errorf("knowledge card validation failed: %w", err)
	}

	// Associate knowledge card
	err = apiClient.Knowledge.AssociateWithPersona(ctx, knowledgeCard.KnowledgeCardID, persona.PersonaID)
	if err != nil {
		return fmt.Errorf("failed to associate knowledge card: %w", err)
	}

	fmt.Printf("✓ Knowledge card '%s' associated with persona '%s'\n", knowledgeName, persona.Name)
	return nil
}

func runDisassociateKnowledgeCard(cmd *cobra.Command, args []string) error {
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

	// Validate persona
	persona, err := validatePersona(ctx, apiClient, knowledgePersona)
	if err != nil {
		return fmt.Errorf("persona validation failed: %w", err)
	}

	// Validate knowledge card
	knowledgeCard, err := validateKnowledgeCard(ctx, apiClient, knowledgeName)
	if err != nil {
		return fmt.Errorf("knowledge card validation failed: %w", err)
	}

	// Disassociate knowledge card
	err = apiClient.Knowledge.DisassociateFromPersona(ctx, knowledgeCard.KnowledgeCardID, persona.PersonaID)
	if err != nil {
		return fmt.Errorf("failed to disassociate knowledge card: %w", err)
	}

	fmt.Printf("✓ Knowledge card '%s' disassociated from persona '%s'\n", knowledgeName, persona.Name)
	return nil
}

func runInteractiveKnowledgeCardCreation() error {
	fmt.Println("Interactive Knowledge Card Creation")
	fmt.Println("============================")

	// Get knowledge card name
	fmt.Print("Enter knowledge card name: ")
	var name string
	fmt.Scanln(&name)
	if name == "" {
		return fmt.Errorf("knowledge card name is required")
	}

	// Get knowledge card instructions
	fmt.Println("\nEnter knowledge card instructions (press Enter twice when done):")
	var instructions []string
	for {
		var line string
		fmt.Scanln(&line)
		if line == "" {
			break
		}
		instructions = append(instructions, line)
	}

	if len(instructions) == 0 {
		return fmt.Errorf("knowledge card instructions are required")
	}

	// Set the values for the create function
	knowledgeName = name
	knowledgeInstructions = strings.Join(instructions, "\n")

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

	// Create knowledge card
	knowledgeCard := &client.KnowledgeCard{
		Name:         knowledgeName,
		Instructions: knowledgeInstructions,
	}

	ctx := context.Background()
	created, err := apiClient.Knowledge.Create(ctx, knowledgeCard)
	if err != nil {
		return fmt.Errorf("failed to create knowledge card: %w", err)
	}

	fmt.Printf("\n✓ Knowledge card '%s' created successfully\n", created.Name)
	fmt.Printf("  ID: %s\n", created.KnowledgeCardID)
	fmt.Printf("  Instructions: %s\n", created.Instructions)

	return nil
}

func filterKnowledge(knowledge []client.KnowledgeCard, filter string) []client.KnowledgeCard {
	if filter == "" {
		return knowledge
	}

	var filtered []client.KnowledgeCard
	filter = strings.ToLower(filter)

	for _, knowledgeCard := range knowledge {
		if strings.Contains(strings.ToLower(knowledgeCard.Name), filter) ||
			strings.Contains(strings.ToLower(knowledgeCard.Instructions), filter) {
			filtered = append(filtered, knowledgeCard)
		}
	}

	return filtered
}

func sortKnowledge(knowledge []client.KnowledgeCard, sortBy string) {
	switch sortBy {
	case "name":
		sort.Slice(knowledge, func(i, j int) bool {
			return knowledge[i].Name < knowledge[j].Name
		})
	case "updated":
		sort.Slice(knowledge, func(i, j int) bool {
			return knowledge[i].UpdatedAt.After(knowledge[j].UpdatedAt)
		})
	case "created":
		fallthrough
	default:
		sort.Slice(knowledge, func(i, j int) bool {
			return knowledge[i].CreatedAt.After(knowledge[j].CreatedAt)
		})
	}
}

func outputKnowledgeTable(knowledge []client.KnowledgeCard) error {
	if len(knowledge) == 0 {
		fmt.Println("No knowledge cards found.")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "NAME\tINSTRUCTIONS\tCREATED\tUPDATED\tID")
	fmt.Fprintln(w, "----\t------------\t-------\t-------\t--")

	// Rows
	for _, knowledgeCard := range knowledge {
		created := formatTime(knowledgeCard.CreatedAt)
		updated := formatTime(knowledgeCard.UpdatedAt)

		// Truncate instructions if too long
		instructions := knowledgeCard.Instructions
		if len(instructions) > 50 {
			instructions = instructions[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			knowledgeCard.Name,
			instructions,
			created,
			updated,
			knowledgeCard.KnowledgeCardID,
		)
	}

	return nil
}

func outputKnowledgeJSON(knowledge []client.KnowledgeCard) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"knowledge": knowledge,
		"count":     len(knowledge),
	})
}

func outputKnowledgeCardDetails(knowledgeCard *client.KnowledgeCard) error {
	fmt.Printf("Knowledge Card Details\n")
	fmt.Printf("=====================\n")
	fmt.Printf("Name:         %s\n", knowledgeCard.Name)
	fmt.Printf("ID:           %s\n", knowledgeCard.KnowledgeCardID)
	fmt.Printf("Instructions: %s\n", knowledgeCard.Instructions)
	fmt.Printf("Created:      %s\n", formatTime(knowledgeCard.CreatedAt))
	fmt.Printf("Updated:      %s\n", formatTime(knowledgeCard.UpdatedAt))

	return nil
}

func outputKnowledgeCardJSON(knowledgeCard *client.KnowledgeCard) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(knowledgeCard)
}
