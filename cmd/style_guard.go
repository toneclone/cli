package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/pkg/client"
)

var (
	// StyleGuard command flags
	sgFormat      string
	sgGlobal      bool
	sgWord        string
	sgMode        string
	sgReplacement string
	sgConfirm     bool
	sgBundleType  string
	sgPersona     string
)

// styleGuardCmd represents the style-guard command group
var styleGuardCmd = &cobra.Command{
	Use:   "style-guard",
	Short: "Manage style guard rules",
	Long: `Manage style guard rules that flag and replace words or patterns in generated output.

Rules can operate in two modes:
  AI     - AI rewrites the flagged word/phrase contextually
  CUSTOM - Replace with a fixed string

Rules can be global or scoped to a specific persona.

Examples:
  toneclone personas style-guard list my-persona
  toneclone personas style-guard add my-persona --word "utilize" --mode AI
  toneclone personas style-guard bundle preview --type comprehensive`,
}

// sgListCmd lists style guard rules
var sgListCmd = &cobra.Command{
	Use:   "list [persona]",
	Short: "List style guard rules",
	Long: `List style guard rules for a persona or global rules.

Examples:
  toneclone personas style-guard list my-persona
  toneclone personas style-guard list --global
  toneclone personas style-guard list my-persona --format json`,
	RunE: runSGList,
}

// sgAddCmd adds a style guard rule
var sgAddCmd = &cobra.Command{
	Use:   "add [persona]",
	Short: "Add a style guard rule",
	Long: `Add a new style guard rule to a persona or globally.

Examples:
  toneclone personas style-guard add my-persona --word "utilize" --mode AI
  toneclone personas style-guard add my-persona --word "in order to" --mode CUSTOM --replacement "to"
  toneclone personas style-guard add --global --word "utilize" --mode AI`,
	RunE: runSGAdd,
}

// sgUpdateCmd updates a style guard rule
var sgUpdateCmd = &cobra.Command{
	Use:   "update <rule-id>",
	Short: "Update a style guard rule",
	Long: `Update an existing style guard rule.

Examples:
  toneclone personas style-guard update sg-abc123 --mode CUSTOM --replacement "use"
  toneclone personas style-guard update sg-abc123 --word "leverage"`,
	Args: cobra.ExactArgs(1),
	RunE: runSGUpdate,
}

// sgDeleteCmd deletes a style guard rule
var sgDeleteCmd = &cobra.Command{
	Use:   "delete <rule-id>",
	Short: "Delete a style guard rule",
	Long: `Delete a style guard rule.

Examples:
  toneclone personas style-guard delete sg-abc123
  toneclone personas style-guard delete sg-abc123 --confirm`,
	Args: cobra.ExactArgs(1),
	RunE: runSGDelete,
}

// bundleCmd represents the bundle subcommand group
var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Manage style guard bundles",
	Long: `Manage curated bundles of style guard rules.

Bundle types:
  limited       - Essential phrase and punctuation replacements (7 items)
  comprehensive - Superset of limited plus common AI phrases (~26 items)

Examples:
  toneclone personas style-guard bundle preview
  toneclone personas style-guard bundle apply --persona my-persona
  toneclone personas style-guard bundle status --persona my-persona`,
}

// bundlePreviewCmd previews a bundle
var bundlePreviewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview bundle items",
	Long: `Preview the items in a style guard bundle before applying.

Examples:
  toneclone personas style-guard bundle preview
  toneclone personas style-guard bundle preview --type limited
  toneclone personas style-guard bundle preview --format json`,
	RunE: runBundlePreview,
}

// bundleStatusCmd shows bundle status
var bundleStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show bundle application status",
	Long: `Show whether a bundle has been applied and its current status.

Examples:
  toneclone personas style-guard bundle status --persona my-persona
  toneclone personas style-guard bundle status --persona my-persona --type limited`,
	RunE: runBundleStatus,
}

// bundleApplyCmd applies a bundle
var bundleApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply a style guard bundle",
	Long: `Apply a curated bundle of style guard rules.

Examples:
  toneclone personas style-guard bundle apply --persona my-persona
  toneclone personas style-guard bundle apply --persona my-persona --type limited`,
	RunE: runBundleApply,
}

// bundleRemoveCmd removes bundle rules
var bundleRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove bundle-sourced rules",
	Long: `Remove all style guard rules that were added from a bundle (source=BUNDLE only).
User-created and user-modified rules are preserved.

Examples:
  toneclone personas style-guard bundle remove --persona my-persona`,
	RunE: runBundleRemove,
}

