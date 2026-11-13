package settings

// These are declaration defaults, not inferred server observations. Never call
// this on an API read: unobserved protection must remain unknown, not false.
func normalizeRuleObject(rule *RulesetRule) {
	if rule.Type == "code_coverage" {
		if rule.Parameters == nil {
			rule.Parameters = &RuleParameters{}
		}
		defaultRuleValue(&rule.Parameters.MinimumCoverage, float64(0))
		defaultRuleValue(&rule.Parameters.MaxCoverageDrop, float64(0))
		return
	}
	if rule.Type == "update" {
		if rule.Parameters == nil {
			rule.Parameters = &RuleParameters{}
		}
		defaultRuleValue(&rule.Parameters.UpdateAllowsFetchAndMerge, false)
		return
	}
	if rule.Type == "copilot_code_review" {
		if rule.Parameters == nil {
			rule.Parameters = &RuleParameters{}
		}
		defaultRuleValue(&rule.Parameters.ReviewDraftPullRequests, false)
		defaultRuleValue(&rule.Parameters.ReviewOnPush, false)
		return
	}
	if rule.Type != "pull_request" && rule.Type != "required_status_checks" {
		return
	}
	if rule.Parameters == nil {
		rule.Parameters = &RuleParameters{}
	}
	p := rule.Parameters
	if rule.Type == "pull_request" {
		defaultRuleValue(&p.AllowedMergeMethods, []string{"merge", "squash", "rebase"})
		defaultRuleValue(&p.DismissStaleReviewsOnPush, false)
		defaultRuleValue(&p.RequireCodeOwnerReview, false)
		defaultRuleValue(&p.RequireExtraApprovalForUnattributedChanges, true)
		defaultRuleValue(&p.RequireLastPushApproval, false)
		defaultRuleValue(&p.RequiredApprovingReviewCount, 0)
		defaultRuleValue(&p.RequiredReviewThreadResolution, false)
		defaultRuleValue(&p.RequiredReviewers, []RequiredReviewer{})
	} else {
		defaultRuleValue(&p.RequiredStatusChecks, []RequiredStatusCheck{})
		defaultRuleValue(&p.StrictRequiredStatusChecksPolicy, false)
		defaultRuleValue(&p.DoNotEnforceOnCreate, false)
	}
}

func defaultRuleValue[T any](field **T, value T) {
	if *field == nil {
		*field = &value
	}
}
