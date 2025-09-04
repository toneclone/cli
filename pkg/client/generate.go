package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
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
	// Make request directly to handle streaming
	resp, err := g.client.makeRequest(ctx, "POST", "/query", request)
	if err != nil {
		return nil, fmt.Errorf("failed to generate text: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Read the streaming response
	fullContent := strings.Builder{}
	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading stream: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip event lines
		if strings.HasPrefix(line, "event:") {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")

		// Parse the streaming chunk
		var chunk struct {
			Content string `json:"content"`
			Done    bool   `json:"done"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// Skip malformed chunks
			continue
		}

		if chunk.Done {
			// Final content - use this as the complete response
			fullContent.Reset()
			fullContent.WriteString(chunk.Content)
			break
		} else {
			// Incremental content
			fullContent.WriteString(chunk.Content)
		}
	}

	response := &GenerateTextResponse{
		Text:      fullContent.String(),
		PersonaID: request.PersonaID,
		ProfileID: request.ProfileID,
		Model:     request.Model,
	}

	return response, nil
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
