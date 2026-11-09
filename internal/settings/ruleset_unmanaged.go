package settings

import (
	"fmt"
	"reflect"
)

// RulesetPreviewApproval is acquired preservation metadata, never public intent.
// A nil value is permitted only for a new PR rule whose server default is unknown.
func RulesetPreviewApproval(policy *RulesetDeclaration) (*bool, bool) {
	if policy != nil && policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Type == "pull_request" && rule.Parameters != nil {
				return copyRuleValue(rule.Parameters.RequireExtraApprovalForUnattributedChanges), true
			}
		}
	}
	return nil, false
}

// PreserveRulesetUnmanagedParameters retains known acquired unowned values exactly.
// Removing their containers cannot preserve them and is refused before any write.
func PreserveRulesetUnmanagedParameters(before, after *RulesetDeclaration) error {
	value, present := RulesetPreviewApproval(before)
	if !present {
		return nil
	}
	if value == nil {
		return fmt.Errorf("missing acquired unowned PR additional-approval value")
	}
	if after != nil && after.Rules != nil {
		for i := range *after.Rules {
			rule := &(*after.Rules)[i]
			if rule.Type == "pull_request" && rule.Parameters != nil {
				rule.Parameters.RequireExtraApprovalForUnattributedChanges = copyRuleValue(value)
				return nil
			}
		}
	}
	return fmt.Errorf("cannot remove PR rule or Ruleset while preserving its unowned additional-approval value")
}

func validateRulesetUnmanagedChange(before, after *RulesetDeclaration) error {
	expected, present := RulesetPreviewApproval(before)
	if !present {
		return nil
	}
	actual, remains := RulesetPreviewApproval(after)
	if !remains {
		return fmt.Errorf("cannot remove PR rule or Ruleset while preserving its unowned additional-approval value")
	}
	if expected == nil || !reflect.DeepEqual(expected, actual) {
		return fmt.Errorf("unowned PR additional-approval value must be retained")
	}
	return nil
}
