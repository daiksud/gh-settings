package settings

import "testing"

func TestRulesetRequiredApprovalsMappingSeparatesUIRangeFromNativeValidation(t *testing.T) {
	if RulesetRequiredApprovalsPath != "repository.settings.rulesets.rules.require_a_pull_request_before_merging.required_approvals" || DefaultRulesetRequiredApprovals != 0 {
		t.Fatal("wrong approvals path/default")
	}
	for _, test := range []struct {
		value     int
		supported bool
	}{{0, true}, {1, true}, {10, true}, {-1, false}, {11, false}, {100, false}} {
		if got := RulesetRequiredApprovalsSupported(test.value); got != test.supported {
			t.Fatal(test.value, got)
		}
	}
	policy := reviewPolicy()
	ruleOf(t, policy, "pull_request").RequiredApprovingReviewCount = paramInt(11)
	if err := ValidateRuleset(policy, true); err != nil {
		t.Fatal("native API-only count must remain valid", err)
	}
	ruleOf(t, policy, "pull_request").RequiredApprovingReviewCount = paramInt(-1)
	if err := ValidateRuleset(policy, true); err == nil {
		t.Fatal("native negative count must remain invalid")
	}
}
