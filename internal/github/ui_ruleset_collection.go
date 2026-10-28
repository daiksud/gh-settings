package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"slices"
	"strings"
)

// PreparedRulesets keeps acquisition diagnostics outside the UI policy entries.
type PreparedRulesets struct {
	Rulesets     []PreparedRuleset
	ExcludedPush int
}

func RulesetsDeclaration(entries *[]uimodel.Ruleset) (*settings.RulesetDeclarations, error) {
	if entries == nil {
		return nil, nil
	}
	policies := make([]settings.RulesetDeclaration, 0, len(*entries))
	for _, entry := range *entries {
		policy, err := RulesetDeclaration(entry)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	result, err := settings.NewRulesetDeclarations(policies)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func RulesetsObservation(observed settings.RulesetObservation) (PreparedRulesets, error) {
	entries, err := observed.Values()
	if err != nil {
		return PreparedRulesets{}, err
	}
	result := PreparedRulesets{Rulesets: make([]PreparedRuleset, 0, len(entries)), ExcludedPush: observed.ExcludedPush()}
	for _, entry := range entries {
		policy := entry.Policy
		policy.ID = &entry.ID
		ui, err := RulesetObservation(policy)
		if err != nil {
			return PreparedRulesets{}, err
		}
		result.Rulesets = append(result.Rulesets, ui)
	}
	slices.SortFunc(result.Rulesets, func(a, b PreparedRuleset) int { return strings.Compare(a.RulesetName, b.RulesetName) })
	return result, nil
}
