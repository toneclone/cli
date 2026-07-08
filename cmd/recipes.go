package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/pkg/client"
)

var (
	recipeCommand           string
	recipeName              string
	recipeDescription       string
	recipeInstruction       string
	recipeInstructionFile   string
	recipeAngleHints        []string
	recipeDefaultDraftCount int
	recipePrompt            string
	recipeDraft             string
	recipeDraftFile         string
	recipeTimeout           int
)

var recipesCmd = &cobra.Command{
	Use:   "recipes",
	Short: "Manage reusable writing recipes",
}

var recipesListCmd = &cobra.Command{Use: "list", Short: "List recipes", RunE: runRecipesList}

var recipesCreateCmd = &cobra.Command{Use: "create", Short: "Create a recipe", RunE: runRecipesCreate}

var recipesUpdateCmd = &cobra.Command{Use: "update <recipe-id>", Short: "Update a recipe", Args: cobra.ExactArgs(1), RunE: runRecipesUpdate}

var recipesSuggestCmd = &cobra.Command{Use: "suggest", Short: "Suggest a recipe from a prompt or draft", RunE: runRecipesSuggest}

func init() {
	rootCmd.AddCommand(recipesCmd)
	recipesCmd.AddCommand(recipesListCmd, recipesCreateCmd, recipesUpdateCmd, recipesSuggestCmd)

	for _, c := range []*cobra.Command{recipesListCmd, recipesCreateCmd, recipesUpdateCmd, recipesSuggestCmd} {
		c.Flags().IntVar(&recipeTimeout, "timeout", 30, "request timeout in seconds")
	}

	recipesCreateCmd.Flags().StringVar(&recipeCommand, "command", "", "recipe command slug")
	recipesCreateCmd.Flags().StringVar(&recipeName, "name", "", "recipe display name")
	recipesCreateCmd.Flags().StringVar(&recipeDescription, "description", "", "recipe description")
	recipesCreateCmd.Flags().StringVar(&recipeInstruction, "instruction", "", "recipe instruction")
	recipesCreateCmd.Flags().StringVar(&recipeInstructionFile, "instruction-file", "", "file containing recipe instruction")
	recipesCreateCmd.Flags().StringArrayVar(&recipeAngleHints, "angle", nil, "angle hint (repeatable)")
	recipesCreateCmd.Flags().IntVar(&recipeDefaultDraftCount, "drafts", 0, "default draft count")
	recipesCreateCmd.MarkFlagRequired("command")
	recipesCreateCmd.MarkFlagRequired("name")

	recipesUpdateCmd.Flags().StringVar(&recipeCommand, "command", "", "recipe command slug")
	recipesUpdateCmd.Flags().StringVar(&recipeName, "name", "", "recipe display name")
	recipesUpdateCmd.Flags().StringVar(&recipeDescription, "description", "", "recipe description")
	recipesUpdateCmd.Flags().StringVar(&recipeInstruction, "instruction", "", "recipe instruction")
	recipesUpdateCmd.Flags().StringVar(&recipeInstructionFile, "instruction-file", "", "file containing recipe instruction")
	recipesUpdateCmd.Flags().StringArrayVar(&recipeAngleHints, "angle", nil, "angle hint (repeatable; replaces existing when provided)")
	recipesUpdateCmd.Flags().IntVar(&recipeDefaultDraftCount, "drafts", 0, "default draft count")

	recipesSuggestCmd.Flags().StringVar(&recipePrompt, "prompt", "", "prompt/context to suggest a recipe from")
	recipesSuggestCmd.Flags().StringVar(&recipeDraft, "draft", "", "draft text to suggest a recipe from")
	recipesSuggestCmd.Flags().StringVar(&recipeDraftFile, "draft-file", "", "file containing draft text")
}

func runRecipesList(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClientWithTimeout(recipeTimeout)
	if err != nil {
		return err
	}
	recipes, err := apiClient.Recipes.List(cmd.Context())
	if err != nil {
		return err
	}
	return outputRecipes(recipes)
}

func runRecipesCreate(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClientWithTimeout(recipeTimeout)
	if err != nil {
		return err
	}
	instruction, err := recipeInstructionSource()
	if err != nil {
		return err
	}
	if strings.TrimSpace(instruction) == "" {
		return fmt.Errorf("recipe instruction is required; use --instruction or --instruction-file")
	}
	recipe, err := apiClient.Recipes.Create(cmd.Context(), &client.RecipeRequest{
		Command:           recipeCommand,
		Name:              recipeName,
		Description:       recipeDescription,
		Instruction:       instruction,
		AngleHints:        recipeAngleHints,
		DefaultDraftCount: recipeDefaultDraftCount,
	})
	if err != nil {
		return err
	}
	return outputRecipe(recipe)
}

