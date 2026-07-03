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
	q.Entitlements.APIAccess = true
	q.CurrentUsage.MonthlyDraftCount = 40
	q.CanCreate.Draft = true

	v := quotaJSONView(&q)
	if v.Plan != "personal" || v.DraftsRemaining != 60 || !v.CanGenerate || !v.APIAccess {
		t.Errorf("unexpected view: %+v", v)
	}
}

func TestQuotaJSONViewUnlimited(t *testing.T) {
	var q client.Quota
	q.Entitlements.DraftsPerMonth = -1
	q.CurrentUsage.MonthlyDraftCount = 999
	if v := quotaJSONView(&q); v.DraftsRemaining != -1 {
		t.Errorf("expected -1 (unlimited), got %d", v.DraftsRemaining)
	}
}
