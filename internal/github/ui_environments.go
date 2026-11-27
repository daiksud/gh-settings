package github

import (
	"fmt"
	"slices"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func EnvironmentDeclaration(ui uimodel.Environment) (settings.EnvironmentDeclaration, error) {
	result := settings.EnvironmentDeclaration{Name: ui.Name}
	if rules := ui.DeploymentProtectionRules; rules != nil {
		if rules.EnableCustomRulesWithGitHubApps != nil {
			ids := slices.Clone(*rules.EnableCustomRulesWithGitHubApps)
			result.CustomRules = &ids
		}
		result.WaitTimer = copyUIValue(rules.WaitTimer)
		result.CanAdminsBypass = copyUIValue(rules.AllowAdministratorsToBypassConfiguredProtectionRules)
		if group := rules.RequiredReviewers; group != nil {
			result.PreventSelfReview = copyUIValue(group.PreventSelfReview)
			reviewers, err := environmentReviewerSelection(group.Reviewers)
			if err != nil {
				return settings.EnvironmentDeclaration{}, err
			}
			result.Reviewers = reviewers
		}
	}
	if ui.DeploymentBranchesAndTags != nil {
		result.DeploymentBranchPolicyDeclared = true
		policy, supported := settings.EnvironmentDeploymentPolicyForChoice(*ui.DeploymentBranchesAndTags)
		if !supported {
			return settings.EnvironmentDeclaration{}, fmt.Errorf("unsupported UI deployment branches and tags choice")
		}
		result.DeploymentBranchPolicy = policy
	}
	if ui.DeploymentBranchOrTagRules != nil {
		patterns := make([]settings.EnvironmentPattern, 0, len(*ui.DeploymentBranchOrTagRules))
		for _, rule := range *ui.DeploymentBranchOrTagRules {
			kind, supported := settings.EnvironmentRefTypeValueForChoice(rule.RefType)
			if !supported {
				return settings.EnvironmentDeclaration{}, fmt.Errorf("unsupported UI deployment rule ref type")
			}
			patterns = append(patterns, settings.EnvironmentPattern{Type: kind, Name: rule.NamePattern})
		}
		result.DeploymentBranchPatterns = &patterns
	}
	validated, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{result})
	if err != nil {
		return settings.EnvironmentDeclaration{}, err
	}
	return validated.Values()[0], nil
}

func environmentReviewerSelection(selection *[]uimodel.EnvironmentReviewer) (*[]settings.EnvironmentReviewer, error) {
	if selection == nil {
		return nil, nil
	}
	reviewers := make([]settings.EnvironmentReviewer, 0, len(*selection))
	for _, reviewer := range *selection {
		switch {
		case reviewer.Person != nil && reviewer.Team == nil:
			reviewers = append(reviewers, settings.EnvironmentReviewer{Type: "User", ID: *reviewer.Person})
		case reviewer.Team != nil && reviewer.Person == nil:
			reviewers = append(reviewers, settings.EnvironmentReviewer{Type: "Team", ID: *reviewer.Team})
		default:
			return nil, fmt.Errorf("environment reviewer requires exactly one person or team")
		}
	}
	return &reviewers, nil
}

func EnvironmentDeclarations(entries []uimodel.Environment) (settings.EnvironmentDeclarations, error) {
	policies := make([]settings.EnvironmentDeclaration, 0, len(entries))
	for _, entry := range entries {
		policy, err := EnvironmentDeclaration(entry)
		if err != nil {
			return settings.EnvironmentDeclarations{}, err
		}
		policies = append(policies, policy)
	}
	return settings.NewEnvironmentDeclarations(policies)
}

