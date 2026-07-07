package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/pkg/client"
)

var (
	// Write command flags
	writePersona    string
	writeKnowledge  string
	writePrompt     string
	writeFile       string
	writeOutput     string
	writeVerbose    bool
	writeTimeout    int
	writeReviewLink bool
	writeDrafts     int
)

// writeCmd represents the write command
var writeCmd = &cobra.Command{
	Use:   "write",
	Short: "Generate content using ToneClone AI",
	Long: `Generate text content using ToneClone's AI with a specified persona and prompt.

The prompt can be provided via (in order of priority):
1. --prompt flag (direct text)
2. --file flag (read from file) 
3. stdin (pipe or interactive input)

If both --prompt and --file are provided, --prompt takes precedence.

Examples:
  toneclone write --persona=professional --prompt="Write a product description"
  toneclone write --persona=creative --file=prompt.txt
  toneclone write --persona=business --knowledge=email --prompt="Write a brief email"
  toneclone write --persona=technical --knowledge="documentation,formal" --prompt="Write API docs"
  echo "Write a brief email" | toneclone write --persona=business
  toneclone write --persona=casual (will prompt for input)

Knowledge Card Support:
  --knowledge "name"           Single knowledge card by name or ID
  --knowledge "name1,name2"    Multiple knowledge cards (comma-separated)
  --knowledge "123,456"        Multiple knowledge cards by ID

Output Options:
  --json            JSON output with metadata
  --verbose         Show generation metadata and statistics`,
	RunE: runWrite,
}

func init() {
	rootCmd.AddCommand(writeCmd)

	// Write command flags
	writeCmd.Flags().StringVar(&writePersona, "persona", "", "persona ID or name to use for generation")
	writeCmd.Flags().StringVar(&writeKnowledge, "knowledge", "", "knowledge card ID or name (supports comma-separated multiple cards)")
	writeCmd.Flags().StringVar(&writePrompt, "prompt", "", "text prompt for generation")
	writeCmd.Flags().StringVar(&writeFile, "file", "", "file containing the prompt")
	writeCmd.Flags().StringVar(&writeOutput, "output", "", "deprecated compatibility alias: use --json for JSON output")
	writeCmd.Flags().MarkHidden("output")
	writeCmd.Flags().BoolVar(&writeVerbose, "verbose", false, "show generation metadata and statistics")
	writeCmd.Flags().IntVar(&writeTimeout, "timeout", 30, "request timeout in seconds")
	writeCmd.Flags().BoolVar(&writeReviewLink, "review-link", false, "create a web review session and return a reviewUrl for the draft")
	writeCmd.Flags().IntVarP(&writeDrafts, "drafts", "n", 1, "number of draft variants to generate (1-5), each from a different angle")

	// Make persona required
	writeCmd.MarkFlagRequired("persona")
}

