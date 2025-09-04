package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/toneclone/cli/pkg/client"
)

// validatePersona validates a persona by ID or name and returns the persona object
func validatePersona(ctx context.Context, apiClient *client.ToneCloneClient, personaInput string) (*client.Persona, error) {
	// First try to get by ID
	persona, err := apiClient.Personas.Get(ctx, personaInput)
	if err == nil {
		return persona, nil
	}

	// If that fails, try to find by name
	personas, err := apiClient.Personas.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list personas: %w", err)
	}

	// Look for exact name match
	for _, p := range personas {
		if strings.EqualFold(p.Name, personaInput) {
			return &p, nil
		}
	}

	// Look for partial name match
	var matches []client.Persona
	for _, p := range personas {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(personaInput)) {
			matches = append(matches, p)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("persona '%s' not found", personaInput)
	}

	if len(matches) > 1 {
		var names []string
		for _, p := range matches {
			names = append(names, fmt.Sprintf("'%s' (%s)", p.Name, p.PersonaID))
		}
		return nil, fmt.Errorf("multiple personas match '%s': %s", personaInput, strings.Join(names, ", "))
	}

	return &matches[0], nil
}

// validateProfile validates a profile by ID or name and returns the profile object
func validateProfile(ctx context.Context, apiClient *client.ToneCloneClient, profileInput string) (*client.Profile, error) {
	// First try to get by ID
	profile, err := apiClient.Profiles.Get(ctx, profileInput)
	if err == nil {
		return profile, nil
	}

	// If that fails, try to find by name
	profiles, err := apiClient.Profiles.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	// Look for exact name match
	for _, p := range profiles {
		if strings.EqualFold(p.Name, profileInput) {
			return &p, nil
		}
	}

	// Look for partial name match
	var matches []client.Profile
	for _, p := range profiles {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(profileInput)) {
			matches = append(matches, p)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("profile '%s' not found", profileInput)
	}

	if len(matches) > 1 {
		var names []string
		for _, p := range matches {
			names = append(names, fmt.Sprintf("'%s' (%s)", p.Name, p.ProfileID))
		}
		return nil, fmt.Errorf("multiple profiles match '%s': %s", profileInput, strings.Join(names, ", "))
	}

	return &matches[0], nil
}