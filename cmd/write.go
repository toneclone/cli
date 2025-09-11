package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/internal/config"
	"github.com/toneclone/cli/pkg/client"
)

var (
	// Write command flags
	writePersona   string
	writeKnowledge string
	writePrompt    string
	writeFile      string
	writeOutput    string
	writeVerbose   bool
	writeTimeout   int
	writeJson      bool
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
  --output text     Plain text output (default)
  --output json     JSON output with metadata
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
	writeCmd.Flags().StringVar(&writeOutput, "output", "text", "output format: text, json")
	writeCmd.Flags().BoolVar(&writeVerbose, "verbose", false, "show generation metadata and statistics")
	writeCmd.Flags().IntVar(&writeTimeout, "timeout", 30, "request timeout in seconds")
	writeCmd.Flags().BoolVar(&writeJson, "json", false, "output in JSON format (shorthand for --output json)")

	// Make persona required
	writeCmd.MarkFlagRequired("persona")
}

func runWrite(cmd *cobra.Command, args []string) error {
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
		time.Duration(writeTimeout)*time.Second,
	)

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
			if len(validatedKnowledgeCards) == 1 {
				// Single knowledge card - use legacy field for backward compatibility
				knowledgeCardID = validatedKnowledgeCards[0].KnowledgeCardID
			} else {
				// Multiple knowledge cards - use new array field
				var knowledgeCardIDs []string
				var knowledgeNames []string
				for _, card := range validatedKnowledgeCards {
					knowledgeCardIDs = append(knowledgeCardIDs, card.KnowledgeCardID)
					knowledgeNames = append(knowledgeNames, card.Name)
				}

				// Create generation request with multiple knowledge cards
				request := &client.GenerateTextRequest{
					Prompt:           prompt,
					PersonaID:        persona.PersonaID,
					KnowledgeCardIDs: knowledgeCardIDs,
				}

				// Show generation info if verbose
				if writeVerbose {
					fmt.Fprintf(os.Stderr, "Generating text with persona: %s (%s)\n", persona.Name, persona.PersonaID)
					fmt.Fprintf(os.Stderr, "Using knowledge cards: %s\n", strings.Join(knowledgeNames, ", "))
				}

				// Generate and return early for multiple knowledge cards
				response, err := apiClient.Generate.Text(cmd.Context(), request)
				if err != nil {
					// Check for rate limit error and provide helpful message
					if rateLimitErr, ok := err.(*client.RateLimitError); ok {
						if rateLimitErr.RetryAfterSeconds > 0 {
							return fmt.Errorf("Rate limit exceeded. Please try again in %d seconds", rateLimitErr.RetryAfterSeconds)
						}
						return fmt.Errorf("Rate limit exceeded. Please wait before making another request")
					}
					return fmt.Errorf("text generation failed: %w", err)
				}
				return outputWriteText(response, persona)
			}
		}
	}

	// Create generation request (single or no knowledge card)
	request := &client.GenerateTextRequest{
		Prompt:          prompt,
		PersonaID:       persona.PersonaID,
		KnowledgeCardID: knowledgeCardID,
	}

	// Show generation info if verbose
	if writeVerbose {
		fmt.Fprintf(os.Stderr, "Generating text with persona: %s (%s)\n", persona.Name, persona.PersonaID)
		if knowledgeCardID != "" {
			fmt.Fprintf(os.Stderr, "Using knowledge card: %s\n", knowledgeCardID)
		}
		fmt.Fprintf(os.Stderr, "Prompt length: %d characters\n", len(prompt))
		fmt.Fprintf(os.Stderr, "Generating...\n\n")
	}

	// Generate text
	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(writeTimeout)*time.Second)
	defer cancel()

	response, err := apiClient.Generate.Text(ctx, request)
	if err != nil {
		// Check for rate limit error and provide helpful message
		if rateLimitErr, ok := err.(*client.RateLimitError); ok {
			if rateLimitErr.RetryAfterSeconds > 0 {
				return fmt.Errorf("Rate limit exceeded. Please try again in %d seconds", rateLimitErr.RetryAfterSeconds)
			}
			return fmt.Errorf("Rate limit exceeded. Please wait before making another request")
		}
		return fmt.Errorf("text generation failed: %w", err)
	}

	// Output based on format
	if writeJson || writeOutput == "json" {
		return outputWriteJSON(response, persona)
	}

	return outputWriteText(response, persona)
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

	// Show metadata if verbose
	if writeVerbose {
		fmt.Fprintf(os.Stderr, "\n--- Generation Metadata ---\n")
		fmt.Fprintf(os.Stderr, "Persona: %s (%s)\n", persona.Name, persona.PersonaID)
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
		"persona": map[string]string{
			"id":   persona.PersonaID,
			"name": persona.Name,
		},
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

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
