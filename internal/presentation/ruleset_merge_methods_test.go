package presentation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRulesetMergeMethodsPresentationKeepsRawOrderAndNamedChoiceOrder(t *testing.T) {
	for _, test := range []struct {
		name      string
		methods   []string
		afterText string
	}{
		{"fallback", []string{"rebase", "squash", "merge"}, `["merge","squash","rebase"] (write required)`},
		{"subset", []string{"rebase", "merge"}, `["merge","rebase"] (write required)`},
		{"remove", nil, `inactive (rule removed; write required)`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var before, after settings.RulesetDeclaration
			fixture := strings.Replace(approvalsDisplayPolicy, `["squash"]`, `["merge","rebase","squash"]`, 1)
			if err := json.Unmarshal([]byte(fixture), &before); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(fixture), &after); err != nil {
				t.Fatal(err)
			}
			if test.methods == nil {
				after.Rules = &[]settings.RulesetRule{}
			} else {
				(*after.Rules)[0].Parameters.AllowedMergeMethods = &test.methods
			}
			records := settings.RulesetNormalizations{
				RequiredApprovals: settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: 11, Fallback: 0},
				MergeMethods:      settings.RulesetMergeMethodsNormalization{Path: settings.RulesetMergeMethodsPath, RawValue: []string{"squash", "<future>\n", ""}, Fallback: []string{"merge", "squash", "rebase"}},
			}
			change, err := settings.NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records)
			if err != nil {
				t.Fatal(err)
			}
			got := RulesetLine(change)
			warning := `Warning: repository.settings.rulesets.rules.require_a_pull_request_before_merging.allowed_merge_methods: observed ["squash","\u003cfuture\u003e\n",""] is outside UI choices; using ["merge","squash","rebase"] (projection only). Ruleset 7 "policy".`
			decision := `repository.settings.rulesets.rules.require_a_pull_request_before_merging.allowed_merge_methods (Ruleset 7 "policy"): ["merge","squash","rebase"] (normalized observation) → ` + test.afterText
			if !strings.Contains(got, "required_approvals: observed 11") || !strings.Contains(got, "required_approvals (Ruleset 7") || !strings.Contains(got, warning+"\n"+decision+"\nEffective policy:") || strings.Count(got, "Effective policy:") != 1 || strings.Index(got, "required_approvals:") > strings.Index(got, "allowed_merge_methods:") {
				t.Fatal(got)
			}
			if strings.Contains(got, "<future>") {
				t.Fatal("unsafe raw display", got)
			}
			if test.name != "fallback" {
				ordinary, err := settings.NewRulesetDisplayChange("update", &before, &after)
				if err != nil {
					t.Fatal(err)
				}
				if plain := RulesetLine(ordinary); strings.Contains(plain, "projection only") || strings.Contains(plain, "Effective policy:") {
					t.Fatal(plain)
				}
			}
		})
	}
}
