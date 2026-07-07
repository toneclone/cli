package cmd

import (
	"testing"

	"github.com/toneclone/cli/pkg/client"
)

func TestQuotaJSONView(t *testing.T) {
	var q client.Quota
	q.PlanID = "personal"
	q.PlanStatus = "active"
	q.Entitlements.DraftsPerMonth = 100
	q.Entitlements.CritiquePassesPerMonth = 10
	q.Entitlements.APIAccess = true
	q.CurrentUsage.MonthlyDraftCount = 40
	q.CurrentUsage.MonthlyCritiqueCount = 3
	q.CanCreate.Draft = true
	q.CanCreate.Critique = true

	v := quotaJSONView(&q)
	if v.Plan != "personal" || v.DraftsRemaining != 60 || !v.CanGenerate || !v.APIAccess {
		t.Errorf("unexpected draft view: %+v", v)
	}
	if v.CritiquePassesPerMonth != 10 || v.CritiquePassesUsed != 3 || v.CritiquePassesRemaining != 7 || !v.CanCritique {
		t.Errorf("unexpected view: %+v", v)
	}
}

func TestQuotaJSONViewUnlimited(t *testing.T) {
	var q client.Quota
	q.Entitlements.DraftsPerMonth = -1
	q.Entitlements.CritiquePassesPerMonth = -1
	q.CurrentUsage.MonthlyDraftCount = 999
	q.CurrentUsage.MonthlyCritiqueCount = 999
	if v := quotaJSONView(&q); v.DraftsRemaining != -1 || v.CritiquePassesRemaining != -1 {
		t.Errorf("expected -1 (unlimited), got %+v", v)
	}
}
