package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDraftsRemaining(t *testing.T) {
	tests := []struct {
		name       string
		perMonth   int
		used       int
		wantRemain int
	}{
		{"normal", 100, 40, 60},
		{"exhausted", 100, 100, 0},
		{"over", 100, 130, 0},
		{"unlimited", -1, 500, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q Quota
			q.Entitlements.DraftsPerMonth = tt.perMonth
			q.CurrentUsage.MonthlyDraftCount = tt.used
			if got := q.DraftsRemaining(); got != tt.wantRemain {
				t.Errorf("DraftsRemaining() = %d, want %d", got, tt.wantRemain)
			}
		})
	}
}

func TestGetQuota(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/plan-entitlements" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"planId": "personal",
			"planStatus": "active",
			"entitlements": {"draftsPerMonth": 100, "apiAccess": true},
			"currentUsage": {"monthlyDraftCount": 40},
			"canCreate": {"draft": true}
		}`))
	}))
	defer server.Close()

	tc := NewToneCloneClientFromConfig(server.URL, "test_key", 0)
	q, err := tc.GetQuota(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.PlanID != "personal" || q.DraftsRemaining() != 60 || !q.CanCreate.Draft {
		t.Errorf("unexpected quota: %+v", q)
	}
}
