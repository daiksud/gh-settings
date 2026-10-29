package settings

// RulesetRequiredApprovalsNormalization retains the acquired integer separately
// from the UI fallback. It is evidence, never desired policy or mutation input.
type RulesetRequiredApprovalsNormalization struct {
	Path     string `json:"path"`
	RawValue int    `json:"raw_value"`
	Fallback int    `json:"fallback"`
}

// RawRequiredApprovals returns an owned acquired count, or nil for an absent PR
// rule. Missing or incomplete acquisition is an error rather than absence.
func (state RulesetState) RawRequiredApprovals() (*int, error) {
	if err := state.validateNormalizationEvidence(); err != nil {
		return nil, err
	}
	count := rulesetRequiredApprovals(state.Policy)
	if count == nil {
		return nil, nil
	}
	value := *count
	if record := state.RequiredApprovalsNormalization; record != (RulesetRequiredApprovalsNormalization{}) {
		value = record.RawValue
	}
	return &value, nil
}

func rulesetRequiredApprovals(policy RulesetDeclaration) *int {
	if policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Type == "pull_request" && rule.Parameters != nil {
				return rule.Parameters.RequiredApprovingReviewCount
			}
		}
	}
	return nil
}
