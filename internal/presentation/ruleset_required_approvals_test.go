package presentation

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

const approvalsDisplayPolicy = `{"id":7,"name":"policy","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"pull_request","parameters":{"allowed_merge_methods":["squash"],"dismiss_stale_reviews_on_push":false,"require_code_owner_review":false,"require_extra_approval_for_unattributed_changes":false,"require_last_push_approval":false,"required_approving_review_count":0,"required_review_thread_resolution":false,"required_reviewers":[]}}]}`

func TestRulesetRequiredApprovalsPresentationUsesTypedNumbersAndRemoval(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, raw := range []int{-1, 11} {
			for _, count := range []int{0, 1, 10, -1} {
				var before, after settings.RulesetDeclaration
				if err := json.Unmarshal([]byte(approvalsDisplayPolicy), &before); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(approvalsDisplayPolicy), &after); err != nil {
					t.Fatal(err)
				}
				records := settings.RulesetNormalizations{RequiredApprovals: settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: raw, Fallback: 0}}
				prefix, suffix := "", ""
				if both {
					disabled := "disabled"
					before.Enforcement, after.Enforcement = &disabled, &disabled
					records.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
					records.QualitySeverity = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "FUTURE_SEVERITY", Fallback: "errors"}
					severity := "errors"
					*before.Rules = append([]settings.RulesetRule{{Type: "code_quality", Parameters: &settings.RuleParameters{Severity: &severity}}}, *before.Rules...)
					*after.Rules = append([]settings.RulesetRule{{Type: "code_quality", Parameters: &settings.RuleParameters{Severity: &severity}}}, *after.Rules...)
					prefix = `Warning: repository.settings.rulesets.enforcement_status: observed "evaluate" is outside UI choices; using "disabled" (projection only). Ruleset 7 "policy".` + "\n" + `repository.settings.rulesets.enforcement_status (Ruleset 7 "policy"): "disabled" (normalized observation) → "disabled" (write required)` + "\n"
					suffix = "\n" + `Warning: repository.settings.rulesets.rules.require_code_quality_results.severity: observed "FUTURE_SEVERITY" is outside UI choices; using "errors" (projection only). Ruleset 7 "policy".` + "\n" + `repository.settings.rulesets.rules.require_code_quality_results.severity (Ruleset 7 "policy"): "errors" (normalized observation) → "errors" (write required)`
				}
				afterText := strconv.Itoa(count) + " (write required)"
				if count == -1 {
					rules := []settings.RulesetRule{}
					for _, rule := range *after.Rules {
						if rule.Type != "pull_request" {
							rules = append(rules, rule)
						}
					}
					after.Rules = &rules
					afterText = "inactive (rule removed; write required)"
				} else {
					for _, rule := range *after.Rules {
						if rule.Type == "pull_request" {
							rule.Parameters.RequiredApprovingReviewCount = &count
						}
					}
				}
				change, err := settings.NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records)
				if err != nil {
					t.Fatal(err)
				}
				warning := `Warning: repository.settings.rulesets.rules.require_a_pull_request_before_merging.required_approvals: observed ` + strconv.Itoa(raw) + ` is outside UI choices; using 0 (projection only). Ruleset 7 "policy".`
				decision := `repository.settings.rulesets.rules.require_a_pull_request_before_merging.required_approvals (Ruleset 7 "policy"): 0 (normalized observation) → ` + afterText
				whole := `repository.settings.rulesets: update "policy": ` + rulesetPolicy(&before) + ` → ` + rulesetPolicy(&after)
				want := prefix + warning + "\n" + decision + suffix + "\nEffective policy: " + whole
				if got := RulesetLine(change); got != want || strings.Count(got, "Effective policy:") != 1 {
					t.Fatal(got, want)
				}
				if count != 0 {
					ordinary, err := settings.NewRulesetDisplayChange("update", &before, &after)
					if err != nil || RulesetLine(ordinary) != whole {
						t.Fatal("ordinary output changed", err)
					}
				}
			}
		}
	}
}
