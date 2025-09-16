package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/toneclone/cli/pkg/client"
)

// validatePersona validates a persona by ID or name and returns the persona object
func validatePersona(ctx context.Context, apiClient *client.ToneCloneClient, personaInput string) (*client.Persona, error) {
	// First try to get by ID (this will work for both user and built-in personas)
	persona, err := apiClient.Personas.Get(ctx, personaInput)
	if err == nil {
		return persona, nil
	}

	// If that fails, try to find by name in both user and built-in personas
	userPersonas, err := apiClient.Personas.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list user personas: %w", err)
	}

	builtInPersonas, err := apiClient.Personas.ListBuiltIn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list built-in personas: %w", err)
	}

	// Mark built-in personas and combine lists
	for i := range builtInPersonas {
		builtInPersonas[i].IsBuiltIn = true
	}
	allPersonas := append(userPersonas, builtInPersonas...)

	// Look for exact ID match first
	for _, p := range allPersonas {
		if p.PersonaID == personaInput {
			return &p, nil
		}
	}

	// Look for exact name match
	for _, p := range allPersonas {
		if strings.EqualFold(p.Name, personaInput) {
			return &p, nil
		}
	}

	// Look for partial name match
	var matches []client.Persona
	for _, p := range allPersonas {
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
			source := "user"
			if p.IsBuiltIn {
				source = "built-in"
			}
			names = append(names, fmt.Sprintf("'%s' (%s, %s)", p.Name, p.PersonaID, source))
		}
		return nil, fmt.Errorf("multiple personas match '%s': %s", personaInput, strings.Join(names, ", "))
	}

	return &matches[0], nil
}

// validateKnowledgeCard validates a knowledge card by ID or name and returns the card object
func validateKnowledgeCard(ctx context.Context, apiClient *client.ToneCloneClient, knowledgeInput string) (*client.KnowledgeCard, error) {
	// First try to get by ID
	card, err := apiClient.Knowledge.Get(ctx, knowledgeInput)
	if err == nil {
		return card, nil
	}

	// If that fails, try to find by name
	cards, err := apiClient.Knowledge.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list knowledge cards: %w", err)
	}

	// Look for exact name match
	for _, c := range cards {
		if strings.EqualFold(c.Name, knowledgeInput) {
			return &c, nil
		}
	}

	// Look for partial name match
	var matches []client.KnowledgeCard
	for _, c := range cards {
		if strings.Contains(strings.ToLower(c.Name), strings.ToLower(knowledgeInput)) {
			matches = append(matches, c)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("knowledge card '%s' not found", knowledgeInput)
	}

	if len(matches) > 1 {
		var names []string
		for _, c := range matches {
			names = append(names, fmt.Sprintf("'%s' (%s)", c.Name, c.KnowledgeCardID))
		}
		return nil, fmt.Errorf("multiple knowledge cards match '%s': %s", knowledgeInput, strings.Join(names, ", "))
	}

	return &matches[0], nil
}
