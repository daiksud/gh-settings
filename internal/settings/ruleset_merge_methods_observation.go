package settings

import (
	"fmt"
	"slices"
)

// RulesetMergeMethodsNormalization retains the complete acquired list separately
// from its named fallback. Neither list supplies mutation values.
type RulesetMergeMethodsNormalization struct {
	Path     string
	RawValue []string
	Fallback []string
}

// IsZero distinguishes absent evidence from a present but invalid empty list.
func (record RulesetMergeMethodsNormalization) IsZero() bool {
	return record.Path == "" && record.RawValue == nil && record.Fallback == nil
}

func cloneRulesetMergeMethodsNormalization(record RulesetMergeMethodsNormalization) RulesetMergeMethodsNormalization {
	record.RawValue = slices.Clone(record.RawValue)
	record.Fallback = slices.Clone(record.Fallback)
	return record
}

// RawMergeMethods returns owned acquired methods, or nil for an absent PR rule.
// Complete normalized evidence retains the original strings and their order.
func (state RulesetState) RawMergeMethods() ([]string, error) {
	if err := state.validateNormalizationEvidence(); err != nil {
		return nil, err
	}
	methods := rulesetMergeMethods(state.Policy)
	if methods == nil {
		return nil, nil
	}
	if record := state.MergeMethodsNormalization; !record.IsZero() {
		return slices.Clone(record.RawValue), nil
	}
	return slices.Clone(*methods), nil
}

func rulesetMergeMethods(policy RulesetDeclaration) *[]string {
	if policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Type == "pull_request" && rule.Parameters != nil {
				return rule.Parameters.AllowedMergeMethods
			}
		}
	}
	return nil
}

func (state RulesetState) validateMergeMethodsNormalization() error {
	record := state.MergeMethodsNormalization
	if record.IsZero() {
		return nil
	}
	methods := rulesetMergeMethods(state.Policy)
	defaults := DefaultRulesetAllowedMergeMethods()
	if methods == nil || record.Path != RulesetMergeMethodsPath || !slices.Equal(record.Fallback, defaults) || len(record.RawValue) == 0 {
		return fmt.Errorf("invalid Ruleset merge methods observation provenance")
	}
	effective := slices.Clone(*methods)
	slices.Sort(effective)
	slices.Sort(defaults)
	if !slices.Equal(effective, defaults) {
		return fmt.Errorf("Ruleset merge methods evidence requires the effective default")
	}
	seen := make(map[string]bool, len(record.RawValue))
	unknown := false
	for _, method := range record.RawValue {
		if seen[method] {
			return fmt.Errorf("duplicate acquired Ruleset merge method")
		}
		seen[method] = true
		if _, supported := RulesetMergeMethodChoiceForValue(method); !supported {
			unknown = true
		}
	}
	if !unknown {
		return fmt.Errorf("Ruleset merge methods provenance requires an unsupported value")
	}
	return nil
}
