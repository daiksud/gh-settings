package github

import (
	"errors"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func apiReviews(enabled bool, count int, children bool) *settings.ClassicPullRequestReviews {
	v := &settings.ClassicPullRequestReviews{Enabled: &enabled}
	if enabled {
		approved := count > 0
		v.RequireApprovals = &approved
		v.DismissStalePullRequestApprovalsWhenNewCommitsArePushed = &children
		v.RequireReviewFromCodeOwners = &children
		v.RequireApprovalOfTheMostRecentReviewablePush = &children
		if approved {
			v.RequiredNumberOfApprovalsBeforeMerging = &count
		}
	}
	return v
}
func TestClassicReviewInputsAndPreservation(t *testing.T) {
	for _, mode := range []string{"enable-null", "enable-latent", "create-zero", "create-six", "create-disabled", "change-count", "zero-with-actors", "disable", "disable-hidden-reset", "omitted", "omitted-hidden-drift", "disabled-noop-actors", "missing-ack", "readback-drift", "companion-drift", "partial-create-error"} {
		t.Run(mode, func(t *testing.T) {
			rule := classicFixtureRule(7, "main")
			f := newClassicAPIFixture(t, rule)
			id, yes := int64(7), true
			d := settings.ClassicRule{ID: &id, BranchNamePattern: "main", RequireAPullRequestBeforeMerging: apiReviews(true, 0, false)}
			if strings.HasPrefix(mode, "enable") {
				rule["requiresApprovingReviews"] = false
				if mode == "enable-null" {
					rule["requiredApprovingReviewCount"] = nil
				}
			}
			if mode == "create-zero" || mode == "create-six" || mode == "create-disabled" || mode == "partial-create-error" {
				f.rules = nil
				d.ID = nil
			}
			if mode == "create-six" {
				d.RequireAPullRequestBeforeMerging = apiReviews(true, 6, false)
			}
			if mode == "change-count" {
				d.RequireAPullRequestBeforeMerging = apiReviews(true, 6, true)
			}
			if mode == "zero-with-actors" {
				d.RequireAPullRequestBeforeMerging = apiReviews(true, 0, true)
				rule["requiredApprovingReviewCount"] = 1
				rule["restrictsReviewDismissals"] = true
			}
			if strings.HasPrefix(mode, "disable") || mode == "create-disabled" {
				d.RequireAPullRequestBeforeMerging = apiReviews(false, 0, false)
			}
			if mode == "disabled-noop-actors" {
				rule["requiresApprovingReviews"] = false
				rule["requiredApprovingReviewCount"] = nil
				rule["restrictsReviewDismissals"] = true
			}
			if strings.HasPrefix(mode, "omitted") {
				d.RequireAPullRequestBeforeMerging = nil
				d.RequireSignedCommits = &yes
				rule["requiresApprovingReviews"] = false
				rule["requiredApprovingReviewCount"] = nil
			}
			actors := mode == "zero-with-actors" || mode == "disabled-noop-actors"
			f.hook = func(phase string, body map[string]any, _ map[string]any) {
				if actors && phase == "actors" {
					node := body["node"].(map[string]any)
					for _, key := range []string{"reviewDismissalAllowances", "bypassPullRequestAllowances"} {
						if _, ok := node[key]; ok {
							node[key] = classicFixturePage([]any{map[string]any{"actor": map[string]any{"__typename": "User", "id": "PRIVATE_U"}}}, 1, false, nil)
						}
					}
				}
			}
			selection := classicSelect(t, f, d)
			oldHook := f.hook
			f.hook = func(phase string, body map[string]any, vars map[string]any) {
				oldHook(phase, body, vars)
				if phase == "write" {
					mutation := "updateBranchProtectionRule"
					if d.ID == nil {
						mutation = "createBranchProtectionRule"
					}
					ack := body[mutation].(map[string]any)["branchProtectionRule"].(map[string]any)
					if mode == "missing-ack" {
						delete(ack, "requiredApprovingReviewCount")
					}
					if mode == "partial-create-error" {
						f.graphqlErrors = []any{map[string]any{"message": "partial resolver error"}}
					}
				}
				if phase == "node" && len(f.writes) > 0 {
					node := body["node"].(map[string]any)
					switch mode {
					case "readback-drift":
						node["requiresCodeOwnerReviews"] = true
					case "omitted-hidden-drift":
						node["requiredApprovingReviewCount"] = 0
					case "companion-drift":
						node["lockAllowsFetchAndMerge"] = true
					}
				}
			}
			if mode == "disable-hidden-reset" {
				f.mutate = func(rule map[string]any) {
					rule["requiredApprovingReviewCount"] = nil
					rule["dismissesStaleReviews"] = false
					rule["requiresCodeOwnerReviews"] = false
					rule["requireLastPushApproval"] = false
				}
			}
			result, err := f.client.Apply(classicRepo(), selection)
			bad := strings.Contains(mode, "drift") || mode == "missing-ack" || mode == "partial-create-error"
			if (err != nil) != bad {
				t.Fatal(result, err)
			}
			if mode == "disabled-noop-actors" {
				if result.Wrote || len(f.writes) != 0 {
					t.Fatal(result, f.writes)
				}
				return
			}
			want := map[string]any{"branchProtectionRuleId": "B_7"}
			switch {
			case mode == "create-disabled":
				want = map[string]any{"repositoryId": "R_current", "pattern": "main", "requiresApprovingReviews": false}
			case strings.HasPrefix(mode, "enable") || d.ID == nil:
				if d.ID == nil {
					want = map[string]any{"repositoryId": "R_current", "pattern": "main"}
				}
				count := float64(0)
				if mode == "create-six" {
					count = 6
				}
				want["requiresApprovingReviews"] = true
				want["requiredApprovingReviewCount"] = count
				want["dismissesStaleReviews"] = false
				want["requiresCodeOwnerReviews"] = false
				want["requireLastPushApproval"] = false
			case strings.HasPrefix(mode, "disable"):
				want["requiresApprovingReviews"] = false
			case strings.HasPrefix(mode, "omitted"):
				want["requiresCommitSignatures"] = true
			case mode == "change-count":
				want["requiredApprovingReviewCount"] = float64(6)
			case mode == "zero-with-actors":
				want["requiredApprovingReviewCount"] = float64(0)
			default:
				want["requiredApprovingReviewCount"] = float64(0)
				want["dismissesStaleReviews"] = false
				want["requiresCodeOwnerReviews"] = false
				want["requireLastPushApproval"] = false
			}
			if len(f.writes) != 1 || !reflect.DeepEqual(f.writes[0], want) {
				t.Fatal(f.writes, want)
			}
			if mode == "partial-create-error" && (result.ID <= 0 || result.NodeID == "" || result.Wrote) {
				t.Fatal("lost recovery", result)
			}
		})
	}
}

func TestClassicReviewDisableRepeatsGuardOnFreshState(t *testing.T) {
	for _, field := range []string{"restriction", "dismissal", "bypass"} {
		f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
		id := int64(7)
		selection := classicSelect(t, f, settings.ClassicRule{ID: &id, BranchNamePattern: "main", RequireAPullRequestBeforeMerging: apiReviews(false, 0, false)})
		f.hook = func(phase string, body map[string]any, _ map[string]any) {
			if phase == "node" && field == "restriction" {
				body["node"].(map[string]any)["restrictsReviewDismissals"] = true
			}
			if phase == "actors" {
				key := "reviewDismissalAllowances"
				if field == "bypass" {
					key = "bypassPullRequestAllowances"
				}
				node := body["node"].(map[string]any)
				if _, ok := node[key]; ok && field != "restriction" {
					node[key] = classicFixturePage([]any{map[string]any{"actor": map[string]any{"__typename": "User", "id": "PRIVATE_U"}}}, 1, false, nil)
				}
			}
		}
		result, err := f.client.Apply(classicRepo(), selection)
		var pre *classicPreWriteFailure
		if err == nil || !errors.As(err, &pre) || result.Wrote || len(f.writes) > 0 {
			t.Fatal(field, result, err)
		}
	}
}
