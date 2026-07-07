package client

import (
	"context"
	"fmt"
	"net/url"
)

// RecipesClient handles user recipe API operations.
type RecipesClient struct {
	client *Client
}

func NewRecipesClient(client *Client) *RecipesClient {
	return &RecipesClient{client: client}
}

type Recipe struct {
	RecipeID          string   `json:"recipeId,omitempty"`
	Command           string   `json:"command"`
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	Instruction       string   `json:"instruction"`
	AngleHints        []string `json:"angleHints,omitempty"`
	DefaultDraftCount int      `json:"defaultDraftCount,omitempty"`
	Source            string   `json:"source,omitempty"`
	CreatedAt         string   `json:"createdAt,omitempty"`
	UpdatedAt         string   `json:"updatedAt,omitempty"`
}

type RecipeRequest struct {
	Command           string   `json:"command"`
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	Instruction       string   `json:"instruction"`
	AngleHints        []string `json:"angleHints,omitempty"`
	DefaultDraftCount int      `json:"defaultDraftCount,omitempty"`
	UpdatedAt         string   `json:"updatedAt,omitempty"`
}

type RecipeSuggestionRequest struct {
	Prompt string `json:"prompt,omitempty"`
	Draft  string `json:"draft,omitempty"`
}

func (r *RecipesClient) List(ctx context.Context) ([]Recipe, error) {
	var recipes []Recipe
	if err := r.client.Get(ctx, "/recipes", &recipes); err != nil {
		return nil, fmt.Errorf("failed to list recipes: %w", err)
	}
	return recipes, nil
}

func (r *RecipesClient) Create(ctx context.Context, request *RecipeRequest) (*Recipe, error) {
	var recipe Recipe
	if err := r.client.Post(ctx, "/recipes", request, &recipe); err != nil {
		return nil, fmt.Errorf("failed to create recipe: %w", err)
	}
	return &recipe, nil
}

func (r *RecipesClient) Update(ctx context.Context, recipeID string, request *RecipeRequest) (*Recipe, error) {
	var recipe Recipe
	path := "/recipes/" + url.PathEscape(recipeID)
	if err := r.client.Put(ctx, path, request, &recipe); err != nil {
		return nil, fmt.Errorf("failed to update recipe: %w", err)
	}
	return &recipe, nil
}

func (r *RecipesClient) Suggest(ctx context.Context, request *RecipeSuggestionRequest) (*Recipe, error) {
	var recipe Recipe
	if err := r.client.Post(ctx, "/recipes/suggest", request, &recipe); err != nil {
		return nil, fmt.Errorf("failed to suggest recipe: %w", err)
	}
	return &recipe, nil
}
