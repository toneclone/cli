package client

import (
	"context"
	"fmt"
	"time"
)

// ToneCloneClient provides access to all ToneClone API resources
type ToneCloneClient struct {
	*Client

	// Resource clients
	Personas *PersonasClient
	Generate *GenerateClient
	Training *TrainingClient
	Profiles *ProfilesClient
}

// NewToneCloneClient creates a new ToneClone API client with all resource clients
func NewToneCloneClient(apiKey string, options ...ClientOption) *ToneCloneClient {
	baseClient := NewClient(apiKey, options...)

	return &ToneCloneClient{
		Client:   baseClient,
		Personas: NewPersonasClient(baseClient),
		Generate: NewGenerateClient(baseClient),
		Training: NewTrainingClient(baseClient),
		Profiles: NewProfilesClient(baseClient),
	}
}

// NewToneCloneClientFromConfig creates a client from configuration
func NewToneCloneClientFromConfig(baseURL, apiKey string, timeout time.Duration) *ToneCloneClient {
	options := []ClientOption{
		WithBaseURL(baseURL),
		WithTimeout(timeout),
	}

	return NewToneCloneClient(apiKey, options...)
}

// Ping tests the connection to the API
func (tc *ToneCloneClient) Ping(ctx context.Context) error {
	return tc.Health(ctx)
}

// WhoAmI returns information about the authenticated user
func (tc *ToneCloneClient) WhoAmI(ctx context.Context) (*User, error) {
	var user User
	err := tc.Get(ctx, "/user", &user)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	return &user, nil
}

// ValidateConnection validates that the client can connect and authenticate
func (tc *ToneCloneClient) ValidateConnection(ctx context.Context) error {
	// First check if the API is reachable
	if err := tc.Ping(ctx); err != nil {
		return fmt.Errorf("API unreachable: %w", err)
	}

	// Then check if authentication works
	if err := tc.ValidateAPIKey(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}
