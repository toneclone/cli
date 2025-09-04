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
	// Profile command flags
	profileFormat       string
	profileSort         string
	profileFilter       string
	profileInteractive  bool
	profileName         string
	profileInstructions string
	profileAppend       string
	profileConfirm      bool
	profilePersona      string
)

// profilesCmd represents the profiles command
var profilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "Manage ToneClone profiles",
	Long: `Manage ToneClone profiles - create, list, update, and delete writing profiles.

Profiles define writing instructions and context that can be used with personas
to customize the writing style and format for specific use cases.

Examples:
  toneclone profiles list
  toneclone profiles list --filter="email"
  toneclone profiles get "Email Template"
  toneclone profiles create --name="Email" --instructions="Write professional emails"
  toneclone profiles update "Email Template" --name="New Name"
  toneclone profiles delete "Email Template"
  toneclone profiles associate --profile="Email Template" --persona=Professional`,
}

// listProfilesCmd represents the list subcommand
var listProfilesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long: `List all profiles associated with your account.

The list can be filtered by name and sorted by various criteria.
By default, profiles are sorted by creation date (most recent first).

Examples:
  toneclone profiles list
  toneclone profiles list --filter="email"
  toneclone profiles list --sort="name"
  toneclone profiles list --format="json"`,
	RunE: runListProfiles,
}

// getProfileCmd represents the get subcommand
var getProfileCmd = &cobra.Command{
	Use:   "get <profile-name-or-id>",
	Short: "Get detailed information about a profile",
	Long: `Get detailed information about a specific profile by name or ID.

Shows all metadata including instructions, creation date, and usage information.

Examples:
  toneclone profiles get "Email Template"
  toneclone profiles get profile-id
  toneclone profiles get "Email Template" --format="json"`,
	Args: cobra.ExactArgs(1),
	RunE: runGetProfile,
}

// createProfileCmd represents the create subcommand
var createProfileCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new profile",
	Long: `Create a new profile with the specified name and instructions.

Profiles define writing instructions and context that can be used with personas
to customize the writing style and format for specific use cases.

Examples:
  toneclone profiles create --name="Email" --instructions="Write professional emails"
  toneclone profiles create --name="Blog Post" --instructions="Write engaging blog posts"
  toneclone profiles create --interactive`,
	RunE: runCreateProfile,
}

// updateProfileCmd represents the update subcommand
var updateProfileCmd = &cobra.Command{
	Use:   "update <profile-name-or-id>",
	Short: "Update an existing profile",
	Long: `Update the properties of an existing profile by name or ID.

You can update the name and instructions of a profile, or append text to existing instructions.

Examples:
  toneclone profiles update "Email Template" --name="New Name"
  toneclone profiles update profile-id --instructions="New instructions"
  toneclone profiles update "Email Template" --append=" Also include examples."`,
	Args: cobra.ExactArgs(1),
	RunE: runUpdateProfile,
}

// deleteProfileCmd represents the delete subcommand
var deleteProfileCmd = &cobra.Command{
	Use:   "delete <profile-name-or-id>",
	Short: "Delete a profile",
	Long: `Delete a profile permanently by name or ID.

This action cannot be undone. The profile will be disassociated from all personas.

Examples:
  toneclone profiles delete "Email Template"
  toneclone profiles delete profile-id
  toneclone profiles delete "Email Template" --confirm`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteProfile,
}

// associateProfileCmd represents the associate subcommand
var associateProfileCmd = &cobra.Command{
	Use:   "associate",
	Short: "Associate a profile with a persona",
	Long: `Associate a profile with a persona for use in text generation.

The profile will be available when generating text with the specified persona.
Both profile and persona can be specified by name or ID.

Examples:
  toneclone profiles associate --profile="Email Template" --persona=Professional
  toneclone profiles associate --profile=profile-id --persona=persona-id`,
	RunE: runAssociateProfile,
}