func runWrite(cmd *cobra.Command, args []string) error {
	if err := validateWriteDrafts(writeDrafts); err != nil {
		return err
	}
	if err := normalizeWriteOutput(writeOutput); err != nil {
		return err
	}

	apiClient, err := newAPIClient(writeTimeout)
	if err != nil {
		return err
	}

	// Get the prompt
	prompt, err := getWritePrompt()
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	// Validate persona exists
	persona, err := validatePersona(cmd.Context(), apiClient, writePersona)
	if err != nil {
		return fmt.Errorf("persona validation failed: %w", err)
	}

	// Validate knowledge cards if specified
	var knowledgeCardID string
	var knowledgeCardIDs []string
	var knowledgeCardNames []string
	var validatedKnowledgeCards []*client.KnowledgeCard
	if writeKnowledge != "" {
		// Support comma-separated knowledge cards
		knowledgeInputs := strings.Split(writeKnowledge, ",")
		for i, knowledgeInput := range knowledgeInputs {
			knowledgeInputs[i] = strings.TrimSpace(knowledgeInput)
		}

		// Validate each knowledge card
		for _, knowledgeInput := range knowledgeInputs {
			if knowledgeInput == "" {
				continue
			}
			card, err := validateKnowledgeCard(cmd.Context(), apiClient, knowledgeInput)
			if err != nil {
				return fmt.Errorf("knowledge card validation failed for '%s': %w", knowledgeInput, err)
			}
			validatedKnowledgeCards = append(validatedKnowledgeCards, card)
		}

		// Set up knowledge information
		if len(validatedKnowledgeCards) > 0 {
			for _, card := range validatedKnowledgeCards {
				knowledgeCardNames = append(knowledgeCardNames, card.Name)
			}

			if len(validatedKnowledgeCards) == 1 {
				// Single knowledge card - use legacy field for backward compatibility
				knowledgeCardID = validatedKnowledgeCards[0].KnowledgeCardID
			} else {
				// Multiple knowledge cards - use new array field
				knowledgeCardIDs = make([]string, 0, len(validatedKnowledgeCards))
				for _, card := range validatedKnowledgeCards {
					knowledgeCardIDs = append(knowledgeCardIDs, card.KnowledgeCardID)
				}
			}
		}
	}

	// Create generation request (single or no knowledge card)
	request := &client.GenerateTextRequest{
		Prompt:           prompt,
		PersonaID:        persona.PersonaID,
		KnowledgeCardID:  knowledgeCardID,
		KnowledgeCardIDs: knowledgeCardIDs,
		CreateSession:    writeReviewLink,
	}

	// Show generation info if verbose
	if writeVerbose {
		fmt.Fprintf(os.Stderr, "Generating text with persona: %s (%s)\n", persona.Name, persona.PersonaID)
		switch {
		case len(knowledgeCardIDs) > 0:
			fmt.Fprintf(os.Stderr, "Using knowledge cards: %s\n", strings.Join(knowledgeCardNames, ", "))
		case knowledgeCardID != "":
			if len(knowledgeCardNames) > 0 && knowledgeCardNames[0] != "" {
				fmt.Fprintf(os.Stderr, "Using knowledge card: %s (%s)\n", knowledgeCardNames[0], knowledgeCardID)
			} else {
				fmt.Fprintf(os.Stderr, "Using knowledge card: %s\n", knowledgeCardID)
			}
		}
		fmt.Fprintf(os.Stderr, "Prompt length: %d characters\n", len(prompt))
		fmt.Fprintf(os.Stderr, "Generating...\n\n")
	}

	// Generate text
	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(writeTimeout)*time.Second)
	defer cancel()

	// Multi-draft path: request several variants and render them together.
	if writeDrafts > 1 {
		request.N = writeDrafts
		drafts, err := apiClient.Generate.TextVariants(ctx, request)
		if err != nil {
			return classifyRateLimit(err)
		}
		if jsonOutput {
			return outputDraftsJSON(drafts, persona)
		}
		return outputDraftsText(drafts, persona)
	}

	response, err := apiClient.Generate.Text(ctx, request)
	if err != nil {
		return classifyRateLimit(err)
	}

	// Output based on format
	if jsonOutput {
		return outputWriteJSON(response, persona)
	}

	return outputWriteText(response, persona)
}

func validateWriteDrafts(drafts int) error {
	if drafts < 1 || drafts > 5 {
		return fmt.Errorf("--drafts must be between 1 and 5")
	}
	return nil
}

func normalizeWriteOutput(output string) error {
	switch output {
	case "", "text", "json":
		if output == "json" {
			jsonOutput = true
		}
		return nil
	default:
		return fmt.Errorf("--output is deprecated; use --json for JSON output")
	}
}

// classifyRateLimit preserves the friendly rate-limit message while wrapping the
// typed error (%w) so the global classifier can still emit a structured code.
func classifyRateLimit(err error) error {
	var rateLimitErr *client.RateLimitError
	if errors.As(err, &rateLimitErr) {
		if rateLimitErr.RetryAfterSeconds > 0 {
			return fmt.Errorf("Rate limit exceeded. Please try again in %d seconds: %w", rateLimitErr.RetryAfterSeconds, err)
		}
		return fmt.Errorf("Rate limit exceeded. Please wait before making another request: %w", err)
	}
	return fmt.Errorf("text generation failed: %w", err)
}

func getWritePrompt() (string, error) {
	// Priority: --prompt flag > --file flag > stdin
	if writePrompt != "" {
		return writePrompt, nil
	}

	if writeFile != "" {
		return readWritePromptFromFile(writeFile)
	}

	return readWritePromptFromStdin()
}

func readWritePromptFromFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	return string(data), nil
}

func readWritePromptFromStdin() (string, error) {
	// Check if stdin has data (piped input)
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to check stdin: %w", err)
	}

	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Piped input - read all data
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return string(data), nil
	}

	// Interactive input - prompt user
	fmt.Print("Enter your prompt (end with Ctrl+D or empty line):\n")
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" && len(lines) > 0 {
			break // Empty line ends input if we have content
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.Join(lines, "\n"), nil
}

