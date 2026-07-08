package client

import (
	"context"
	"fmt"
	"net/url"
)

// CritiqueClient handles editorial feedback API operations.
type CritiqueClient struct {
	client *Client
}

func NewCritiqueClient(client *Client) *CritiqueClient {
	return &CritiqueClient{client: client}
}

type SuggestionAnchor struct {
	Quote      string `json:"quote"`
	Before     string `json:"before,omitempty"`
	After      string `json:"after,omitempty"`
	Occurrence int    `json:"occurrence,omitempty"`
}

type CritiqueSuggestion struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Category    string            `json:"category"`
	Priority    int               `json:"priority"`
	Title       string            `json:"title"`
	Rationale   string            `json:"rationale"`
	Anchor      *SuggestionAnchor `json:"anchor,omitempty"`
	Replacement string            `json:"replacement,omitempty"`
	Instruction string            `json:"instruction,omitempty"`
	Scope       string            `json:"scope,omitempty"`
}

type CritiquePlan struct {
	CritiqueID      string               `json:"critiqueId,omitempty"`
	SessionID       string               `json:"sessionId,omitempty"`
	SourceVersionID string               `json:"sourceVersionId,omitempty"`
	DocumentHash    string               `json:"documentHash,omitempty"`
	CreatedAt       string               `json:"createdAt,omitempty"`
	StrategyNote    string               `json:"strategyNote"`
	Suggestions     []CritiqueSuggestion `json:"suggestions"`
}

type CritiqueRequest struct {
	PersonaID        string   `json:"personaId"`
	KnowledgeCardIDs []string `json:"knowledgeCardIds,omitempty"`
	SessionID        string   `json:"sessionId,omitempty"`
	SourceVersionID  string   `json:"sourceVersionId,omitempty"`
	Document         string   `json:"document"`
	Selection        string   `json:"selection,omitempty"`
	SelectionStart   *int     `json:"selectionStart,omitempty"`
	SelectionEnd     *int     `json:"selectionEnd,omitempty"`
	Context          string   `json:"context,omitempty"`
	Categories       []string `json:"categories,omitempty"`
	Intensity        string   `json:"intensity,omitempty"`
	MaxSuggestions   int      `json:"maxSuggestions,omitempty"`
}

type CritiqueHistoryResponse struct {
	Items []CritiquePlan `json:"items"`
}

func (c *CritiqueClient) Get(ctx context.Context, request *CritiqueRequest) (*CritiquePlan, error) {
	var response CritiquePlan
	if err := c.client.Post(ctx, "/query/critique", request, &response); err != nil {
		return nil, fmt.Errorf("failed to get editorial feedback: %w", err)
	}
	return &response, nil
}

func (c *CritiqueClient) History(ctx context.Context, sessionID string) (*CritiqueHistoryResponse, error) {
	var response CritiqueHistoryResponse
	path := "/query/critique/history?sessionId=" + url.QueryEscape(sessionID)
	if err := c.client.Get(ctx, path, &response); err != nil {
		return nil, fmt.Errorf("failed to list editorial feedback history: %w", err)
	}
	return &response, nil
}