// disassociateProfileCmd represents the disassociate subcommand
var disassociateProfileCmd = &cobra.Command{
	Use:   "disassociate",
	Short: "Disassociate a profile from a persona",
	Long: `Disassociate a profile from a persona.

The profile will no longer be available when generating text with the specified persona.
Both profile and persona can be specified by name or ID.

Examples:
  toneclone profiles disassociate --profile="Email Template" --persona=Professional
  toneclone profiles disassociate --profile=profile-id --persona=persona-id`,
	RunE: runDisassociateProfile,
}

func init() {
	rootCmd.AddCommand(profilesCmd)

	// Add subcommands
	profilesCmd.AddCommand(listProfilesCmd)
	profilesCmd.AddCommand(getProfileCmd)
	profilesCmd.AddCommand(createProfileCmd)
	profilesCmd.AddCommand(updateProfileCmd)
	profilesCmd.AddCommand(deleteProfileCmd)
	profilesCmd.AddCommand(associateProfileCmd)
	profilesCmd.AddCommand(disassociateProfileCmd)

	// List command flags
	listProfilesCmd.Flags().StringVar(&profileFormat, "format", "table", "output format: table, json")
	listProfilesCmd.Flags().StringVar(&profileSort, "sort", "created", "sort by: name, created, updated")
	listProfilesCmd.Flags().StringVar(&profileFilter, "filter", "", "filter profiles by name")

	// Get command flags
	getProfileCmd.Flags().StringVar(&profileFormat, "format", "table", "output format: table, json")

	// Create command flags
	createProfileCmd.Flags().StringVar(&profileName, "name", "", "profile name")
	createProfileCmd.Flags().StringVar(&profileInstructions, "instructions", "", "profile instructions")
	createProfileCmd.Flags().BoolVar(&profileInteractive, "interactive", false, "interactive profile creation")
	createProfileCmd.Flags().StringVar(&profileFormat, "format", "table", "output format: table, json")

	// Update command flags
	updateProfileCmd.Flags().StringVar(&profileName, "name", "", "new profile name")
	updateProfileCmd.Flags().StringVar(&profileInstructions, "instructions", "", "new profile instructions")
	updateProfileCmd.Flags().StringVar(&profileAppend, "append", "", "append text to existing instructions")
	updateProfileCmd.Flags().StringVar(&profileFormat, "format", "table", "output format: table, json")

	// Delete command flags
	deleteProfileCmd.Flags().BoolVar(&profileConfirm, "confirm", false, "skip confirmation prompt")

	// Associate command flags
	associateProfileCmd.Flags().StringVar(&profilePersona, "persona", "", "persona name or ID")
	associateProfileCmd.Flags().StringVar(&profileName, "profile", "", "profile name or ID to associate")
	associateProfileCmd.MarkFlagRequired("persona")
	associateProfileCmd.MarkFlagRequired("profile")

	// Disassociate command flags
	disassociateProfileCmd.Flags().StringVar(&profilePersona, "persona", "", "persona name or ID")
	disassociateProfileCmd.Flags().StringVar(&profileName, "profile", "", "profile name or ID to disassociate")
	disassociateProfileCmd.MarkFlagRequired("persona")
	disassociateProfileCmd.MarkFlagRequired("profile")
}

func runListProfiles(cmd *cobra.Command, args []string) error {
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

	// Get profiles
	ctx := context.Background()
	profiles, err := apiClient.Profiles.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list profiles: %w", err)
	}

	// Filter profiles
	if profileFilter != "" {
		profiles = filterProfiles(profiles, profileFilter)
	}

	// Sort profiles
	sortProfiles(profiles, profileSort)

	// Output profiles
	if profileFormat == "json" {
		return outputProfilesJSON(profiles)
	}

	return outputProfilesTable(profiles)
}

func runGetProfile(cmd *cobra.Command, args []string) error {
	profileInput := args[0]

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

	// Validate and get profile by ID or name
	ctx := context.Background()
	profile, err := validateProfile(ctx, apiClient, profileInput)
	if err != nil {
		return fmt.Errorf("profile validation failed: %w", err)
	}

	// Output profile
	if profileFormat == "json" {
		return outputProfileJSON(profile)
	}

	return outputProfileDetails(profile)
}