func init() {
	personasCmd.AddCommand(styleGuardCmd)

	// Style guard subcommands
	styleGuardCmd.AddCommand(sgListCmd)
	styleGuardCmd.AddCommand(sgAddCmd)
	styleGuardCmd.AddCommand(sgUpdateCmd)
	styleGuardCmd.AddCommand(sgDeleteCmd)
	styleGuardCmd.AddCommand(bundleCmd)

	// Bundle subcommands
	bundleCmd.AddCommand(bundlePreviewCmd)
	bundleCmd.AddCommand(bundleStatusCmd)
	bundleCmd.AddCommand(bundleApplyCmd)
	bundleCmd.AddCommand(bundleRemoveCmd)

	// List flags
	sgListCmd.Flags().StringVar(&sgFormat, "format", "table", "output format: table, json")
	sgListCmd.Flags().BoolVar(&sgGlobal, "global", false, "list global rules instead of persona rules")

	// Add flags
	sgAddCmd.Flags().StringVar(&sgFormat, "format", "table", "output format: table, json")
	sgAddCmd.Flags().BoolVar(&sgGlobal, "global", false, "add as a global rule")
	sgAddCmd.Flags().StringVar(&sgWord, "word", "", "word or phrase to flag")
	sgAddCmd.Flags().StringVar(&sgMode, "mode", "", "replacement mode: AI or CUSTOM")
	sgAddCmd.Flags().StringVar(&sgReplacement, "replacement", "", "custom replacement text (required for CUSTOM mode)")

	// Update flags
	sgUpdateCmd.Flags().StringVar(&sgFormat, "format", "table", "output format: table, json")
	sgUpdateCmd.Flags().StringVar(&sgWord, "word", "", "new word or phrase")
	sgUpdateCmd.Flags().StringVar(&sgMode, "mode", "", "new replacement mode: AI or CUSTOM")
	sgUpdateCmd.Flags().StringVar(&sgReplacement, "replacement", "", "new custom replacement text")

	// Delete flags
	sgDeleteCmd.Flags().BoolVar(&sgConfirm, "confirm", false, "skip confirmation prompt")

	// Bundle preview flags
	bundlePreviewCmd.Flags().StringVar(&sgFormat, "format", "table", "output format: table, json")
	bundlePreviewCmd.Flags().StringVar(&sgBundleType, "type", "comprehensive", "bundle type: limited, comprehensive")

	// Bundle status flags
	bundleStatusCmd.Flags().StringVar(&sgFormat, "format", "table", "output format: table, json")
	bundleStatusCmd.Flags().StringVar(&sgPersona, "persona", "", "persona ID or name")
	bundleStatusCmd.Flags().StringVar(&sgBundleType, "type", "comprehensive", "bundle type: limited, comprehensive")

	// Bundle apply flags
	bundleApplyCmd.Flags().StringVar(&sgPersona, "persona", "", "persona ID or name")
	bundleApplyCmd.Flags().StringVar(&sgBundleType, "type", "comprehensive", "bundle type: limited, comprehensive")

	// Bundle remove flags
	bundleRemoveCmd.Flags().StringVar(&sgPersona, "persona", "", "persona ID or name")
}

func runSGList(cmd *cobra.Command, args []string) error {
	if !sgGlobal && len(args) == 0 {
		return fmt.Errorf("persona argument required (or use --global for global rules)")
	}

	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	var words []client.StyleGuardWord
	if sgGlobal {
		words, err = apiClient.StyleGuard.List(ctx)
	} else {
		persona, perr := validatePersona(ctx, apiClient, args[0])
		if perr != nil {
			return fmt.Errorf("failed to resolve persona: %w", perr)
		}
		words, err = apiClient.StyleGuard.ListForPersona(ctx, persona.PersonaID)
	}
	if err != nil {
		return err
	}

	if sgFormat == "json" {
		return outputSGJSON(words)
	}
	return outputSGTable(words)
}

func runSGAdd(cmd *cobra.Command, args []string) error {
	if !sgGlobal && len(args) == 0 {
		return fmt.Errorf("persona argument required (or use --global)")
	}
	if sgWord == "" {
		return fmt.Errorf("--word is required")
	}
	mode := strings.ToUpper(sgMode)
	if mode != "AI" && mode != "CUSTOM" {
		return fmt.Errorf("--mode must be AI or CUSTOM")
	}
	if mode == "CUSTOM" && sgReplacement == "" {
		return fmt.Errorf("--replacement is required for CUSTOM mode")
	}

	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	req := &client.CreateStyleGuardWordRequest{
		Word:              sgWord,
		Mode:              mode,
		CustomReplacement: sgReplacement,
	}

	if !sgGlobal {
		persona, perr := validatePersona(ctx, apiClient, args[0])
		if perr != nil {
			return fmt.Errorf("failed to resolve persona: %w", perr)
		}
		req.PersonaID = persona.PersonaID
	}

	result, err := apiClient.StyleGuard.Create(ctx, req)
	if err != nil {
		return err
	}

	if sgFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	fmt.Printf("Created style guard rule:\n")
	fmt.Printf("  ID:          %s\n", result.StyleGuardID)
	fmt.Printf("  Word:        %s\n", result.Word)
	fmt.Printf("  Mode:        %s\n", result.Mode)
	if result.CustomReplacement != "" {
		fmt.Printf("  Replacement: %s\n", result.CustomReplacement)
	}
	return nil
}

