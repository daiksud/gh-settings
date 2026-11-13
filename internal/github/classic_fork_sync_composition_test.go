package github

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestClassicForkSyncStatusReviewsAndDeploymentsCompose(t *testing.T) {
	for _, mode := range []string{"all-selected", "create-all", "all-disabled-sync-false", "status-selected-sync-drift", "sync-selected-status-drift", "sync-selected-review-drift", "sync-selected-deployment-drift", "status-selected-fork-drift", "nonfork-selection", "pre-app-invalid", "environment-fork-drift"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
			f.hook = func(_ string, body, _ map[string]any) { setClassicForkEvidence(body, mode != "nonfork-selection") }
			id, yes, no := int64(7), true, false
			desired := settings.ClassicRule{
				ID: &id, BranchNamePattern: "main", LockBranch: &yes, AllowForkSyncing: &yes,
				RequireAPullRequestBeforeMerging:         apiReviews(true, 3, false),
				RequireStatusChecksToPassBeforeMerging:   classicStatusDesired(false, settings.ClassicStatusCheck{Name: "build", AcceptUpdatesFrom: "APP_private"}),
				RequireDeploymentsToSucceedBeforeMerging: deploymentGroup(true, "staging"),
			}
			want := map[string]any{
				"branchProtectionRuleId": "B_7", "lockBranch": true, "lockAllowsFetchAndMerge": true,
				"requiredApprovingReviewCount": float64(3), "dismissesStaleReviews": false,
				"requiresCodeOwnerReviews": false, "requireLastPushApproval": false,
				"requiresStrictStatusChecks": false, "requiredDeploymentEnvironments": []any{"staging"},
			}
			switch mode {
			case "create-all":
				f.rules, desired.ID = nil, nil
				delete(want, "branchProtectionRuleId")
				want["repositoryId"], want["pattern"] = "R_current", "main"
				want["requiresApprovingReviews"], want["requiresStatusChecks"], want["requiresDeployments"] = true, true, true
				want["requiredStatusChecks"] = []any{map[string]any{"context": "build", "appId": "APP_private"}}
			case "all-disabled-sync-false":
				f.rules[0]["lockBranch"], f.rules[0]["lockAllowsFetchAndMerge"] = true, true
				desired.AllowForkSyncing = &no
				desired.RequireAPullRequestBeforeMerging = apiReviews(false, 0, false)
				desired.RequireStatusChecksToPassBeforeMerging = &settings.ClassicStatusChecks{Enabled: &no}
				desired.RequireDeploymentsToSucceedBeforeMerging = deploymentGroup(false)
				want = map[string]any{"branchProtectionRuleId": "B_7", "lockAllowsFetchAndMerge": false, "requiresApprovingReviews": false, "requiresStatusChecks": false, "requiresDeployments": false}
				f.mutate = func(rule map[string]any) {
					for _, key := range []string{"requiredApprovingReviewCount", "requiredStatusCheckContexts", "requiredStatusChecks", "requiredDeploymentEnvironments"} {
						rule[key] = nil
					}
				}
			case "status-selected-sync-drift", "status-selected-fork-drift":
				desired.AllowForkSyncing, desired.LockBranch = nil, nil
				delete(want, "lockBranch")
				delete(want, "lockAllowsFetchAndMerge")
			case "sync-selected-status-drift":
				desired.RequireStatusChecksToPassBeforeMerging = nil
				delete(want, "requiresStrictStatusChecks")
			case "sync-selected-review-drift":
				desired.RequireAPullRequestBeforeMerging = nil
				for _, key := range []string{"requiredApprovingReviewCount", "dismissesStaleReviews", "requiresCodeOwnerReviews", "requireLastPushApproval"} {
					delete(want, key)
				}
			case "sync-selected-deployment-drift":
				desired.RequireDeploymentsToSucceedBeforeMerging = nil
				delete(want, "requiredDeploymentEnvironments")
			}
			observed, err := f.client.Read(classicRepo())
			if err != nil {
				t.Fatal(err)
			}
			selections, err := settings.PrepareClassicRules(observed, []settings.ClassicRule{desired})
			if mode == "nonfork-selection" {
				if err == nil || len(f.writes) != 0 {
					t.Fatal("combined intent bypassed nonfork applicability", err, f.writes)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			f.calls = nil
			f.hook = func(phase string, body, _ map[string]any) {
				setClassicForkEvidence(body, !(mode == "environment-fork-drift" && phase == "environments"))
				if phase == "app" && mode == "pre-app-invalid" {
					body["node"] = map[string]any{"__typename": "User", "id": "APP_private"}
				}
				if phase != "node" || len(f.writes) == 0 {
					return
				}
				node := body["node"].(map[string]any)
				switch mode {
				case "status-selected-sync-drift":
					node["lockAllowsFetchAndMerge"] = true
				case "sync-selected-status-drift":
					node["requiresStrictStatusChecks"] = false
				case "sync-selected-review-drift":
					node["requiredApprovingReviewCount"] = float64(1)
				case "sync-selected-deployment-drift":
					node["requiredDeploymentEnvironments"] = []string{"staging"}
				case "status-selected-fork-drift":
					setClassicForkEvidence(body, false)
				}
			}
			result, err := f.client.Apply(classicRepo(), selections[0])
			prewrite := mode == "pre-app-invalid" || mode == "environment-fork-drift"
			bad := prewrite || strings.Contains(mode, "drift")
			var marker interface{ ReadFailedBeforeWrite() }
			if (err != nil) != bad || errors.As(err, &marker) != prewrite {
				t.Fatal(result, err)
			}
			if prewrite {
				if result.Wrote || len(f.writes) != 0 {
					t.Fatal("combined prerequisite failure wrote", result, f.writes)
				}
				return
			}
			if !result.Wrote || !reflect.DeepEqual(f.writes, []map[string]any{want}) {
				t.Fatal("combined payload differs", result, err, f.writes, want)
			}
			if bad {
				return
			}
			counts := map[string]int{}
			for _, phase := range f.calls {
				counts[phase]++
			}
			wantReferences := 2
			if mode == "all-disabled-sync-false" {
				wantReferences = 0
			}
			if counts["app"] != wantReferences || counts["environments"] != wantReferences {
				t.Fatal("independent reference guards lost", f.calls)
			}
			if mode == "create-all" {
				desired.ID = &result.ID
			}
			selection := classicSelect(t, f, desired)
			f.calls = nil
			result, err = f.client.Apply(classicRepo(), selection)
			if err != nil || result.Wrote || len(f.writes) != 1 {
				t.Fatal("matching repeated combined intent wrote", result, err, f.writes)
			}
			for _, phase := range f.calls {
				if phase == "app" || phase == "environments" {
					t.Fatal("matching state resolved write prerequisites", f.calls)
				}
			}
		})
	}
}
