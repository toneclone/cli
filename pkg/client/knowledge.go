package client

import (
	"context"
	"fmt"
)

// KnowledgeClient handles knowledge-related API operations
type KnowledgeClient struct {
	client *Client
}

// NewKnowledgeClient creates a new knowledge client
func NewKnowledgeClient(client *Client) *KnowledgeClient {
	return &KnowledgeClient{client: client}
}

// List retrieves all knowledge cards for the authenticated user
func (k *KnowledgeClient) List(ctx context.Context) ([]KnowledgeCard, error) {
	var cards []KnowledgeCard
	err := k.client.Get(ctx, "/knowledge", &cards)
	if err != nil {
		// Handle empty response case (Content-Length: 0)
		if err.Error() == "unexpected end of JSON input" {
			return []KnowledgeCard{}, nil
		}
		return nil, fmt.Errorf("failed to list knowledge cards: %w", err)
	}
	return cards, nil
}

// Get retrieves a specific knowledge card by ID
func (k *KnowledgeClient) Get(ctx context.Context, knowledgeCardID string) (*KnowledgeCard, error) {
	var card KnowledgeCard
	err := k.client.Get(ctx, fmt.Sprintf("/knowledge/%s", knowledgeCardID), &card)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge card %s: %w", knowledgeCardID, err)
	}
	return &card, nil
}

// Create creates a new knowledge card
func (k *KnowledgeClient) Create(ctx context.Context, knowledge *KnowledgeCard) (*KnowledgeCard, error) {
	var result KnowledgeCard
	err := k.client.Post(ctx, "/knowledge", knowledge, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge card: %w", err)
	}
	return &result, nil
}

// Update updates an existing knowledge card
func (k *KnowledgeClient) Update(ctx context.Context, knowledgeCardID string, knowledge *KnowledgeCard) (*KnowledgeCard, error) {
	var result KnowledgeCard
	err := k.client.Put(ctx, fmt.Sprintf("/knowledge/%s", knowledgeCardID), knowledge, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to update knowledge card %s: %w", knowledgeCardID, err)
	}
	return &result, nil
}

// Delete deletes a knowledge card
func (k *KnowledgeClient) Delete(ctx context.Context, knowledgeCardID string) error {
	err := k.client.Delete(ctx, fmt.Sprintf("/knowledge/%s", knowledgeCardID))
	if err != nil {
		return fmt.Errorf("failed to delete knowledge card %s: %w", knowledgeCardID, err)
	}
	return nil
}

// AssociateWithPersona associates a knowledge card with a persona
func (k *KnowledgeClient) AssociateWithPersona(ctx context.Context, knowledgeCardID, personaID string) error {
	body := map[string]interface{}{
		"knowledgeCardIds": []string{knowledgeCardID},
	}
	err := k.client.Post(ctx, fmt.Sprintf("/personas/%s/knowledge", personaID), body, nil)
	if err != nil {
		return fmt.Errorf("failed to associate knowledge card %s with persona %s: %w", knowledgeCardID, personaID, err)
	}
	return nil
}

// DisassociateFromPersona disassociates a knowledge card from a persona
func (k *KnowledgeClient) DisassociateFromPersona(ctx context.Context, knowledgeCardID, personaID string) error {
	body := map[string]interface{}{
		"knowledgeCardIds": []string{knowledgeCardID},
	}
	err := k.client.doRequest(ctx, "DELETE", fmt.Sprintf("/personas/%s/knowledge", personaID), body, nil)
	if err != nil {
		return fmt.Errorf("failed to disassociate knowledge card %s from persona %s: %w", knowledgeCardID, personaID, err)
	}
	return nil
}

// GetPersonaKnowledge retrieves all knowledge cards associated with a persona
func (k *KnowledgeClient) GetPersonaKnowledge(ctx context.Context, personaID string) ([]KnowledgeCard, error) {
	var cards []KnowledgeCard
	err := k.client.Get(ctx, fmt.Sprintf("/personas/%s/knowledge", personaID), &cards)
	if err != nil {
		return nil, fmt.Errorf("failed to get knowledge for persona %s: %w", personaID, err)
	}
	return cards, nil
}
