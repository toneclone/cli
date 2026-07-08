package client

import (
	"context"
	"fmt"
)

// GenerateClient handles text generation API operations
type GenerateClient struct {
	client *Client
}

type generateResponseBody struct {
	Content   string `json:"content"`
	Done      bool   `json:"done"`
	SessionID string `json:"sessionId"`
	ReviewURL string `json:"reviewUrl"`
}

// NewGenerateClient creates a new generate client
func NewGenerateClient(client *Client) *GenerateClient {
	return &GenerateClient{client: client}
}

// Text generates text using the specified parameters
func (g *GenerateClient) Text(ctx context.Context, request *GenerateTextRequest) (*GenerateTextResponse, error) {
	// Set streaming to false to get JSON response instead of SSE
	streaming := false
	request.Streaming = &streaming

	// Use the standard client Post method for JSON response
	var response generateResponseBody

	if err := g.client.Post(ctx, "/query", request, &response); err != nil {
		return nil, fmt.Errorf("failed to generate text: %w", err)
	}

	return &GenerateTextResponse{
		Text:            response.Content,
		PersonaID:       request.PersonaID,
		KnowledgeCardID: request.KnowledgeCardID,
		Model:           request.Model,
		SessionID:       response.SessionID,
		ReviewURL:       response.ReviewURL,
	}, nil
}

// Humanize runs a StyleGuard-only pass over the given text (no model generation,
// no quota charge). Persona is optional; when provided, its StyleGuard config is
// used. If createSession is true, the response carries a reviewUrl.
func (g *GenerateClient) Humanize(ctx context.Context, text, personaID string, createSession bool) (*GenerateTextResponse, error) {
	request := &GenerateTextRequest{
		Text:          text,
		PersonaID:     personaID,
		CreateSession: createSession,
	}

	var response generateResponseBody

	if err := g.client.Post(ctx, "/query/humanize", request, &response); err != nil {
		return nil, fmt.Errorf("failed to humanize text: %w", err)
	}

	return &GenerateTextResponse{
		Text:      response.Content,
		PersonaID: personaID,
		SessionID: response.SessionID,
		ReviewURL: response.ReviewURL,
	}, nil
}

// TextVariants generates multiple draft variants (n 1..5) in a single
// non-streaming request. Each variant may carry a planned editorial angle when
// the backend produced an angle plan.
func (g *GenerateClient) TextVariants(ctx context.Context, request *GenerateTextRequest) (*GenerateDraftsResponse, error) {
	streaming := false
	request.Streaming = &streaming

	var response GenerateDraftsResponse
	if err := g.client.Post(ctx, "/query", request, &response); err != nil {
		return nil, fmt.Errorf("failed to generate drafts: %w", err)
	}
	return &response, nil
}

// SimpleText generates text with just a prompt and optional persona
func (g *GenerateClient) SimpleText(ctx context.Context, prompt string, personaID ...string) (string, error) {
	request := &GenerateTextRequest{
		Prompt: prompt,
	}

	if len(personaID) > 0 && personaID[0] != "" {
		request.PersonaID = personaID[0]
	}

	response, err := g.Text(ctx, request)
	if err != nil {
		return "", err
	}

	return response.Text, nil
}