// EnvironmentCollectionDeclaration retains omitted vs explicitly empty selection.
func EnvironmentCollectionDeclaration(entries *[]uimodel.Environment) (*settings.EnvironmentDeclarations, error) {
	if entries == nil {
		return nil, nil
	}
	if _, err := EnvironmentDeclarations(*entries); err != nil {
		return nil, err
	}
	selected := make([]uimodel.Environment, 0, len(*entries))
	for _, entry := range *entries {
		if entry.Variables == nil || entry.DeploymentProtectionRules != nil || entry.DeploymentBranchesAndTags != nil || entry.DeploymentBranchOrTagRules != nil {
			selected = append(selected, entry)
		}
	}
	value, err := EnvironmentDeclarations(selected)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// EnvironmentPolicyUI projects complete acquired policy without identity metadata.
func EnvironmentPolicyUI(p settings.EnvironmentDeclaration) (uimodel.Environment, error) {
	if err := p.ValidateObservationPolicy(); err != nil {
		return uimodel.Environment{}, err
	}
	if p.DeploymentBranchPolicy != nil && p.DeploymentBranchPolicy.CustomBranchPolicies && p.DeploymentBranchPatterns == nil {
		return uimodel.Environment{}, fmt.Errorf("custom deployment patterns were not acquired")
	}
	return EnvironmentReportUI(p)
}

// EnvironmentReportUI projects an already validated complete observation or
// partial request without turning preserved companions into owned values.
func EnvironmentReportUI(p settings.EnvironmentDeclaration) (uimodel.Environment, error) {
	ui := uimodel.Environment{Name: p.Name}
	if p.CustomRules != nil || p.WaitTimer != nil || p.PreventSelfReview != nil || p.CanAdminsBypass != nil || p.Reviewers != nil {
		protection := &uimodel.EnvironmentDeploymentProtectionRules{
			WaitTimer: copyUIValue(p.WaitTimer),
			AllowAdministratorsToBypassConfiguredProtectionRules: copyUIValue(p.CanAdminsBypass),
		}
		if p.CustomRules != nil {
			ids := slices.Clone(*p.CustomRules)
			protection.EnableCustomRulesWithGitHubApps = &ids
		}
		if p.PreventSelfReview != nil || p.Reviewers != nil {
			protection.RequiredReviewers = &uimodel.EnvironmentRequiredReviewers{PreventSelfReview: copyUIValue(p.PreventSelfReview)}
		}
		if p.Reviewers != nil {
			reviewers := make([]uimodel.EnvironmentReviewer, 0, len(*p.Reviewers))
			for _, reviewer := range *p.Reviewers {
				switch reviewer.Type {
				case "User":
					reviewers = append(reviewers, uimodel.EnvironmentReviewer{Person: copyUIValue(&reviewer.ID)})
				case "Team":
					reviewers = append(reviewers, uimodel.EnvironmentReviewer{Team: copyUIValue(&reviewer.ID)})
				default:
					return uimodel.Environment{}, fmt.Errorf("environment reviewer is outside UI identities")
				}
			}
			protection.RequiredReviewers.Reviewers = &reviewers
		}
		ui.DeploymentProtectionRules = protection
	}
	if p.DeploymentBranchPolicyDeclared || p.DeploymentBranchPolicy != nil {
		choice, ok := settings.EnvironmentDeploymentChoiceForPolicy(p.DeploymentBranchPolicy)
		if !ok {
			return uimodel.Environment{}, fmt.Errorf("environment branch policy is outside UI choices")
		}
		ui.DeploymentBranchesAndTags = &choice
	}
	if p.DeploymentBranchPatterns != nil {
		patterns := make([]uimodel.EnvironmentDeploymentRule, 0, len(*p.DeploymentBranchPatterns))
		for _, pattern := range *p.DeploymentBranchPatterns {
			kind, ok := settings.EnvironmentRefTypeChoiceForValue(pattern.Type)
			if !ok {
				return uimodel.Environment{}, fmt.Errorf("environment pattern ref type is outside UI choices")
			}
			patterns = append(patterns, uimodel.EnvironmentDeploymentRule{RefType: kind, NamePattern: pattern.Name})
		}
		ui.DeploymentBranchOrTagRules = &patterns
	}
	return ui, nil
}

// EnvironmentVariablesDeclaration splits value selection from policy ownership.
func EnvironmentVariablesDeclaration(entries *[]uimodel.Environment) ([]settings.EnvironmentVariableDeclaration, error) {
	if entries == nil {
		return nil, nil
	}
	if _, err := EnvironmentDeclarations(*entries); err != nil {
		return nil, err
	}
	var result []settings.EnvironmentVariableDeclaration
	for _, entry := range *entries {
		variables, err := VariablesDeclaration(entry.Variables)
		if err != nil {
			return nil, fmt.Errorf("environment %q variables: %w", entry.Name, err)
		}
		if variables != nil {
			result = append(result, settings.EnvironmentVariableDeclaration{Environment: entry.Name, Variables: *variables})
		}
	}
	return result, nil
}
