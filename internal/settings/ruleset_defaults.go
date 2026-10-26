package settings

// These are declaration defaults, not inferred server observations. Never call
// this on an API read: missing protection data must remain an error, not false.
func normalizeRuleObject(rule *RulesetRule) {
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
		defaultRuleValue(&p.RequireExtraApprovalForUnattributedChanges, false)
		defaultRuleValue(&p.RequireLastPushApproval, false)
		defaultRuleValue(&p.RequiredApprovingReviewCount, 0)
		defaultRuleValue(&p.RequiredReviewThreadResolution, false)
		defaultRuleValue(&p.RequiredReviewers, []RequiredReviewer{})
		if p.DismissalRestriction == nil {
			p.DismissalRestriction = &ReviewDismissalRestriction{}
		}
		defaultRuleValue(&p.DismissalRestriction.Enabled, false)
		defaultRuleValue(&p.DismissalRestriction.AllowedActors, []ReviewActor{})
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
