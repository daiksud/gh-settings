package github

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestClassicForceReviewsAndDeploymentsCompose(t *testing.T) {
	for _, mode := range []string{"all-selected", "create", "all-disabled", "reviews-selected-force-drift", "force-selected-review-drift", "force-selected-deployment-drift", "fresh-review-guard"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
			id := int64(7)
			d := settings.ClassicRule{ID: &id, BranchNamePattern: "main", RequireAPullRequestBeforeMerging: apiReviews(true, 0, false), RequireDeploymentsToSucceedBeforeMerging: deploymentGroup(true, "staging"), AllowForcePushes: desiredForce("specified_actors", settings.ClassicForcePushActor{Type: "app", ID: "A_1"})}
			want := map[string]any{"branchProtectionRuleId": "B_7", "requiredApprovingReviewCount": float64(0), "dismissesStaleReviews": false, "requiresCodeOwnerReviews": false, "requireLastPushApproval": false, "requiredDeploymentEnvironments": []any{"staging"}, "allowsForcePushes": false, "bypassForcePushActorIds": []any{"A_1"}}
			switch mode {
			case "create":
				f.rules = nil
				d.ID = nil
				delete(want, "branchProtectionRuleId")
				want["repositoryId"] = "R_current"
				want["pattern"] = "main"
				want["requiresApprovingReviews"] = true
				want["requiresDeployments"] = true
			case "all-disabled":
				f.forceActors["B_7"] = []settings.ClassicActor{{Kind: "Team", ID: "T_old"}}
				d.RequireAPullRequestBeforeMerging = apiReviews(false, 0, false)
				d.RequireDeploymentsToSucceedBeforeMerging = deploymentGroup(false)
				d.AllowForcePushes = desiredForce("disabled")
				want = map[string]any{"branchProtectionRuleId": "B_7", "requiresApprovingReviews": false, "requiresDeployments": false, "allowsForcePushes": false, "bypassForcePushActorIds": []any{}}
			case "reviews-selected-force-drift":
				d.AllowForcePushes = nil
				d.RequireDeploymentsToSucceedBeforeMerging = nil
				f.forceActors["B_7"] = []settings.ClassicActor{{Kind: "Team", ID: "T_old"}}
				delete(want, "requiredDeploymentEnvironments")
				delete(want, "allowsForcePushes")
				delete(want, "bypassForcePushActorIds")
			case "force-selected-review-drift", "force-selected-deployment-drift":
				d.RequireAPullRequestBeforeMerging = nil
				d.RequireDeploymentsToSucceedBeforeMerging = nil
				want = map[string]any{"branchProtectionRuleId": "B_7", "allowsForcePushes": false, "bypassForcePushActorIds": []any{"A_1"}}
			case "fresh-review-guard":
				d.RequireAPullRequestBeforeMerging = apiReviews(false, 0, false)
			}
			selection := classicSelect(t, f, d)
			f.calls = nil
			f.hook = func(phase string, body, _ map[string]any) {
				if phase == "node" {
					n := body["node"].(map[string]any)
					if mode == "fresh-review-guard" {
						n["restrictsReviewDismissals"] = true
					}
					if len(f.writes) > 0 && mode == "force-selected-review-drift" {
						n["requiredApprovingReviewCount"] = 3
					}
					if len(f.writes) > 0 && mode == "force-selected-deployment-drift" {
						n["requiredDeploymentEnvironments"] = []string{"staging"}
					}
				}
				if phase == "actors" && len(f.writes) > 0 && mode == "reviews-selected-force-drift" {
					n := body["node"].(map[string]any)
					if _, ok := n["bypassForcePushAllowances"]; ok {
						n["bypassForcePushAllowances"] = classicFixturePage([]any{map[string]any{"actor": map[string]any{"__typename": "Team", "id": "T_new"}}}, 1, false, nil)
					}
				}
			}
			result, err := f.client.Apply(classicRepo(), selection)
			var pre *classicPreWriteFailure
			if mode == "fresh-review-guard" {
				if err == nil || !errors.As(err, &pre) || result.Wrote || len(f.writes) != 0 {
					t.Fatal("guard did not stop combined write", result, err, f.writes)
				}
				for _, call := range f.calls {
					if call == "force-identities" || call == "environments" {
						t.Fatal("guard did not stop prerequisites", f.calls)
					}
				}
				return
			}
			drift := strings.Contains(mode, "drift")
			if (err != nil) != drift || !result.Wrote || errors.As(err, &pre) {
				t.Fatal(result, err)
			}
			if len(f.writes) != 1 || !reflect.DeepEqual(f.writes[0], want) {
				t.Fatal("combined payload", f.writes, want)
			}
			if drift {
				return
			}
			if mode == "create" {
				if result.ID <= 0 || result.NodeID == "" {
					t.Fatal("missing confirmed creation identity", result)
				}
				d.ID = &result.ID
				selection = classicSelect(t, f, d)
			}
			f.calls = nil
			result, err = f.client.Apply(classicRepo(), selection)
			if err != nil || result.Wrote || len(f.writes) != 1 {
				t.Fatal("repeat wrote", result, err, f.writes)
			}
			for _, call := range f.calls {
				if call == "force-identities" || call == "environments" {
					t.Fatal("matching groups fetched prerequisites", f.calls)
				}
			}
		})
	}
}
