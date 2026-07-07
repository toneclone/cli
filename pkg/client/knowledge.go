package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"path/filepath"
)

// KnowledgeClient handles knowledge-related API operations
type KnowledgeClient struct {
	client *Client
}

const maxKnowledgeSourceFileBytes = 10 * 1024 * 1024

type KnowledgeCardSource struct {
	SourceID             string `json:"sourceId"`
	KnowledgeCardID      string `json:"knowledgeCardId"`
	UserID               string `json:"userId,omitempty"`
	Type                 string `json:"type"`
	DisplayName          string `json:"displayName"`
	URL                  string `json:"url,omitempty"`
	Filename             string `json:"filename,omitempty"`
	MimeType             string `json:"mimeType,omitempty"`
	Extension            string `json:"extension,omitempty"`
	SizeBytes            int64  `json:"sizeBytes,omitempty"`
	Status               string `json:"status"`
	ErrorMessage         string `json:"errorMessage,omitempty"`
	ExtractedTextPreview string `json:"extractedTextPreview,omitempty"`
	ExtractedCharCount   int    `json:"extractedCharCount,omitempty"`
	ContentHash          string `json:"contentHash,omitempty"`
	LastProcessedAt      string `json:"lastProcessedAt,omitempty"`
	CreatedAt            string `json:"createdAt,omitempty"`
	UpdatedAt            string `json:"updatedAt,omitempty"`
}

type KnowledgeCardSynthesis struct {
	Name         string   `json:"name,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
	Summary      string   `json:"summary,omitempty"`
	KeyFacts     []string `json:"keyFacts,omitempty"`
	UsageNotes   []string `json:"usageNotes,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

type KnowledgeCardIngestResponse struct {
	KnowledgeCard KnowledgeCard          `json:"knowledgeCard"`
	Source        KnowledgeCardSource    `json:"source"`
	Synthesis     KnowledgeCardSynthesis `json:"synthesis,omitempty"`
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
	err := k.client.Get(ctx, fmt.Sprintf("/knowledge/%s", url.PathEscape(knowledgeCardID)), &card)
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

// CreateFromURL creates a Knowledge Card by fetching and synthesizing durable
// instructions from a public HTTP(S) URL.
func (k *KnowledgeClient) CreateFromURL(ctx context.Context, rawURL, instructionsHint string) (*KnowledgeCardIngestResponse, error) {
	body := map[string]string{"url": rawURL}
	if instructionsHint != "" {
		body["instructionsHint"] = instructionsHint
	}
	var result KnowledgeCardIngestResponse
	if err := k.client.Post(ctx, "/knowledge/from-url", body, &result); err != nil {
		return nil, fmt.Errorf("failed to create knowledge card from URL: %w", err)
	}
	return &result, nil
}

// CreateFromFile creates a Knowledge Card from an uploaded document. filename
// should include the source extension so the backend can choose the extractor.
func (k *KnowledgeClient) CreateFromFile(ctx context.Context, filename string, content io.Reader, instructionsHint string) (*KnowledgeCardIngestResponse, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(filename))
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart file part: %w", err)
	}
	written, err := io.Copy(part, io.LimitReader(content, maxKnowledgeSourceFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}
	if written > maxKnowledgeSourceFileBytes {
		return nil, fmt.Errorf("source file is too large: maximum size is 10MB")
	}
	if instructionsHint != "" {
		if err := writer.WriteField("instructionsHint", instructionsHint); err != nil {
			return nil, fmt.Errorf("failed to add instructions hint: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finish multipart upload: %w", err)
	}

	var result KnowledgeCardIngestResponse
	if err := k.client.PostMultipart(ctx, "/knowledge/from-file", writer.FormDataContentType(), body.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to create knowledge card from file: %w", err)
	}
	return &result, nil
}

// Sources lists source provenance metadata for a Knowledge Card.
func (k *KnowledgeClient) Sources(ctx context.Context, knowledgeCardID string) ([]KnowledgeCardSource, error) {
	var sources []KnowledgeCardSource
	path := fmt.Sprintf("/knowledge/%s/sources", url.PathEscape(knowledgeCardID))
	if err := k.client.Get(ctx, path, &sources); err != nil {
		return nil, fmt.Errorf("failed to list knowledge card sources: %w", err)
	}
	return sources, nil
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
