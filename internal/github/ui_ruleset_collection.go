package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

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
