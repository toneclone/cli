package client

import (
	"context"
	"fmt"
	"time"
)

// Quota is the user's plan and usage snapshot from GET /user/plan-entitlements.
type Quota struct {
	PlanID       string `json:"planId"`
	PlanStatus   string `json:"planStatus"`
	Entitlements struct {
		DraftsPerMonth int  `json:"draftsPerMonth"`
		Personas       int  `json:"personas"`
		KnowledgeCards int  `json:"knowledgeCards"`
		APIAccess      bool `json:"apiAccess"`
	} `json:"entitlements"`
	CurrentUsage struct {
		MonthlyDraftCount  int `json:"monthlyDraftCount"`
		PersonaCount       int `json:"personaCount"`
		KnowledgeCardCount int `json:"knowledgeCardCount"`
	} `json:"currentUsage"`
	CanCreate struct {
		Draft         bool `json:"draft"`
		Persona       bool `json:"persona"`
		KnowledgeCard bool `json:"knowledgeCard"`
	} `json:"canCreate"`
	UsagePeriod struct {
		Start *time.Time `json:"start"`
		End   *time.Time `json:"end"`
	} `json:"usagePeriod"`
}

// DraftsRemaining returns how many drafts remain this period. A negative
// draftsPerMonth entitlement (unlimited) returns -1.
func (q *Quota) DraftsRemaining() int {
	if q.Entitlements.DraftsPerMonth < 0 {
		return -1
	}
	remaining := q.Entitlements.DraftsPerMonth - q.CurrentUsage.MonthlyDraftCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetQuota fetches the user's current plan entitlements and usage.
func (tc *ToneCloneClient) GetQuota(ctx context.Context) (*Quota, error) {
	var q Quota
	if err := tc.Get(ctx, "/user/plan-entitlements", &q); err != nil {
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}
	return &q, nil
}
