package settings

// RulesetAdditionalApproval returns an isolated value for pre-update drift checks.
// Missing acquired values are rejected by complete Ruleset validation.
func RulesetAdditionalApproval(policy *RulesetDeclaration) (*bool, bool) {
	if policy != nil && policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Type == "pull_request" && rule.Parameters != nil {
				return copyRuleValue(rule.Parameters.RequireExtraApprovalForUnattributedChanges), true
			}
		}
	}
	return nil, false
}
