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
	// Generate command flags
	generatePersona   string
	generateProfile   string
	generatePrompt    string
	generateFile      string
	generateMaxTokens int
	generateModel     string
	generateOutput    string
	generateVerbose   bool
	generateTimeout   int
	generateJson      bool
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate content using ToneClone AI",
	Long: `Generate various types of content using ToneClone's AI capabilities.

Use subcommands to specify the type of content to generate:
- text: Generate text content using a persona and prompt

Examples:
  toneclone generate text --persona=professional --prompt="Write a product description"
  toneclone generate text --file=prompt.txt --persona=creative
  echo "Write a blog post" | toneclone generate text --persona=blogger`,
}

// textCmd represents the text subcommand
var textCmd = &cobra.Command{
	Use:   "text",
	Short: "Generate text content",
	Long: `Generate text content using ToneClone's AI with a specified persona and prompt.

The prompt can be provided via:
1. --prompt flag (direct text)
2. --file flag (read from file)
3. stdin (pipe or interactive input)

Examples:
  toneclone generate text --persona=professional --prompt="Write a product description for a smartphone"
  toneclone generate text --persona=creative --file=prompt.txt
  toneclone generate text --persona=business --profile=email --prompt="Write a brief email"
  toneclone generate text --persona=technical --profile="documentation,formal" --prompt="Write API docs"
  echo "Write a brief email" | toneclone generate text --persona=business
  toneclone generate text --persona=casual (will prompt for input)

Profile Support:
  --profile "name"           Single profile by name or ID
  --profile "name1,name2"    Multiple profiles (comma-separated)
  --profile "123,456"        Multiple profiles by ID

Output Options:
  --output text     Plain text output (default)
  --output json     JSON output with metadata
  --verbose         Show generation metadata and statistics`,
	RunE: runGenerateText,
}

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.AddCommand(textCmd)

	// Text generation flags
	textCmd.Flags().StringVar(&generatePersona, "persona", "", "persona ID or name to use for generation")
	textCmd.Flags().StringVar(&generateProfile, "profile", "", "profile ID or name (supports comma-separated multiple profiles)")
	textCmd.Flags().StringVar(&generatePrompt, "prompt", "", "text prompt for generation")
	textCmd.Flags().StringVar(&generateFile, "file", "", "file containing the prompt")
	textCmd.Flags().IntVar(&generateMaxTokens, "max-tokens", 0, "maximum number of tokens to generate")
	textCmd.Flags().StringVar(&generateModel, "model", "", "AI model to use for generation")
	textCmd.Flags().StringVar(&generateOutput, "output", "text", "output format: text, json")
	textCmd.Flags().BoolVar(&generateVerbose, "verbose", false, "show generation metadata and statistics")
	textCmd.Flags().IntVar(&generateTimeout, "timeout", 30, "request timeout in seconds")
	textCmd.Flags().BoolVar(&generateJson, "json", false, "output in JSON format (shorthand for --output json)")

	// Make persona required
	textCmd.MarkFlagRequired("persona")
}

