package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/pkg/client"
)

// Task-verb commands provide an agent-friendly surface over the same generation
// backend as `write`. personalize rewrites existing text in a persona's voice;
// humanize runs a StyleGuard-only cleanup pass (no persona, no quota).

var (
	personalizeText       string
	personalizeFile       string
	personalizePersona    string
	personalizeKnowledge  string
	personalizeTimeout    int
	personalizeReviewLink bool

	humanizeText       string
	humanizeFile       string
	humanizePersona    string
	humanizeTimeout    int
	humanizeReviewLink bool
)

var personalizeCmd = &cobra.Command{
	Use:   "personalize",
	Short: "Rewrite existing text in a persona's voice",
	Long: `Rewrite existing text so it sounds like the selected persona, preserving the
meaning. Provide the source text via --text, --file, or stdin.

Examples:
  toneclone personalize --persona="Founder" --text="We are pleased to announce..."
  cat draft.md | toneclone personalize --persona="Founder" --json`,
	RunE: runPersonalize,
}

var humanizeCmd = &cobra.Command{
	Use:   "humanize",
	Short: "Strip AI-sounding phrasing from existing text (StyleGuard only)",
	Long: `Run a StyleGuard-only pass over existing text to remove AI-sounding phrases,
without re-generating in a persona's voice. Persona is optional. This does not
consume generation quota.

Provide the source text via --text, --file, or stdin.

Examples:
  toneclone humanize --text="In today's fast-paced world, we leverage synergies..."
  cat draft.md | toneclone humanize --json`,
	RunE: runHumanize,
}

func init() {
	rootCmd.AddCommand(personalizeCmd)
	rootCmd.AddCommand(humanizeCmd)

	personalizeCmd.Flags().StringVar(&personalizePersona, "persona", "", "persona ID or name to rewrite in")
	personalizeCmd.Flags().StringVar(&personalizeText, "text", "", "text to rewrite")
	personalizeCmd.Flags().StringVar(&personalizeFile, "file", "", "file containing text to rewrite")
	personalizeCmd.Flags().StringVar(&personalizeKnowledge, "knowledge", "", "knowledge card ID or name (comma-separated for multiple)")
	personalizeCmd.Flags().IntVar(&personalizeTimeout, "timeout", 30, "request timeout in seconds")
	personalizeCmd.Flags().BoolVar(&personalizeReviewLink, "review-link", false, "create a web review session and return a reviewUrl")
	personalizeCmd.MarkFlagRequired("persona")

	humanizeCmd.Flags().StringVar(&humanizePersona, "persona", "", "optional persona ID or name (uses its StyleGuard config)")
	humanizeCmd.Flags().StringVar(&humanizeText, "text", "", "text to humanize")
	humanizeCmd.Flags().StringVar(&humanizeFile, "file", "", "file containing text to humanize")
	humanizeCmd.Flags().IntVar(&humanizeTimeout, "timeout", 30, "request timeout in seconds")
	humanizeCmd.Flags().BoolVar(&humanizeReviewLink, "review-link", false, "create a web review session and return a reviewUrl")
}

// sourceText resolves input from --text, --file, or stdin (in that order).
func sourceText(text, file string) (string, error) {
	if text != "" {
		return text, nil
	}
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", file, err)
		}
		return string(data), nil
	}
	return readWritePromptFromStdin()
}

func runPersonalize(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient(personalizeTimeout)
	if err != nil {
		return err
	}

	source, err := sourceText(personalizeText, personalizeFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("no text to personalize: provide --text, --file, or stdin")
	}

	persona, err := validatePersona(cmd.Context(), apiClient, personalizePersona)
	if err != nil {
		return fmt.Errorf("persona validation failed: %w", err)
	}

	knowledgeCardID, knowledgeCardIDs, err := resolveKnowledgeCards(cmd.Context(), apiClient, personalizeKnowledge)
	if err != nil {
		return err
	}

	request := &client.GenerateTextRequest{
		Prompt:           "Rewrite the following text so it sounds like this persona, preserving the meaning.",
		PersonaID:        persona.PersonaID,
		Selection:        source,
		Document:         source,
		KnowledgeCardID:  knowledgeCardID,
		KnowledgeCardIDs: knowledgeCardIDs,
		CreateSession:    personalizeReviewLink,
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(personalizeTimeout)*time.Second)
	defer cancel()

	response, err := apiClient.Generate.Text(ctx, request)
	if err != nil {
		return fmt.Errorf("personalize failed: %w", err)
	}

	if jsonOutput {
		return outputWriteJSON(response, persona)
	}
	return outputWriteText(response, persona)
}

func runHumanize(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClient(humanizeTimeout)
	if err != nil {
		return err
	}

	source, err := sourceText(humanizeText, humanizeFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("no text to humanize: provide --text, --file, or stdin")
	}

	// Persona is optional for humanize.
	var persona *client.Persona
	personaID := ""
	if humanizePersona != "" {
		persona, err = validatePersona(cmd.Context(), apiClient, humanizePersona)
		if err != nil {
			return fmt.Errorf("persona validation failed: %w", err)
		}
		personaID = persona.PersonaID
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(humanizeTimeout)*time.Second)
	defer cancel()

	response, err := apiClient.Generate.Humanize(ctx, source, personaID, humanizeReviewLink)
	if err != nil {
		return fmt.Errorf("humanize failed: %w", err)
	}

	if jsonOutput {
		return outputWriteJSON(response, persona)
	}
	return outputWriteText(response, persona)
}

// resolveKnowledgeCards validates a comma-separated --knowledge value into a
// single ID (legacy field) or a list of IDs.
func resolveKnowledgeCards(ctx context.Context, apiClient *client.ToneCloneClient, knowledge string) (string, []string, error) {
	if knowledge == "" {
		return "", nil, nil
	}
	var ids []string
	for _, input := range strings.Split(knowledge, ",") {
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		card, err := validateKnowledgeCard(ctx, apiClient, input)
		if err != nil {
			return "", nil, fmt.Errorf("knowledge card validation failed for '%s': %w", input, err)
		}
		ids = append(ids, card.KnowledgeCardID)
	}
	if len(ids) == 1 {
		return ids[0], nil, nil
	}
	return "", ids, nil
}
