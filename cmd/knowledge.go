package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/internal/config"
	"github.com/toneclone/cli/pkg/client"
)

var lookupKnowledgeSourceHost = func(ctx context.Context, host string) ([]netip.Addr, error) {
	return net.DefaultResolver.LookupNetIP(ctx, "ip", host)
}

var specialUseKnowledgeSourcePrefixes = []netip.Prefix{
	mustKnowledgeSourcePrefix("0.0.0.0/8"),
	mustKnowledgeSourcePrefix("100.64.0.0/10"),
	mustKnowledgeSourcePrefix("127.0.0.0/8"),
	mustKnowledgeSourcePrefix("169.254.0.0/16"),
	mustKnowledgeSourcePrefix("192.0.0.0/24"),
	mustKnowledgeSourcePrefix("192.0.2.0/24"),
	mustKnowledgeSourcePrefix("198.18.0.0/15"),
	mustKnowledgeSourcePrefix("198.51.100.0/24"),
	mustKnowledgeSourcePrefix("203.0.113.0/24"),
	mustKnowledgeSourcePrefix("224.0.0.0/4"),
	mustKnowledgeSourcePrefix("240.0.0.0/4"),
	mustKnowledgeSourcePrefix("::/128"),
	mustKnowledgeSourcePrefix("::1/128"),
	mustKnowledgeSourcePrefix("64:ff9b:1::/48"),
	mustKnowledgeSourcePrefix("100::/64"),
	mustKnowledgeSourcePrefix("2001::/23"),
	mustKnowledgeSourcePrefix("2001:2::/48"),
	mustKnowledgeSourcePrefix("2001:db8::/32"),
	mustKnowledgeSourcePrefix("fc00::/7"),
	mustKnowledgeSourcePrefix("fe80::/10"),
	mustKnowledgeSourcePrefix("ff00::/8"),
}