func runCreateProfile(cmd *cobra.Command, args []string) error {
	// Interactive mode
	if profileInteractive {
		return runInteractiveProfileCreation()
	}

	// Validate required flags
	if profileName == "" {
		return fmt.Errorf("profile name is required (use --name or --interactive)")
	}
	if profileInstructions == "" {
		return fmt.Errorf("profile instructions are required (use --instructions or --interactive)")
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

	// Create profile
	profile := &client.Profile{
		Name:         profileName,
		Instructions: profileInstructions,
	}

	ctx := context.Background()
	created, err := apiClient.Profiles.Create(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	fmt.Printf("✓ Profile '%s' created successfully\n", created.Name)
	fmt.Printf("  ID: %s\n", created.ProfileID)
	fmt.Printf("  Instructions: %s\n", created.Instructions)

	return nil
}

func runUpdateProfile(cmd *cobra.Command, args []string) error {
	profileInput := args[0]

	// Check if any update flags are provided
	if profileName == "" && profileInstructions == "" && profileAppend == "" {
		return fmt.Errorf("at least one update flag must be provided (--name, --instructions, or --append)")
	}

	// Validate that --instructions and --append are not used together
	if profileInstructions != "" && profileAppend != "" {
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

	// Validate and get existing profile by ID or name
	existing, err := validateProfile(ctx, apiClient, profileInput)
	if err != nil {
		return fmt.Errorf("profile validation failed: %w", err)
	}

	// Update fields
	if profileName != "" {
		existing.Name = profileName
	}
	if profileInstructions != "" {
		existing.Instructions = profileInstructions
	}
	if profileAppend != "" {
		existing.Instructions = existing.Instructions + profileAppend
	}

	// Update profile
	updated, err := apiClient.Profiles.Update(ctx, existing.ProfileID, existing)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	fmt.Printf("✓ Profile updated successfully\n")
	fmt.Printf("  Name: %s\n", updated.Name)
	fmt.Printf("  Instructions: %s\n", updated.Instructions)

	return nil
}

func runDeleteProfile(cmd *cobra.Command, args []string) error {
	profileInput := args[0]

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

	// Validate and get profile by ID or name
	profile, err := validateProfile(ctx, apiClient, profileInput)
	if err != nil {
		return fmt.Errorf("profile validation failed: %w", err)
	}

	// Confirm deletion
	if !profileConfirm {
		fmt.Printf("Are you sure you want to delete profile '%s' (%s)? [y/N]: ", profile.Name, profile.ProfileID)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	// Delete profile
	err = apiClient.Profiles.Delete(ctx, profile.ProfileID)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	fmt.Printf("✓ Profile '%s' deleted successfully\n", profile.Name)
	return nil
}

func runAssociateProfile(cmd *cobra.Command, args []string) error {
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
	persona, err := validatePersona(ctx, apiClient, profilePersona)
	if err != nil {
		return fmt.Errorf("persona validation failed: %w", err)
	}

	// Validate profile
	profile, err := validateProfile(ctx, apiClient, profileName)
	if err != nil {
		return fmt.Errorf("profile validation failed: %w", err)
	}

	// Associate profile
	err = apiClient.Profiles.AssociateWithPersona(ctx, profile.ProfileID, persona.PersonaID)
	if err != nil {
		return fmt.Errorf("failed to associate profile: %w", err)
	}

	fmt.Printf("✓ Profile '%s' associated with persona '%s'\n", profileName, persona.Name)
	return nil
}

func runDisassociateProfile(cmd *cobra.Command, args []string) error {
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
	persona, err := validatePersona(ctx, apiClient, profilePersona)
	if err != nil {
		return fmt.Errorf("persona validation failed: %w", err)
	}

	// Validate profile
	profile, err := validateProfile(ctx, apiClient, profileName)
	if err != nil {
		return fmt.Errorf("profile validation failed: %w", err)
	}

	// Disassociate profile
	err = apiClient.Profiles.DisassociateFromPersona(ctx, profile.ProfileID, persona.PersonaID)
	if err != nil {
		return fmt.Errorf("failed to disassociate profile: %w", err)
	}

	fmt.Printf("✓ Profile '%s' disassociated from persona '%s'\n", profileName, persona.Name)
	return nil
}

func runInteractiveProfileCreation() error {
	fmt.Println("Interactive Profile Creation")
	fmt.Println("============================")

	// Get profile name
	fmt.Print("Enter profile name: ")
	var name string
	fmt.Scanln(&name)
	if name == "" {
		return fmt.Errorf("profile name is required")
	}

	// Get profile instructions
	fmt.Println("\nEnter profile instructions (press Enter twice when done):")
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
		return fmt.Errorf("profile instructions are required")
	}

	// Set the values for the create function
	profileName = name
	profileInstructions = strings.Join(instructions, "\n")

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

	// Create profile
	profile := &client.Profile{
		Name:         profileName,
		Instructions: profileInstructions,
	}

	ctx := context.Background()
	created, err := apiClient.Profiles.Create(ctx, profile)
	if err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	fmt.Printf("\n✓ Profile '%s' created successfully\n", created.Name)
	fmt.Printf("  ID: %s\n", created.ProfileID)
	fmt.Printf("  Instructions: %s\n", created.Instructions)

	return nil
}

func filterProfiles(profiles []client.Profile, filter string) []client.Profile {
	if filter == "" {
		return profiles
	}

	var filtered []client.Profile
	filter = strings.ToLower(filter)

	for _, profile := range profiles {
		if strings.Contains(strings.ToLower(profile.Name), filter) ||
			strings.Contains(strings.ToLower(profile.Instructions), filter) {
			filtered = append(filtered, profile)
		}
	}

	return filtered
}

func sortProfiles(profiles []client.Profile, sortBy string) {
	switch sortBy {
	case "name":
		sort.Slice(profiles, func(i, j int) bool {
			return profiles[i].Name < profiles[j].Name
		})
	case "updated":
		sort.Slice(profiles, func(i, j int) bool {
			return profiles[i].UpdatedAt.After(profiles[j].UpdatedAt)
		})
	case "created":
		fallthrough
	default:
		sort.Slice(profiles, func(i, j int) bool {
			return profiles[i].CreatedAt.After(profiles[j].CreatedAt)
		})
	}
}

func outputProfilesTable(profiles []client.Profile) error {
	if len(profiles) == 0 {
		fmt.Println("No profiles found.")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	// Header
	fmt.Fprintln(w, "NAME\tINSTRUCTIONS\tCREATED\tUPDATED\tID")
	fmt.Fprintln(w, "----\t------------\t-------\t-------\t--")

	// Rows
	for _, profile := range profiles {
		created := formatTime(profile.CreatedAt)
		updated := formatTime(profile.UpdatedAt)

		// Truncate instructions if too long
		instructions := profile.Instructions
		if len(instructions) > 50 {
			instructions = instructions[:47] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			profile.Name,
			instructions,
			created,
			updated,
			profile.ProfileID,
		)
	}

	return nil
}

func outputProfilesJSON(profiles []client.Profile) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"profiles": profiles,
		"count":    len(profiles),
	})
}

func outputProfileDetails(profile *client.Profile) error {
	fmt.Printf("Profile Details\n")
	fmt.Printf("===============\n")
	fmt.Printf("Name:         %s\n", profile.Name)
	fmt.Printf("ID:           %s\n", profile.ProfileID)
	fmt.Printf("Instructions: %s\n", profile.Instructions)
	fmt.Printf("Created:      %s\n", formatTime(profile.CreatedAt))
	fmt.Printf("Updated:      %s\n", formatTime(profile.UpdatedAt))

	return nil
}

func outputProfileJSON(profile *client.Profile) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(profile)
}
