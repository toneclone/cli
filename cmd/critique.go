package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/pkg/client"
)

var (
	critiquePersona         string
	critiqueKnowledge       string
	critiqueText            string
	critiqueFile            string
	critiqueSelection       string
	critiqueSelectionStart  int
	critiqueSelectionEnd    int
	critiqueContext         string
	critiqueCategories      []string
	critiqueIntensity       string
	critiqueMaxSuggestions  int
	critiqueSessionID       string
	critiqueSourceVersionID string
	critiqueTimeout         int
	critiqueHistorySession  string
)

var critiqueCmd = &cobra.Command{
	Use:   "critique",
	Short: "Get editorial feedback on existing writing",
	Long: `Get structured editorial feedback for a draft. Suggestions are typed as
rewrite, directive, or note so agents can decide what to apply or pass to a
rewrite step.`,
}

var critiqueGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Request editorial feedback for a document",
	RunE:  runCritiqueGet,
}

var critiqueHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "List prior editorial feedback passes for a session",
	RunE:  runCritiqueHistory,
}

func init() {
	rootCmd.AddCommand(critiqueCmd)
	critiqueCmd.AddCommand(critiqueGetCmd)
	critiqueCmd.AddCommand(critiqueHistoryCmd)

	critiqueGetCmd.Flags().StringVar(&critiquePersona, "persona", "", "persona ID or name to critique against")
	critiqueGetCmd.Flags().StringVar(&critiqueKnowledge, "knowledge", "", "knowledge card ID or name (comma-separated for multiple)")
	critiqueGetCmd.Flags().StringVar(&critiqueText, "text", "", "document text to critique")
	critiqueGetCmd.Flags().StringVar(&critiqueFile, "file", "", "file containing document text to critique")
	critiqueGetCmd.Flags().StringVar(&critiqueSelection, "selection", "", "selected text/span to focus feedback on")
	critiqueGetCmd.Flags().IntVar(&critiqueSelectionStart, "selection-start", -1, "0-based selection start offset")
	critiqueGetCmd.Flags().IntVar(&critiqueSelectionEnd, "selection-end", -1, "0-based selection end offset")
	critiqueGetCmd.Flags().StringVar(&critiqueContext, "context", "", "extra context for the critique")
	critiqueGetCmd.Flags().StringArrayVar(&critiqueCategories, "category", nil, "feedback category (repeatable: structure, clarity, concision, voice, mechanics, accuracy)")
	critiqueGetCmd.Flags().StringVar(&critiqueIntensity, "intensity", "", "critique intensity, for example light, balanced, or deep")
	critiqueGetCmd.Flags().IntVar(&critiqueMaxSuggestions, "max-suggestions", 0, "maximum number of suggestions")
	critiqueGetCmd.Flags().StringVar(&critiqueSessionID, "session-id", "", "session ID for history persistence")
	critiqueGetCmd.Flags().StringVar(&critiqueSourceVersionID, "source-version-id", "", "source version ID for history persistence")
	critiqueGetCmd.Flags().IntVar(&critiqueTimeout, "timeout", 30, "request timeout in seconds")
	critiqueGetCmd.MarkFlagRequired("persona")

	critiqueHistoryCmd.Flags().StringVar(&critiqueHistorySession, "session-id", "", "session ID to list critique history for")
	critiqueHistoryCmd.Flags().IntVar(&critiqueTimeout, "timeout", 30, "request timeout in seconds")
	critiqueHistoryCmd.MarkFlagRequired("session-id")
}

func runCritiqueGet(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient(critiqueTimeout)
	if err != nil {
		return err
	}
	document, err := sourceText(critiqueText, critiqueFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(document) == "" {
		return fmt.Errorf("no document to critique: provide --text, --file, or stdin")
	}
	persona, err := validatePersona(cmd.Context(), apiClient, critiquePersona)
	if err != nil {
		return fmt.Errorf("persona validation failed: %w", err)
	}
	knowledgeCardID, knowledgeCardIDs, err := resolveKnowledgeCards(cmd.Context(), apiClient, critiqueKnowledge)
	if err != nil {
		return err
	}
	if knowledgeCardID != "" {
		knowledgeCardIDs = []string{knowledgeCardID}
	}

	request := &client.CritiqueRequest{
		PersonaID:        persona.PersonaID,
		KnowledgeCardIDs: knowledgeCardIDs,
		Document:         document,
		Selection:        critiqueSelection,
		Context:          critiqueContext,
		Categories:       critiqueCategories,
		Intensity:        critiqueIntensity,
		MaxSuggestions:   critiqueMaxSuggestions,
		SessionID:        critiqueSessionID,
		SourceVersionID:  critiqueSourceVersionID,
	}
	if critiqueSelectionStart >= 0 {
		request.SelectionStart = &critiqueSelectionStart
	}
	if critiqueSelectionEnd >= 0 {
		request.SelectionEnd = &critiqueSelectionEnd
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(critiqueTimeout)*time.Second)
	defer cancel()
	plan, err := apiClient.Critique.Get(ctx, request)
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(plan)
	}
	return outputCritiqueText(plan)
}

func runCritiqueHistory(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient(critiqueTimeout)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(critiqueTimeout)*time.Second)
	defer cancel()
	history, err := apiClient.Critique.History(ctx, critiqueHistorySession)
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(history)
	}
	for _, item := range history.Items {
		fmt.Printf("%s  %s  %d suggestions\n", item.CreatedAt, item.CritiqueID, len(item.Suggestions))
		if item.StrategyNote != "" {
			fmt.Printf("  %s\n", item.StrategyNote)
		}
	}
	return nil
}

func outputCritiqueText(plan *client.CritiquePlan) error {
	if plan.StrategyNote != "" {
		fmt.Printf("Strategy: %s\n\n", plan.StrategyNote)
	}
	groups := map[string][]client.CritiqueSuggestion{}
	for _, suggestion := range plan.Suggestions {
		key := suggestion.Type
		if key == "" {
			key = "suggestion"
		}
		groups[key] = append(groups[key], suggestion)
	}
	var keys []string
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("%s\n", strings.ToUpper(key))
		for _, suggestion := range groups[key] {
			fmt.Printf("- [%s p%d] %s\n", suggestion.Category, suggestion.Priority, suggestion.Title)
			if suggestion.Rationale != "" {
				fmt.Printf("  Why: %s\n", suggestion.Rationale)
			}
			if suggestion.Anchor != nil && suggestion.Anchor.Quote != "" {
				fmt.Printf("  Anchor: %q\n", suggestion.Anchor.Quote)
			}
			if suggestion.Replacement != "" {
				fmt.Printf("  Replacement: %s\n", suggestion.Replacement)
			}
			if suggestion.Instruction != "" {
				fmt.Printf("  Instruction: %s\n", suggestion.Instruction)
			}
		}
		fmt.Println()
	}
	return nil
}