func runGenerateText(cmd *cobra.Command, args []string) error {
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
		time.Duration(generateTimeout)*time.Second,
	)

	// Get the prompt
	prompt, err := getPrompt()
	if err != nil {
		return fmt.Errorf("failed to get prompt: %w", err)
	}

	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("prompt cannot be empty")
	}

	// Validate persona exists
	persona, err := validatePersona(cmd.Context(), apiClient, generatePersona)
	if err != nil {
		return fmt.Errorf("persona validation failed: %w", err)
	}

	// Validate profiles if specified
	var profileID string
	var validatedProfiles []*client.Profile
	if generateProfile != "" {
		// Support comma-separated profiles
		profileInputs := strings.Split(generateProfile, ",")
		for i, profileInput := range profileInputs {
			profileInputs[i] = strings.TrimSpace(profileInput)
		}

		// Validate each profile
		for _, profileInput := range profileInputs {
			if profileInput == "" {
				continue
			}
			profile, err := validateProfile(cmd.Context(), apiClient, profileInput)
			if err != nil {
				return fmt.Errorf("profile validation failed for '%s': %w", profileInput, err)
			}
			validatedProfiles = append(validatedProfiles, profile)
		}

		// Set up profile information
		if len(validatedProfiles) > 0 {
			if len(validatedProfiles) == 1 {
				// Single profile - use legacy field for backward compatibility
				profileID = validatedProfiles[0].ProfileID
			} else {
				// Multiple profiles - use new array field
				var profileIDs []string
				var profileNames []string
				for _, profile := range validatedProfiles {
					profileIDs = append(profileIDs, profile.ProfileID)
					profileNames = append(profileNames, profile.Name)
				}
				
				// Create generation request with multiple profiles
				request := &client.GenerateTextRequest{
					Prompt:     prompt,
					PersonaID:  persona.PersonaID,
					ProfileIDs: profileIDs,
					Model:      generateModel,
				}
				
				// Show generation info if verbose
				if generateVerbose {
					fmt.Fprintf(os.Stderr, "Generating text with persona: %s (%s)\n", persona.Name, persona.PersonaID)
					fmt.Fprintf(os.Stderr, "Using profiles: %s\n", strings.Join(profileNames, ", "))
				}
				
				// Generate and return early for multiple profiles
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
				return outputText(response, persona)
			}
		}
	}

	// Create generation request (single or no profile)
	request := &client.GenerateTextRequest{
		Prompt:    prompt,
		PersonaID: persona.PersonaID,
		ProfileID: profileID,
		Model:     generateModel,
	}

	// Show generation info if verbose
	if generateVerbose {
		fmt.Fprintf(os.Stderr, "Generating text with persona: %s (%s)\n", persona.Name, persona.PersonaID)
		if profileID != "" {
			fmt.Fprintf(os.Stderr, "Using profile: %s\n", profileID)
		}
		if generateModel != "" {
			fmt.Fprintf(os.Stderr, "Using model: %s\n", generateModel)
		}
		fmt.Fprintf(os.Stderr, "Prompt length: %d characters\n", len(prompt))
		fmt.Fprintf(os.Stderr, "Generating...\n\n")
	}

	// Generate text
	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(generateTimeout)*time.Second)
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
	if generateJson || generateOutput == "json" {
		return outputJSON(response, persona)
	}

	return outputText(response, persona)
}

func getPrompt() (string, error) {
	// Priority: --prompt flag > --file flag > stdin
	if generatePrompt != "" {
		return generatePrompt, nil
	}

	if generateFile != "" {
		return readPromptFromFile(generateFile)
	}

	return readPromptFromStdin()
}

func readPromptFromFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filename, err)
	}
	return string(data), nil
}

func readPromptFromStdin() (string, error) {
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

func validatePersona(ctx context.Context, apiClient *client.ToneCloneClient, personaInput string) (*client.Persona, error) {
	// First try to get by ID
	persona, err := apiClient.Personas.Get(ctx, personaInput)
	if err == nil {
		return persona, nil
	}

	// If that fails, try to find by name
	personas, err := apiClient.Personas.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list personas: %w", err)
	}

	// Look for exact name match
	for _, p := range personas {
		if strings.EqualFold(p.Name, personaInput) {
			return &p, nil
		}
	}

	// Look for partial name match
	var matches []client.Persona
	for _, p := range personas {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(personaInput)) {
			matches = append(matches, p)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("persona '%s' not found", personaInput)
	}

	if len(matches) > 1 {
		var names []string
		for _, p := range matches {
			names = append(names, fmt.Sprintf("'%s' (%s)", p.Name, p.PersonaID))
		}
		return nil, fmt.Errorf("multiple personas match '%s': %s", personaInput, strings.Join(names, ", "))
	}

	return &matches[0], nil
}

func validateProfile(ctx context.Context, apiClient *client.ToneCloneClient, profileInput string) (*client.Profile, error) {
	// First try to get by ID
	profile, err := apiClient.Profiles.Get(ctx, profileInput)
	if err == nil {
		return profile, nil
	}

	// If that fails, try to find by name
	profiles, err := apiClient.Profiles.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	// Look for exact name match
	for _, p := range profiles {
		if strings.EqualFold(p.Name, profileInput) {
			return &p, nil
		}
	}

	// Look for partial name match
	var matches []client.Profile
	for _, p := range profiles {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(profileInput)) {
			matches = append(matches, p)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("profile '%s' not found", profileInput)
	}

	if len(matches) > 1 {
		var names []string
		for _, p := range matches {
			names = append(names, fmt.Sprintf("'%s' (%s)", p.Name, p.ProfileID))
		}
		return nil, fmt.Errorf("multiple profiles match '%s': %s", profileInput, strings.Join(names, ", "))
	}

	return &matches[0], nil
}

func outputText(response *client.GenerateTextResponse, persona *client.Persona) error {
	// Just output the generated text
	fmt.Print(response.Text)

	// Add newline if the text doesn't end with one
	if !strings.HasSuffix(response.Text, "\n") {
		fmt.Println()
	}

	// Show metadata if verbose
	if generateVerbose {
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

func outputJSON(response *client.GenerateTextResponse, persona *client.Persona) error {
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
	if response.ProfileID != "" {
		output["profile_id"] = response.ProfileID
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
