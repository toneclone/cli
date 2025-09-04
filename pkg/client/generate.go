package client

import (
	"context"
	"fmt"
)

// GenerateClient handles text generation API operations
type GenerateClient struct {
	client *Client
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
	var response struct {
		Content string `json:"content"`
		Done    bool   `json:"done"`
	}
	
	if err := g.client.Post(ctx, "/query", request, &response); err != nil {
		return nil, fmt.Errorf("failed to generate text: %w", err)
	}

	return &GenerateTextResponse{
		Text:      response.Content,
		PersonaID: request.PersonaID,
		ProfileID: request.ProfileID,
		Model:     request.Model,
	}, nil
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
