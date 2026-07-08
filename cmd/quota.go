package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/toneclone/cli/pkg/client"
)

var quotaCmd = &cobra.Command{
	Use:   "quota",
	Short: "Show the current plan, usage, and remaining drafts",
	Long: `Show the authenticated user's plan, current usage, and how many drafts
remain this period. Useful for agents to check before generating so they can
handle limits and paywalls gracefully.

Examples:
  toneclone quota
  toneclone quota --json`,
	RunE: runQuota,
}

func init() {
	rootCmd.AddCommand(quotaCmd)
}

func runQuota(cmd *cobra.Command, args []string) error {
	apiClient, err := newAPIClientWithTimeout(30)
	if err != nil {
		return err
	}

	q, err := apiClient.GetQuota(cmd.Context())
	if err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(quotaJSONView(q))
	}

	return outputQuotaText(q)
}

// quotaView is the stable, agent-friendly JSON shape for `quota --json`,
// exposing a computed draftsRemaining (-1 = unlimited).
type quotaView struct {
	Plan                    string          `json:"plan"`
	PlanStatus              string          `json:"planStatus"`
	DraftsPerMonth          int             `json:"draftsPerMonth"`
	DraftsUsed              int             `json:"draftsUsed"`
	DraftsRemaining         int             `json:"draftsRemaining"`
	CritiquePassesPerMonth  int             `json:"critiquePassesPerMonth"`
	CritiquePassesUsed      int             `json:"critiquePassesUsed"`
	CritiquePassesRemaining int             `json:"critiquePassesRemaining"`
	CanGenerate             bool            `json:"canGenerate"`
	CanCritique             bool            `json:"canCritique"`
	APIAccess               bool            `json:"apiAccess"`
	UsagePeriod             quotaPeriodView `json:"usagePeriod"`
}

type quotaPeriodView struct {
	Start *time.Time `json:"start"`
	End   *time.Time `json:"end"`
}

func quotaJSONView(q *client.Quota) quotaView {
	return quotaView{
		Plan:                    q.PlanID,
		PlanStatus:              q.PlanStatus,
		DraftsPerMonth:          q.Entitlements.DraftsPerMonth,
		DraftsUsed:              q.CurrentUsage.MonthlyDraftCount,
		DraftsRemaining:         q.DraftsRemaining(),
		CritiquePassesPerMonth:  q.Entitlements.CritiquePassesPerMonth,
		CritiquePassesUsed:      q.CurrentUsage.MonthlyCritiqueCount,
		CritiquePassesRemaining: q.CritiquePassesRemaining(),
		CanGenerate:             q.CanCreate.Draft,
		CanCritique:             q.CanCreate.Critique,
		APIAccess:               q.Entitlements.APIAccess,
		UsagePeriod: quotaPeriodView{
			Start: q.UsagePeriod.Start,
			End:   q.UsagePeriod.End,
		},
	}
}

func outputQuotaText(q *client.Quota) error {
	fmt.Printf("Plan:         %s (%s)\n", q.PlanID, q.PlanStatus)
	if q.Entitlements.DraftsPerMonth < 0 {
		fmt.Printf("Drafts:       %d used (unlimited)\n", q.CurrentUsage.MonthlyDraftCount)
	} else {
		fmt.Printf("Drafts:       %d/%d used, %d remaining\n",
			q.CurrentUsage.MonthlyDraftCount, q.Entitlements.DraftsPerMonth, q.DraftsRemaining())
	}
	if q.Entitlements.CritiquePassesPerMonth < 0 {
		fmt.Printf("Critique:     %d used (unlimited)\n", q.CurrentUsage.MonthlyCritiqueCount)
	} else if q.Entitlements.CritiquePassesPerMonth > 0 || q.CurrentUsage.MonthlyCritiqueCount > 0 {
		fmt.Printf("Critique:     %d/%d used, %d remaining\n",
			q.CurrentUsage.MonthlyCritiqueCount, q.Entitlements.CritiquePassesPerMonth, q.CritiquePassesRemaining())
	}
	fmt.Printf("Can generate: %v\n", q.CanCreate.Draft)
	fmt.Printf("Can critique: %v\n", q.CanCreate.Critique)
	fmt.Printf("API access:   %v\n", q.Entitlements.APIAccess)
	return nil
}