var (
	// Knowledge command flags
	knowledgeFormat           string
	knowledgeSort             string
	knowledgeFilter           string
	knowledgeInteractive      bool
	knowledgeName             string
	knowledgeInstructions     string
	knowledgeInstructionsHint string
	knowledgeAppend           string
	knowledgeConfirm          bool
	knowledgePersona          string
	knowledgeURL              string
	knowledgeFile             string
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

var createKnowledgeCardFromURLCmd = &cobra.Command{
	Use:   "create-from-url",
	Short: "Create a knowledge card from a public URL",
	Long: `Create a knowledge card by fetching a public HTTP(S) URL, extracting text,
and synthesizing concise editable instructions. The backend rejects private/local
hosts and unsupported or oversized responses.

Examples:
  toneclone knowledge create-from-url --url=https://example.com/docs --json
  toneclone knowledge create-from-url --url=https://example.com/docs --instructions-hint="Focus on pricing and limits"`,
	RunE: runCreateKnowledgeCardFromURL,
}

var createKnowledgeCardFromFileCmd = &cobra.Command{
	Use:   "create-from-file",
	Short: "Create a knowledge card from a local file",
	Long: `Create a knowledge card by uploading a local source file for immediate text
extraction and synthesis. Supported backend formats include text, Markdown, PDF,
DOCX/DOC, HTML, CSV, JSON, and XML, up to 10MB.

Examples:
  toneclone knowledge create-from-file --file=product-faq.md --json
  toneclone knowledge create-from-file --file=pricing.pdf --instructions-hint="Extract durable facts"`,
	RunE: runCreateKnowledgeCardFromFile,
}

var knowledgeSourcesCmd = &cobra.Command{
	Use:   "sources <knowledge-card-name-or-id>",
	Short: "List source metadata for a knowledge card",
	Args:  cobra.ExactArgs(1),
	RunE:  runKnowledgeSources,
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
	knowledgeCmd.AddCommand(createKnowledgeCardFromURLCmd)
	knowledgeCmd.AddCommand(createKnowledgeCardFromFileCmd)
	knowledgeCmd.AddCommand(updateKnowledgeCardCmd)
	knowledgeCmd.AddCommand(deleteKnowledgeCardCmd)
	knowledgeCmd.AddCommand(knowledgeSourcesCmd)
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

	// Source-backed create flags
	createKnowledgeCardFromURLCmd.Flags().StringVar(&knowledgeURL, "url", "", "public HTTP(S) URL to ingest")
	createKnowledgeCardFromURLCmd.Flags().StringVar(&knowledgeInstructionsHint, "instructions-hint", "", "optional synthesis guidance for the source")
	createKnowledgeCardFromURLCmd.Flags().StringVar(&knowledgeFormat, "format", "table", "output format: table, json")
	createKnowledgeCardFromURLCmd.MarkFlagRequired("url")

	createKnowledgeCardFromFileCmd.Flags().StringVar(&knowledgeFile, "file", "", "local file to ingest")
	createKnowledgeCardFromFileCmd.Flags().StringVar(&knowledgeInstructionsHint, "instructions-hint", "", "optional synthesis guidance for the source")
	createKnowledgeCardFromFileCmd.Flags().StringVar(&knowledgeFormat, "format", "table", "output format: table, json")
	createKnowledgeCardFromFileCmd.MarkFlagRequired("file")

	knowledgeSourcesCmd.Flags().StringVar(&knowledgeFormat, "format", "table", "output format: table, json")

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
	if wantsJSONFormat(knowledgeFormat) {
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
	if wantsJSONFormat(knowledgeFormat) {
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

	return outputKnowledgeCardCreated(created)
}

func runCreateKnowledgeCardFromURL(cmd *cobra.Command, args []string) error {
	normalizedURL, err := validateKnowledgeSourceURL(knowledgeURL)
	if err != nil {
		return err
	}
	apiClient, err := newAPIClient(30)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	response, err := apiClient.Knowledge.CreateFromURL(ctx, normalizedURL, knowledgeInstructionsHint)
	if err != nil {
		return err
	}
	return outputKnowledgeIngestResponse(response)
}

func runCreateKnowledgeCardFromFile(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient(30)
	if err != nil {
		return err
	}
	info, err := os.Stat(knowledgeFile)
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", knowledgeFile, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("source file must be a regular file")
	}
	if info.Size() > client.KnowledgeSourceFileMaxBytes {
		return fmt.Errorf("source file is too large: maximum size is 10MB")
	}
	file, err := os.Open(knowledgeFile)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", knowledgeFile, err)
	}
	defer file.Close()
	ctx := cmd.Context()
	response, err := apiClient.Knowledge.CreateFromFile(ctx, knowledgeFile, file, knowledgeInstructionsHint)
	if err != nil {
		return err
	}
	return outputKnowledgeIngestResponse(response)
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

	ctx := cmd.Context()

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

	return outputKnowledgeCardUpdated(updated)
}

func runKnowledgeSources(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient(30)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	knowledgeCard, err := validateKnowledgeCard(ctx, apiClient, args[0])
	if err != nil {
		return fmt.Errorf("knowledge card validation failed: %w", err)
	}
	sources, err := apiClient.Knowledge.Sources(ctx, knowledgeCard.KnowledgeCardID)
	if err != nil {
		return err
	}
	return outputKnowledgeSources(sources)
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
		if wantsJSONFormat(knowledgeFormat) {
			return fmt.Errorf("--confirm is required when deleting with --json")
		}
		fmt.Printf("Are you sure you want to delete knowledge card '%s' (%s)? [y/N]: ", terminalSafe(knowledgeCard.Name), knowledgeCard.KnowledgeCardID)
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

	if wantsJSONFormat(knowledgeFormat) {
		return writeJSON(map[string]interface{}{"deleted": true, "knowledgeCard": knowledgeCard})
	}
	fmt.Printf("✓ Knowledge card '%s' deleted successfully\n", terminalSafe(knowledgeCard.Name))
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

	if wantsJSONFormat(knowledgeFormat) {
		return writeJSON(map[string]interface{}{"associated": true, "knowledgeCard": knowledgeCard, "persona": persona})
	}
	fmt.Printf("✓ Knowledge card '%s' associated with persona '%s'\n", terminalSafe(knowledgeName), terminalSafe(persona.Name))
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

	if wantsJSONFormat(knowledgeFormat) {
		return writeJSON(map[string]interface{}{"disassociated": true, "knowledgeCard": knowledgeCard, "persona": persona})
	}
	fmt.Printf("✓ Knowledge card '%s' disassociated from persona '%s'\n", terminalSafe(knowledgeName), terminalSafe(persona.Name))
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

	return outputKnowledgeCardCreated(created)
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
			terminalSafe(knowledgeCard.Name),
			terminalSafe(instructions),
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
	fmt.Printf("Name:         %s\n", terminalSafe(knowledgeCard.Name))
	fmt.Printf("ID:           %s\n", knowledgeCard.KnowledgeCardID)
	fmt.Printf("Instructions: %s\n", terminalSafe(knowledgeCard.Instructions))
	fmt.Printf("Created:      %s\n", formatTime(knowledgeCard.CreatedAt))
	fmt.Printf("Updated:      %s\n", formatTime(knowledgeCard.UpdatedAt))

	return nil
}

func outputKnowledgeCardJSON(knowledgeCard *client.KnowledgeCard) error {
	return writeJSON(knowledgeCard)
}

func outputKnowledgeCardCreated(knowledgeCard *client.KnowledgeCard) error {
	if wantsJSONFormat(knowledgeFormat) {
		return outputKnowledgeCardJSON(knowledgeCard)
	}
	fmt.Printf("✓ Knowledge card '%s' created successfully\n", terminalSafe(knowledgeCard.Name))
	fmt.Printf("  ID: %s\n", knowledgeCard.KnowledgeCardID)
	fmt.Printf("  Instructions: %s\n", terminalSafe(knowledgeCard.Instructions))
	return nil
}

func outputKnowledgeCardUpdated(knowledgeCard *client.KnowledgeCard) error {
	if wantsJSONFormat(knowledgeFormat) {
		return outputKnowledgeCardJSON(knowledgeCard)
	}
	fmt.Printf("✓ Knowledge card updated successfully\n")
	fmt.Printf("  Name: %s\n", terminalSafe(knowledgeCard.Name))
	fmt.Printf("  Instructions: %s\n", terminalSafe(knowledgeCard.Instructions))
	return nil
}

func outputKnowledgeIngestResponse(response *client.KnowledgeCardIngestResponse) error {
	if wantsJSONFormat(knowledgeFormat) {
		return writeJSON(sanitizeKnowledgeIngestResponse(response))
	}
	fmt.Printf("✓ Knowledge card '%s' created successfully\n", terminalSafe(response.KnowledgeCard.Name))
	fmt.Printf("  ID: %s\n", response.KnowledgeCard.KnowledgeCardID)
	fmt.Printf("  Source: %s", terminalSafe(response.Source.Type))
	if response.Source.URL != "" {
		fmt.Printf(" %s", sanitizeURLForOutput(response.Source.URL))
	} else if response.Source.Filename != "" {
		fmt.Printf(" %s", terminalSafe(response.Source.Filename))
	} else if response.Source.DisplayName != "" {
		fmt.Printf(" %s", terminalSafe(response.Source.DisplayName))
	}
	fmt.Println()
	if response.Source.ExtractedCharCount > 0 {
		fmt.Printf("  Extracted: %d chars\n", response.Source.ExtractedCharCount)
	}
	if response.Synthesis.Summary != "" {
		fmt.Printf("\nSummary:\n%s\n", terminalSafe(response.Synthesis.Summary))
	}
	if len(response.Synthesis.KeyFacts) > 0 {
		fmt.Println("\nKey facts:")
		for _, fact := range response.Synthesis.KeyFacts {
			fmt.Printf("- %s\n", terminalSafe(fact))
		}
	}
	if len(response.Synthesis.UsageNotes) > 0 {
		fmt.Println("\nUsage notes:")
		for _, note := range response.Synthesis.UsageNotes {
			fmt.Printf("- %s\n", terminalSafe(note))
		}
	}
	if len(response.Synthesis.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, warning := range response.Synthesis.Warnings {
			fmt.Printf("- %s\n", terminalSafe(warning))
		}
	}
	return nil
}

func outputKnowledgeSources(sources []client.KnowledgeCardSource) error {
	if wantsJSONFormat(knowledgeFormat) {
		sanitized := sanitizeKnowledgeSources(sources)
		return writeJSON(map[string]interface{}{"sources": sanitized, "count": len(sanitized)})
	}
	if len(sources) == 0 {
		fmt.Println("No source metadata found.")
		return nil
	}
	for _, source := range sources {
		fmt.Printf("Source: %s — %s\n", terminalSafe(source.Type), terminalSafe(source.DisplayName))
		fmt.Printf("Status: %s\n", terminalSafe(source.Status))
		if source.URL != "" {
			fmt.Printf("URL: %s\n", sanitizeURLForOutput(source.URL))
		}
		if source.Filename != "" {
			fmt.Printf("File: %s\n", terminalSafe(source.Filename))
		}
		if source.ExtractedCharCount > 0 {
			fmt.Printf("Extracted: %d chars\n", source.ExtractedCharCount)
		}
		if source.ExtractedTextPreview != "" {
			fmt.Printf("Preview:\n%s\n", terminalSafe(source.ExtractedTextPreview))
		}
		fmt.Println()
	}
	return nil
}

func validateKnowledgeSourceURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed == nil || parsed.Hostname() == "" {
		return "", fmt.Errorf("invalid URL")
	}
	if parsed.User != nil {
		return "", fmt.Errorf("URLs with embedded credentials are not allowed")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("URL must use http or https")
	}
	if parsed.Fragment != "" {
		return "", fmt.Errorf("URL fragments are not allowed")
	}
	for key := range parsed.Query() {
		if isSensitiveURLParam(key) {
			return "", fmt.Errorf("URLs with sensitive query parameters are not allowed")
		}
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return "", fmt.Errorf("localhost URLs are not allowed")
	}
	if err := validateKnowledgeSourceHostPublic(host); err != nil {
		return "", err
	}
	return parsed.String(), nil
}

