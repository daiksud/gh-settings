package github

import (
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// RulesetDeclaration translates UI intent before existing whole-object defaults.
func RulesetDeclaration(ui uimodel.Ruleset) (settings.RulesetDeclaration, error) {
	if (ui.TargetBranches == nil) == (ui.TargetTags == nil) {
		return settings.RulesetDeclaration{}, fmt.Errorf("ruleset requires exactly one of target_branches or target_tags")
	}
	if ui.EnforcementStatus == nil {
		return settings.RulesetDeclaration{}, fmt.Errorf("ruleset enforcement_status requires active or disabled")
	}
	enforcement, valid := settings.RulesetEnforcementValueForChoice(*ui.EnforcementStatus)
	if !valid {
		return settings.RulesetDeclaration{}, fmt.Errorf("ruleset enforcement_status requires active or disabled")
	}
	var patternLists []*[]string
	if ui.TargetBranches != nil {
		patternLists = []*[]string{ui.TargetBranches.IncludeByPattern, ui.TargetBranches.ExcludeByPattern}
	} else {
		patternLists = []*[]string{ui.TargetTags.IncludeByPattern, ui.TargetTags.ExcludeByPattern}
	}
	for _, patterns := range patternLists {
		if patterns != nil {
			for _, pattern := range *patterns {
				if strings.TrimSpace(pattern) == "" {
					return settings.RulesetDeclaration{}, fmt.Errorf("ruleset naming pattern must not be blank")
				}
			}
		}
	}
	target, prefix := "branch", "refs/heads/"
	include, exclude := []string{}, []string{}
	if b := ui.TargetBranches; b != nil {
		if b.IncludeDefaultBranch != nil && *b.IncludeDefaultBranch {
			include = append(include, "~DEFAULT_BRANCH")
		}
		if b.IncludeAllBranches != nil && *b.IncludeAllBranches {
			include = append(include, "~ALL")
		}
		include = appendUITargetPatterns(include, b.IncludeByPattern, prefix)
		exclude = appendUITargetPatterns(exclude, b.ExcludeByPattern, prefix)
	} else {
		target, prefix = "tag", "refs/tags/"
		if b := ui.TargetTags; b.IncludeAllTags != nil && *b.IncludeAllTags {
			include = append(include, "~ALL")
		}
		include = appendUITargetPatterns(include, ui.TargetTags.IncludeByPattern, prefix)
		exclude = appendUITargetPatterns(exclude, ui.TargetTags.ExcludeByPattern, prefix)
	}
	result := settings.RulesetDeclaration{ID: copyUIRulesetValue(ui.ID), Name: ui.RulesetName, Target: &target, Enforcement: &enforcement, Conditions: &settings.RulesetConditions{RefName: &settings.RefNameCondition{Include: &include, Exclude: &exclude}}}
	var err error
	result.BypassActors, err = declareUIRulesetBypass(ui.BypassList)
	if err != nil {
		return settings.RulesetDeclaration{}, err
	}
	result.Rules, err = declareUIRulesetRules(ui.Rules)
	if err != nil {
		return settings.RulesetDeclaration{}, err
	}
	if err := settings.ValidateRuleset(result, false); err != nil {
		return settings.RulesetDeclaration{}, err
	}
	return result, nil
}

func appendUITargetPatterns(result []string, patterns *[]string, prefix string) []string {
	if patterns != nil {
		for _, pattern := range *patterns {
			result = append(result, prefix+pattern)
		}
	}
	return result
}

// PreparedRuleset keeps acquired write-only state outside visible UI controls.
// The raw restriction is not authorized for export/comparison by this projection.
type PreparedRuleset struct {
	uimodel.Ruleset
	UnverifiedReviewDismissalRestriction *settings.ReviewDismissalRestriction
}

// RulesetObservation projects only complete, validated acquired API policies.
// Never run declaration-default normalization on an observation.
func RulesetObservation(policy settings.RulesetDeclaration) (PreparedRuleset, error) {
	if err := settings.ValidateRuleset(policy, true); err != nil {
		return PreparedRuleset{}, err
	}
	status, valid := settings.RulesetEnforcementChoiceForValue(*policy.Enforcement)
	if !valid {
		return PreparedRuleset{}, fmt.Errorf("ruleset enforcement_status is outside UI choices")
	}
	result := uimodel.Ruleset{ID: copyUIRulesetValue(policy.ID), RulesetName: policy.Name, EnforcementStatus: &status}
	var err error
	result.BypassList, err = observeUIRulesetBypass(*policy.BypassActors)
	if err != nil {
		return PreparedRuleset{}, err
	}
	result.Rules, err = observeUIRulesetRules(*policy.Rules)
	if err != nil {
		return PreparedRuleset{}, err
	}
	prefix := "refs/heads/"
	if *policy.Target == "tag" {
		prefix = "refs/tags/"
	}
	include, exclude := []string{}, []string{}
	defaultBranch, all := false, false
	for _, value := range *policy.Conditions.RefName.Include {
		switch {
		case value == "~ALL":
			all = true
		case value == "~DEFAULT_BRANCH" && *policy.Target == "branch":
			defaultBranch = true
		case strings.HasPrefix(value, prefix):
			pattern := strings.TrimPrefix(value, prefix)
			if strings.TrimSpace(pattern) == "" {
				return PreparedRuleset{}, fmt.Errorf("ruleset inclusion pattern has blank UI suffix")
			}
			include = append(include, pattern)
		default:
			return PreparedRuleset{}, fmt.Errorf("ruleset inclusion pattern is outside target UI choices")
		}
	}
	for _, value := range *policy.Conditions.RefName.Exclude {
		if !strings.HasPrefix(value, prefix) {
			return PreparedRuleset{}, fmt.Errorf("ruleset exclusion pattern is outside target UI choices")
		}
		pattern := strings.TrimPrefix(value, prefix)
		if strings.TrimSpace(pattern) == "" {
			return PreparedRuleset{}, fmt.Errorf("ruleset exclusion pattern has blank UI suffix")
		}
		exclude = append(exclude, pattern)
	}
	if *policy.Target == "branch" {
		result.TargetBranches = &uimodel.BranchTargets{IncludeDefaultBranch: &defaultBranch, IncludeAllBranches: &all, IncludeByPattern: &include, ExcludeByPattern: &exclude}
	} else {
		result.TargetTags = &uimodel.TagTargets{IncludeAllTags: &all, IncludeByPattern: &include, ExcludeByPattern: &exclude}
	}
	prepared := PreparedRuleset{Ruleset: result}
	for _, rule := range *policy.Rules {
		if rule.Type == "pull_request" && rule.Parameters != nil && rule.Parameters.DismissalRestriction != nil {
			raw := rule.Parameters.DismissalRestriction
			actors := append([]settings.ReviewActor{}, (*raw.AllowedActors)...)
			prepared.UnverifiedReviewDismissalRestriction = &settings.ReviewDismissalRestriction{Enabled: copyUIRulesetValue(raw.Enabled), AllowedActors: &actors}
		}
	}
	return prepared, nil
}

func copyUIRulesetValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
