package client

import (
	"context"
	"fmt"
	"net/url"
)

// StyleGuardClient handles style guard API operations
type StyleGuardClient struct {
	client *Client
}

// NewStyleGuardClient creates a new style guard client
func NewStyleGuardClient(client *Client) *StyleGuardClient {
	return &StyleGuardClient{client: client}
}

// CreateStyleGuardWordRequest represents a request to create a style guard rule
type CreateStyleGuardWordRequest struct {
	Word              string `json:"word"`
	PersonaID         string `json:"personaId,omitempty"`
	Mode              string `json:"mode"`
	CustomReplacement string `json:"customReplacement,omitempty"`
}

// UpdateStyleGuardWordRequest represents a request to update a style guard rule
type UpdateStyleGuardWordRequest struct {
	Word              string `json:"word,omitempty"`
	Mode              string `json:"mode,omitempty"`
	CustomReplacement string `json:"customReplacement,omitempty"`
}

// ApplyBundleRequest represents a request to apply a style guard bundle
type ApplyBundleRequest struct {
	PersonaID  string `json:"personaId,omitempty"`
	BundleType string `json:"bundleType,omitempty"`
}

// BundlePreviewItem represents a single item in a bundle preview
type BundlePreviewItem struct {
	Word              string `json:"word"`
	Mode              string `json:"mode"`
	CustomReplacement string `json:"replacement,omitempty"`
	Category          string `json:"category,omitempty"`
}

// BundlePreviewResponse represents the API response for bundle preview
type BundlePreviewResponse struct {
	Version string              `json:"version"`
	Type    string              `json:"type"`
	Items   []BundlePreviewItem `json:"items"`
}

// BundleStatusResponse represents the status of a bundle application
type BundleStatusResponse struct {
	Applied     bool   `json:"applied"`
	Version     string `json:"version,omitempty"`
	Type        string `json:"type,omitempty"`
	WordsCount  int    `json:"wordsCount"`
	ActiveCount int    `json:"activeCount"`
}

// List retrieves all global style guard rules
func (s *StyleGuardClient) List(ctx context.Context) ([]StyleGuardWord, error) {
	var words []StyleGuardWord
	err := s.client.Get(ctx, "/style-guard", &words)
	if err != nil {
		return nil, fmt.Errorf("failed to list style guard rules: %w", err)
	}
	return words, nil
}

// ListForPersona retrieves style guard rules for a specific persona
func (s *StyleGuardClient) ListForPersona(ctx context.Context, personaID string) ([]StyleGuardWord, error) {
	var words []StyleGuardWord
	err := s.client.Get(ctx, fmt.Sprintf("/personas/%s/style-guard", personaID), &words)
	if err != nil {
		return nil, fmt.Errorf("failed to list style guard rules for persona %s: %w", personaID, err)
	}
	return words, nil
}

// Create creates a new style guard rule
func (s *StyleGuardClient) Create(ctx context.Context, req *CreateStyleGuardWordRequest) (*StyleGuardWord, error) {
	var result StyleGuardWord
	err := s.client.Post(ctx, "/style-guard", req, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to create style guard rule: %w", err)
	}
	return &result, nil
}

// Update updates an existing style guard rule
func (s *StyleGuardClient) Update(ctx context.Context, styleGuardID string, req *UpdateStyleGuardWordRequest) (*StyleGuardWord, error) {
	var result StyleGuardWord
	err := s.client.Put(ctx, fmt.Sprintf("/style-guard/%s", styleGuardID), req, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to update style guard rule %s: %w", styleGuardID, err)
	}
	return &result, nil
}

// Delete deletes a style guard rule
func (s *StyleGuardClient) Delete(ctx context.Context, styleGuardID string) error {
	err := s.client.Delete(ctx, fmt.Sprintf("/style-guard/%s", styleGuardID))
	if err != nil {
		return fmt.Errorf("failed to delete style guard rule %s: %w", styleGuardID, err)
	}
	return nil
}

// BundlePreview previews items in a style guard bundle
func (s *StyleGuardClient) BundlePreview(ctx context.Context, bundleType string) (*BundlePreviewResponse, error) {
	var response BundlePreviewResponse
	path := "/style-guard/bundle/preview"
	if bundleType != "" {
		path = fmt.Sprintf("%s?bundleType=%s", path, url.QueryEscape(bundleType))
	}
	err := s.client.Get(ctx, path, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to preview bundle: %w", err)
	}
	return &response, nil
}

// BundleStatus retrieves the bundle application status
func (s *StyleGuardClient) BundleStatus(ctx context.Context, personaID, bundleType string) (*BundleStatusResponse, error) {
	var result BundleStatusResponse
	params := url.Values{}
	if personaID != "" {
		params.Set("personaId", personaID)
	}
	if bundleType != "" {
		params.Set("bundleType", bundleType)
	}
	path := "/style-guard/bundle/status"
	if len(params) > 0 {
		path = fmt.Sprintf("%s?%s", path, params.Encode())
	}
	err := s.client.Get(ctx, path, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to get bundle status: %w", err)
	}
	return &result, nil
}

// BundleApply applies a style guard bundle
func (s *StyleGuardClient) BundleApply(ctx context.Context, req *ApplyBundleRequest) error {
	err := s.client.Post(ctx, "/style-guard/bundle/apply", req, nil)
	if err != nil {
		return fmt.Errorf("failed to apply bundle: %w", err)
	}
	return nil
}

// BundleRemove removes bundle-sourced style guard rules
func (s *StyleGuardClient) BundleRemove(ctx context.Context, personaID string) error {
	path := "/style-guard/bundle/remove"
	if personaID != "" {
		path = fmt.Sprintf("%s?personaId=%s", path, url.QueryEscape(personaID))
	}
	err := s.client.Delete(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to remove bundle: %w", err)
	}
	return nil
}
