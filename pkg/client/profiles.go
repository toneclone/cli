package client

import (
	"context"
	"fmt"
)

// ProfilesClient handles profile-related API operations
type ProfilesClient struct {
	client *Client
}

// NewProfilesClient creates a new profiles client
func NewProfilesClient(client *Client) *ProfilesClient {
	return &ProfilesClient{client: client}
}

// List retrieves all profiles for the authenticated user
func (p *ProfilesClient) List(ctx context.Context) ([]Profile, error) {
	var profiles []Profile
	err := p.client.Get(ctx, "/profiles", &profiles)
	if err != nil {
		// Handle empty response case (Content-Length: 0)
		if err.Error() == "unexpected end of JSON input" {
			return []Profile{}, nil
		}
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}
	return profiles, nil
}

// Get retrieves a specific profile by ID
func (p *ProfilesClient) Get(ctx context.Context, profileID string) (*Profile, error) {
	var profile Profile
	err := p.client.Get(ctx, fmt.Sprintf("/profiles/%s", profileID), &profile)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile %s: %w", profileID, err)
	}
	return &profile, nil
}

// Create creates a new profile
func (p *ProfilesClient) Create(ctx context.Context, profile *Profile) (*Profile, error) {
	var result Profile
	err := p.client.Post(ctx, "/profiles", profile, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}
	return &result, nil
}

// Update updates an existing profile
func (p *ProfilesClient) Update(ctx context.Context, profileID string, profile *Profile) (*Profile, error) {
	var result Profile
	err := p.client.Put(ctx, fmt.Sprintf("/profiles/%s", profileID), profile, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to update profile %s: %w", profileID, err)
	}
	return &result, nil
}

// Delete deletes a profile
func (p *ProfilesClient) Delete(ctx context.Context, profileID string) error {
	err := p.client.Delete(ctx, fmt.Sprintf("/profiles/%s", profileID))
	if err != nil {
		return fmt.Errorf("failed to delete profile %s: %w", profileID, err)
	}
	return nil
}

// AssociateWithPersona associates a profile with a persona
func (p *ProfilesClient) AssociateWithPersona(ctx context.Context, profileID, personaID string) error {
	body := map[string]interface{}{
		"profileIds": []string{profileID},
	}
	err := p.client.Post(ctx, fmt.Sprintf("/personas/%s/profiles", personaID), body, nil)
	if err != nil {
		return fmt.Errorf("failed to associate profile %s with persona %s: %w", profileID, personaID, err)
	}
	return nil
}

// DisassociateFromPersona disassociates a profile from a persona
func (p *ProfilesClient) DisassociateFromPersona(ctx context.Context, profileID, personaID string) error {
	body := map[string]interface{}{
		"profileIds": []string{profileID},
	}
	err := p.client.doRequest(ctx, "DELETE", fmt.Sprintf("/personas/%s/profiles", personaID), body, nil)
	if err != nil {
		return fmt.Errorf("failed to disassociate profile %s from persona %s: %w", profileID, personaID, err)
	}
	return nil
}

// GetPersonaProfiles retrieves all profiles associated with a persona
func (p *ProfilesClient) GetPersonaProfiles(ctx context.Context, personaID string) ([]Profile, error) {
	var profiles []Profile
	err := p.client.Get(ctx, fmt.Sprintf("/personas/%s/profiles", personaID), &profiles)
	if err != nil {
		return nil, fmt.Errorf("failed to get profiles for persona %s: %w", personaID, err)
	}
	return profiles, nil
}
