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
	writePersona  string
	writeProfile  string
	writePrompt   string
	writeFile     string
	writeOutput   string
	writeVerbose  bool
	writeTimeout  int
	writeJson     bool
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
  toneclone write --persona=business --profile=email --prompt="Write a brief email"
  toneclone write --persona=technical --profile="documentation,formal" --prompt="Write API docs"
  echo "Write a brief email" | toneclone write --persona=business
  toneclone write --persona=casual (will prompt for input)

Profile Support:
  --profile "name"           Single profile by name or ID
  --profile "name1,name2"    Multiple profiles (comma-separated)
  --profile "123,456"        Multiple profiles by ID

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
	writeCmd.Flags().StringVar(&writeProfile, "profile", "", "profile ID or name (supports comma-separated multiple profiles)")
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

	// Validate profiles if specified
	var profileID string
	var validatedProfiles []*client.Profile
	if writeProfile != "" {
		// Support comma-separated profiles
		profileInputs := strings.Split(writeProfile, ",")
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
				}
				
				// Show generation info if verbose
				if writeVerbose {
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
				return outputWriteText(response, persona)
			}
		}
	}

	// Create generation request (single or no profile)
	request := &client.GenerateTextRequest{
		Prompt:    prompt,
		PersonaID: persona.PersonaID,
		ProfileID: profileID,
	}

	// Show generation info if verbose
	if writeVerbose {
		fmt.Fprintf(os.Stderr, "Generating text with persona: %s (%s)\n", persona.Name, persona.PersonaID)
		if profileID != "" {
			fmt.Fprintf(os.Stderr, "Using profile: %s\n", profileID)
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
	if response.ProfileID != "" {
		output["profile_id"] = response.ProfileID
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}