func outputWriteText(response *client.GenerateTextResponse, persona *client.Persona) error {
	// Just output the generated text
	fmt.Print(response.Text)

	// Add newline if the text doesn't end with one
	if !strings.HasSuffix(response.Text, "\n") {
		fmt.Println()
	}

	// Share the review link (if any) on stderr so it doesn't pollute piped output.
	if response.ReviewURL != "" {
		fmt.Fprintf(os.Stderr, "\nReview/edit: %s\n", response.ReviewURL)
	}

	// Show metadata if verbose
	if writeVerbose {
		fmt.Fprintf(os.Stderr, "\n--- Generation Metadata ---\n")
		if persona != nil {
			fmt.Fprintf(os.Stderr, "Persona: %s (%s)\n", persona.Name, persona.PersonaID)
		}
		if response.Model != "" {
			fmt.Fprintf(os.Stderr, "Model: %s\n", response.Model)
		}
		if response.Tokens > 0 {
			fmt.Fprintf(os.Stderr, "Tokens generated: %d\n", response.Tokens)
		}
	}

	return nil
}

func outputWriteJSON(response *client.GenerateTextResponse, persona *client.Persona) error {
	output := map[string]interface{}{
		"text": response.Text,
	}
	if persona != nil {
		output["persona"] = map[string]string{
			"id":   persona.PersonaID,
			"name": persona.Name,
		}
	}

	if response.Model != "" {
		output["model"] = response.Model
	}
	if response.Tokens > 0 {
		output["tokens"] = response.Tokens
	}
	if response.KnowledgeCardID != "" {
		output["knowledge_card_id"] = response.KnowledgeCardID
	}
	if response.ReviewURL != "" {
		output["reviewUrl"] = response.ReviewURL
	}
	if response.SessionID != "" {
		output["sessionId"] = response.SessionID
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

func outputDraftsText(drafts *client.GenerateDraftsResponse, persona *client.Persona) error {
	if drafts.AnglePlan != nil && drafts.AnglePlan.StrategyNote != "" {
		fmt.Fprintf(os.Stderr, "Strategy: %s\n\n", drafts.AnglePlan.StrategyNote)
	}
	for i, v := range drafts.Variants {
		label := fmt.Sprintf("Draft %d", i+1)
		if v.Angle != nil {
			if v.Angle.ShortLabel != "" {
				label = fmt.Sprintf("Draft %d - %s", i+1, v.Angle.ShortLabel)
			} else if v.Angle.Title != "" {
				label = fmt.Sprintf("Draft %d - %s", i+1, v.Angle.Title)
			}
		}
		fmt.Fprintf(os.Stderr, "=== %s ===\n", label)
		if writeVerbose && v.Angle != nil {
			if v.Angle.Approach != "" {
				fmt.Fprintf(os.Stderr, "Approach: %s\n", v.Angle.Approach)
			}
			if len(v.Angle.VoiceEmphasis) > 0 {
				fmt.Fprintf(os.Stderr, "Voice: %s\n", strings.Join(v.Angle.VoiceEmphasis, ", "))
			}
			if len(v.Angle.Avoid) > 0 {
				fmt.Fprintf(os.Stderr, "Avoid: %s\n", strings.Join(v.Angle.Avoid, ", "))
			}
			fmt.Fprintln(os.Stderr)
		}
		fmt.Print(v.Content)
		if !strings.HasSuffix(v.Content, "\n") {
			fmt.Println()
		}
		if i < len(drafts.Variants)-1 {
			fmt.Println()
		}
	}
	if drafts.ReviewURL != "" {
		fmt.Fprintf(os.Stderr, "\nReview/edit: %s\n", drafts.ReviewURL)
	}
	return nil
}

func outputDraftsJSON(drafts *client.GenerateDraftsResponse, persona *client.Persona) error {
	output := map[string]interface{}{
		"drafts": drafts.Variants,
	}
	if persona != nil {
		output["persona"] = map[string]string{
			"id":   persona.PersonaID,
			"name": persona.Name,
		}
	}
	if drafts.AnglePlan != nil {
		output["anglePlan"] = drafts.AnglePlan
	}
	if drafts.ReviewURL != "" {
		output["reviewUrl"] = drafts.ReviewURL
	}
	if drafts.SessionID != "" {
		output["sessionId"] = drafts.SessionID
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
