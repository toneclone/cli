package client

import (
	"context"
	"fmt"
)

// PersonasClient handles persona-related API operations
type PersonasClient struct {
	client *Client
}

// NewPersonasClient creates a new personas client
func NewPersonasClient(client *Client) *PersonasClient {
	return &PersonasClient{client: client}
}

// List retrieves all personas
func (p *PersonasClient) List(ctx context.Context) ([]Persona, error) {
	var personas []Persona
	err := p.client.Get(ctx, "/personas", &personas)
	if err != nil {
		return nil, fmt.Errorf("failed to list personas: %w", err)
	}
	return personas, nil
}

// Get retrieves a specific persona by ID
func (p *PersonasClient) Get(ctx context.Context, personaID string) (*Persona, error) {
	var persona Persona
	err := p.client.Get(ctx, fmt.Sprintf("/personas/%s", personaID), &persona)
	if err != nil {
		return nil, fmt.Errorf("failed to get persona %s: %w", personaID, err)
	}
	return &persona, nil
}

// Create creates a new persona
func (p *PersonasClient) Create(ctx context.Context, persona *Persona) (*Persona, error) {
	var result Persona
	err := p.client.Post(ctx, "/personas", persona, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to create persona: %w", err)
	}
	return &result, nil
}

// Update updates an existing persona
func (p *PersonasClient) Update(ctx context.Context, personaID string, persona *Persona) (*Persona, error) {
	var result Persona
	err := p.client.Put(ctx, fmt.Sprintf("/personas/%s", personaID), persona, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to update persona %s: %w", personaID, err)
	}
	return &result, nil
}

// Delete deletes a persona
func (p *PersonasClient) Delete(ctx context.Context, personaID string) error {
	err := p.client.Delete(ctx, fmt.Sprintf("/personas/%s", personaID))
	if err != nil {
		return fmt.Errorf("failed to delete persona %s: %w", personaID, err)
	}
	return nil
}

// ListFiles retrieves files associated with a persona
func (p *PersonasClient) ListFiles(ctx context.Context, personaID string) ([]TrainingFile, error) {
	var response TrainingFileListResponse
	err := p.client.Get(ctx, fmt.Sprintf("/personas/%s/files", personaID), &response)
	if err != nil {
		return nil, fmt.Errorf("failed to list files for persona %s: %w", personaID, err)
	}
	return response.Files, nil
}

// AssociateFiles associates files with a persona
func (p *PersonasClient) AssociateFiles(ctx context.Context, personaID string, fileIDs []string) error {
	body := map[string]interface{}{
		"fileIds": fileIDs,
	}
	err := p.client.Post(ctx, fmt.Sprintf("/personas/%s/files", personaID), body, nil)
	if err != nil {
		return fmt.Errorf("failed to associate files with persona %s: %w", personaID, err)
	}
	return nil
}

// DisassociateFiles removes file associations from a persona
func (p *PersonasClient) DisassociateFiles(ctx context.Context, personaID string, fileIDs []string) error {
	body := map[string]interface{}{
		"fileIds": fileIDs,
	}
	err := p.client.doRequest(ctx, "DELETE", fmt.Sprintf("/personas/%s/files", personaID), body, nil)
	if err != nil {
		return fmt.Errorf("failed to disassociate files from persona %s: %w", personaID, err)
	}
	return nil
}