func validateKnowledgeSourceHostPublic(host string) error {
	if ip, err := netip.ParseAddr(host); err == nil {
		if !isPublicKnowledgeSourceIP(ip) {
			return fmt.Errorf("private or local IP URLs are not allowed")
		}
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	addrs, err := lookupKnowledgeSourceHost(ctx, host)
	if err != nil || len(addrs) == 0 {
		return fmt.Errorf("URL host could not be verified as public")
	}
	for _, ip := range addrs {
		if !isPublicKnowledgeSourceIP(ip) {
			return fmt.Errorf("URL host resolves to a private or local IP")
		}
	}
	return nil
}

func isPublicKnowledgeSourceIP(ip netip.Addr) bool {
	if !ip.IsValid() {
		return false
	}
	ip = ip.Unmap()
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, prefix := range specialUseKnowledgeSourcePrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

func mustKnowledgeSourcePrefix(raw string) netip.Prefix {
	prefix, err := netip.ParsePrefix(raw)
	if err != nil {
		panic(err)
	}
	return prefix
}

func sanitizeURLForOutput(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil {
		return "[redacted-url]"
	}
	parsed.User = nil
	parsed.Fragment = ""
	query := parsed.Query()
	for key := range query {
		if isSensitiveURLParam(key) {
			query.Set(key, "[REDACTED]")
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func isSensitiveURLParam(key string) bool {
	k := strings.ToLower(key)
	sensitive := map[string]bool{
		"access_token":  true,
		"api_key":       true,
		"apikey":        true,
		"auth":          true,
		"authorization": true,
		"credential":    true,
		"key":           true,
		"password":      true,
		"secret":        true,
		"sig":           true,
		"signature":     true,
		"token":         true,
	}
	if sensitive[k] {
		return true
	}
	return strings.HasSuffix(k, "_token") || strings.HasSuffix(k, "-token") || strings.HasSuffix(k, "_secret") || strings.HasSuffix(k, "-secret")
}

func sanitizeKnowledgeIngestResponse(response *client.KnowledgeCardIngestResponse) *client.KnowledgeCardIngestResponse {
	if response == nil {
		return nil
	}
	copy := *response
	copy.Source = sanitizeKnowledgeSource(copy.Source)
	return &copy
}

func sanitizeKnowledgeSources(sources []client.KnowledgeCardSource) []client.KnowledgeCardSource {
	sanitized := make([]client.KnowledgeCardSource, len(sources))
	for i, source := range sources {
		sanitized[i] = sanitizeKnowledgeSource(source)
	}
	return sanitized
}

func sanitizeKnowledgeSource(source client.KnowledgeCardSource) client.KnowledgeCardSource {
	if source.URL != "" {
		source.URL = sanitizeURLForOutput(source.URL)
	}
	return source
}

func terminalSafe(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\t':
			return r
		}
		if r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			return -1
		}
		return r
	}, value)
}

func writeJSON(value interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
