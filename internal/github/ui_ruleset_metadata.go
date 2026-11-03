package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

var metadataRequirements = []struct {
	choice, operator string
	negate           bool
}{
	{"must_start_with_matching_pattern", "starts_with", false},
	{"must_not_start_with_matching_pattern", "starts_with", true},
	{"must_end_with_matching_pattern", "ends_with", false},
	{"must_not_end_with_matching_pattern", "ends_with", true},
	{"must_contain_matching_pattern", "contains", false},
	{"must_not_contain_matching_pattern", "contains", true},
	{"must_match_given_regex_pattern", "regex", false},
	{"must_not_match_given_regex_pattern", "regex", true},
}

func declareUIMetadataPattern(ui uimodel.MetadataPatternRule) (*settings.RuleParameters, error) {
	if ui.Pattern == nil {
		return nil, fmt.Errorf("metadata restriction requires pattern")
	}
	for _, requirement := range metadataRequirements {
		if requirement.choice != ui.Requirement {
			continue
		}
		p := &settings.RuleParameters{Name: copyUIRulesetValue(ui.Description), Operator: copyUIRulesetValue(&requirement.operator), Pattern: copyUIRulesetValue(ui.Pattern)}
		if requirement.negate {
			p.Negate = copyUIRulesetValue(&requirement.negate)
		}
		return p, nil
	}
	return nil, fmt.Errorf("unsupported metadata requirement %q", ui.Requirement)
}
func observeUIMetadataPattern(p *settings.RuleParameters) (*uimodel.MetadataPatternRule, error) {
	if p == nil || p.Operator == nil || p.Pattern == nil {
		return nil, fmt.Errorf("incomplete metadata restriction")
	}
	negate := p.Negate != nil && *p.Negate
	for _, requirement := range metadataRequirements {
		if requirement.operator == *p.Operator && requirement.negate == negate {
			return &uimodel.MetadataPatternRule{Description: copyUIRulesetValue(p.Name), Requirement: requirement.choice, Pattern: copyUIRulesetValue(p.Pattern)}, nil
		}
	}
	return nil, fmt.Errorf("unsupported metadata operator")
}