func runSGUpdate(cmd *cobra.Command, args []string) error {
	ruleID := args[0]

	if sgWord == "" && sgMode == "" && sgReplacement == "" {
		return fmt.Errorf("at least one update flag must be provided (--word, --mode, --replacement)")
	}

	mode := strings.ToUpper(sgMode)
	if sgMode != "" {
		if mode != "AI" && mode != "CUSTOM" {
			return fmt.Errorf("--mode must be AI or CUSTOM")
		}
	}

	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	req := &client.UpdateStyleGuardWordRequest{
		Word:              sgWord,
		Mode:              mode,
		CustomReplacement: sgReplacement,
	}

	result, err := apiClient.StyleGuard.Update(ctx, ruleID, req)
	if err != nil {
		return err
	}

	if sgFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	fmt.Printf("Updated style guard rule:\n")
	fmt.Printf("  ID:          %s\n", result.StyleGuardID)
	fmt.Printf("  Word:        %s\n", result.Word)
	fmt.Printf("  Mode:        %s\n", result.Mode)
	if result.CustomReplacement != "" {
		fmt.Printf("  Replacement: %s\n", result.CustomReplacement)
	}
	return nil
}

func runSGDelete(cmd *cobra.Command, args []string) error {
	ruleID := args[0]

	if !sgConfirm {
		fmt.Printf("Are you sure you want to delete style guard rule '%s'? [y/N]: ", ruleID)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}
	}

	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	err = apiClient.StyleGuard.Delete(ctx, ruleID)
	if err != nil {
		return err
	}

	fmt.Printf("Deleted style guard rule '%s'\n", ruleID)
	return nil
}

func runBundlePreview(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()
	items, err := apiClient.StyleGuard.BundlePreview(ctx, sgBundleType)
	if err != nil {
		return err
	}

	if sgFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(map[string]interface{}{
			"bundleType": sgBundleType,
			"items":      items,
			"count":      len(items),
		})
	}

	if len(items) == 0 {
		fmt.Println("No items in bundle.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "WORD\tMODE\tREPLACEMENT\tCATEGORY\n")
	fmt.Fprintf(w, "----\t----\t-----------\t--------\n")
	for _, item := range items {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			item.Word,
			item.Mode,
			item.CustomReplacement,
			item.Category,
		)
	}

	return nil
}

func runBundleStatus(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	personaID := sgPersona
	if personaID != "" {
		persona, perr := validatePersona(ctx, apiClient, personaID)
		if perr != nil {
			return fmt.Errorf("failed to resolve persona: %w", perr)
		}
		personaID = persona.PersonaID
	}

	status, err := apiClient.StyleGuard.BundleStatus(ctx, personaID, sgBundleType)
	if err != nil {
		return err
	}

	if sgFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(status)
	}

	fmt.Printf("Bundle Status\n")
	fmt.Printf("=============\n")
	fmt.Printf("Applied:        %t\n", status.Applied)
	fmt.Printf("Bundle Type:    %s\n", status.BundleType)
	fmt.Printf("Bundle Version: %s\n", status.BundleVersion)
	fmt.Printf("Word Count:     %d\n", status.WordCount)

	return nil
}

func runBundleApply(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	req := &client.ApplyBundleRequest{
		BundleType: sgBundleType,
	}

	if sgPersona != "" {
		persona, perr := validatePersona(ctx, apiClient, sgPersona)
		if perr != nil {
			return fmt.Errorf("failed to resolve persona: %w", perr)
		}
		req.PersonaID = persona.PersonaID
	}

	err = apiClient.StyleGuard.BundleApply(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("Bundle '%s' applied successfully\n", sgBundleType)
	return nil
}

func runBundleRemove(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient()
	if err != nil {
		return err
	}

	ctx := context.Background()

	personaID := ""
	if sgPersona != "" {
		persona, perr := validatePersona(ctx, apiClient, sgPersona)
		if perr != nil {
			return fmt.Errorf("failed to resolve persona: %w", perr)
		}
		personaID = persona.PersonaID
	}

	err = apiClient.StyleGuard.BundleRemove(ctx, personaID)
	if err != nil {
		return err
	}

	fmt.Println("Bundle rules removed successfully")
	return nil
}

func outputSGTable(words []client.StyleGuardWord) error {
	if len(words) == 0 {
		fmt.Println("No style guard rules found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "WORD\tMODE\tREPLACEMENT\tSOURCE\tID\n")
	fmt.Fprintf(w, "----\t----\t-----------\t------\t--\n")
	for _, word := range words {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			word.Word,
			word.Mode,
			word.CustomReplacement,
			word.Source,
			word.StyleGuardID,
		)
	}

	return nil
}

func outputSGJSON(words []client.StyleGuardWord) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]interface{}{
		"rules": words,
		"count": len(words),
	})
}