func runRecipesUpdate(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClientWithTimeout(recipeTimeout)
	if err != nil {
		return err
	}
	existing, err := findRecipe(cmd.Context(), apiClient, args[0])
	if err != nil {
		return err
	}
	instruction, err := recipeInstructionSource()
	if err != nil {
		return err
	}
	request := &client.RecipeRequest{
		Command:           firstNonEmpty(recipeCommand, existing.Command),
		Name:              firstNonEmpty(recipeName, existing.Name),
		Description:       firstNonEmpty(recipeDescription, existing.Description),
		Instruction:       firstNonEmpty(instruction, existing.Instruction),
		AngleHints:        existing.AngleHints,
		DefaultDraftCount: existing.DefaultDraftCount,
		UpdatedAt:         existing.UpdatedAt,
	}
	if len(recipeAngleHints) > 0 {
		request.AngleHints = recipeAngleHints
	}
	if recipeDefaultDraftCount > 0 {
		request.DefaultDraftCount = recipeDefaultDraftCount
	}
	recipe, err := apiClient.Recipes.Update(cmd.Context(), existing.RecipeID, request)
	if err != nil {
		return err
	}
	return outputRecipe(recipe)
}

func runRecipesSuggest(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClientWithTimeout(recipeTimeout)
	if err != nil {
		return err
	}
	draft, err := optionalSourceText(recipeDraft, recipeDraftFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(recipePrompt) == "" && strings.TrimSpace(draft) == "" {
		return fmt.Errorf("prompt or draft is required; use --prompt, --draft, or --draft-file/stdin")
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), time.Duration(recipeTimeout)*time.Second)
	defer cancel()
	recipe, err := apiClient.Recipes.Suggest(ctx, &client.RecipeSuggestionRequest{Prompt: recipePrompt, Draft: draft})
	if err != nil {
		return err
	}
	return outputRecipe(recipe)
}

func optionalSourceText(text, file string) (string, error) {
	if text != "" || file != "" {
		return sourceText(text, file)
	}
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to check stdin: %w", err)
	}
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return string(data), nil
	}
	return "", nil
}

func recipeInstructionSource() (string, error) {
	if recipeInstruction != "" {
		return recipeInstruction, nil
	}
	if recipeInstructionFile != "" {
		data, err := os.ReadFile(recipeInstructionFile)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", recipeInstructionFile, err)
		}
		return string(data), nil
	}
	return "", nil
}

func findRecipe(ctx context.Context, apiClient *client.ToneCloneClient, recipeID string) (*client.Recipe, error) {
	recipes, err := apiClient.Recipes.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, recipe := range recipes {
		if recipe.RecipeID == recipeID || recipe.Command == recipeID || strings.EqualFold(recipe.Name, recipeID) {
			return &recipe, nil
		}
	}
	return nil, fmt.Errorf("recipe %q not found", recipeID)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func outputRecipes(recipes []client.Recipe) error {
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(recipes)
	}
	for _, recipe := range recipes {
		fmt.Printf("%s\t%s\t%s\t%d drafts\n", recipe.RecipeID, recipe.Command, recipe.Name, recipe.DefaultDraftCount)
		if recipe.Description != "" {
			fmt.Printf("  %s\n", recipe.Description)
		}
	}
	return nil
}

func outputRecipe(recipe *client.Recipe) error {
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(recipe)
	}
	fmt.Printf("%s\t%s\t%s\n", recipe.RecipeID, recipe.Command, recipe.Name)
	if recipe.Description != "" {
		fmt.Printf("Description: %s\n", recipe.Description)
	}
	if recipe.Instruction != "" {
		fmt.Printf("Instruction: %s\n", recipe.Instruction)
	}
	if len(recipe.AngleHints) > 0 {
		fmt.Printf("Angles: %s\n", strings.Join(recipe.AngleHints, ", "))
	}
	if recipe.DefaultDraftCount > 0 {
		fmt.Printf("Default drafts: %d\n", recipe.DefaultDraftCount)
	}
	return nil
}
