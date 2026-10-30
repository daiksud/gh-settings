package github

import (
	"cmp"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"slices"
)

func EnvironmentDeclaration(ui uimodel.Environment) (settings.EnvironmentDeclaration, error) {
	result := settings.EnvironmentDeclaration{Name: ui.Name}
	if rules := ui.DeploymentProtectionRules; rules != nil {
		result.WaitTimer = copyUIEnvironmentValue(rules.WaitTimer)
		result.PreventSelfReview = copyUIEnvironmentValue(rules.PreventSelfReview)
		result.CanAdminsBypass = copyUIEnvironmentValue(rules.AllowAdministratorsToBypassConfiguredProtectionRules)
		if rules.RequiredReviewers != nil {
			reviewers := make([]settings.EnvironmentReviewer, 0, len(*rules.RequiredReviewers))
			for _, reviewer := range *rules.RequiredReviewers {
				switch {
				case reviewer.Person != nil && reviewer.Team == nil:
					reviewers = append(reviewers, settings.EnvironmentReviewer{Type: "User", ID: *reviewer.Person})
				case reviewer.Team != nil && reviewer.Person == nil:
					reviewers = append(reviewers, settings.EnvironmentReviewer{Type: "Team", ID: *reviewer.Team})
				default:
					return settings.EnvironmentDeclaration{}, fmt.Errorf("environment reviewer requires exactly one person or team")
				}
			}
			result.Reviewers = &reviewers
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
func copyUIEnvironmentValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
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
	value, err := EnvironmentDeclarations(*entries)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// PreparedEnvironment retains resource IDs only as acquired adapter metadata.
type PreparedEnvironment struct {
	Environment uimodel.Environment
	ID          int64
	Patterns    []settings.EnvironmentPatternState
}

// EnvironmentPolicyUI projects complete acquired policy without identity metadata.
func EnvironmentPolicyUI(p settings.EnvironmentDeclaration) (uimodel.Environment, error) {
	if err := p.ValidateObservationPolicy(); err != nil {
		return uimodel.Environment{}, err
	}
	ui := uimodel.Environment{Name: p.Name, DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{
		WaitTimer: copyUIEnvironmentValue(p.WaitTimer), PreventSelfReview: copyUIEnvironmentValue(p.PreventSelfReview), AllowAdministratorsToBypassConfiguredProtectionRules: copyUIEnvironmentValue(p.CanAdminsBypass),
	}}
	reviewers := make([]uimodel.EnvironmentReviewer, 0, len(*p.Reviewers))
	for _, reviewer := range *p.Reviewers {
		switch reviewer.Type {
		case "User":
			reviewers = append(reviewers, uimodel.EnvironmentReviewer{Person: copyUIEnvironmentValue(&reviewer.ID)})
		case "Team":
			reviewers = append(reviewers, uimodel.EnvironmentReviewer{Team: copyUIEnvironmentValue(&reviewer.ID)})
		default:
			return uimodel.Environment{}, fmt.Errorf("environment reviewer is outside UI identities")
		}
	}
	ui.DeploymentProtectionRules.RequiredReviewers = &reviewers
	choice, supported := settings.EnvironmentDeploymentChoiceForPolicy(p.DeploymentBranchPolicy)
	if !supported {
		return uimodel.Environment{}, fmt.Errorf("environment branch policy is outside UI choices")
	}
	ui.DeploymentBranchesAndTags = &choice
	if choice == uimodel.SelectedBranchesAndTags {
		if p.DeploymentBranchPatterns == nil {
			return uimodel.Environment{}, fmt.Errorf("custom deployment patterns were not acquired")
		}
		patterns := make([]uimodel.EnvironmentDeploymentRule, 0, len(*p.DeploymentBranchPatterns))
		for _, pattern := range *p.DeploymentBranchPatterns {
			kind, supported := settings.EnvironmentRefTypeChoiceForValue(pattern.Type)
			if !supported {
				return uimodel.Environment{}, fmt.Errorf("environment pattern ref type is outside UI choices")
			}
			patterns = append(patterns, uimodel.EnvironmentDeploymentRule{RefType: kind, NamePattern: pattern.Name})
		}
		ui.DeploymentBranchOrTagRules = &patterns
	}
	return ui, nil
}

func EnvironmentsObservation(observation settings.EnvironmentObservation) ([]PreparedEnvironment, error) {
	states, err := observation.Values()
	if err != nil {
		return nil, err
	}
	result := make([]PreparedEnvironment, 0, len(states))
	for _, state := range states {
		p, err := state.EffectivePolicy()
		if err != nil {
			return nil, err
		}
		if mode := p.DeploymentBranchPolicy; mode != nil && mode.CustomBranchPolicies {
			if p.DeploymentBranchPatterns == nil {
				return nil, fmt.Errorf("custom deployment patterns were not acquired")
			}
			// Compare effective identities here; retain the acquired raw evidence
			// below for later reconciliation and verification of mutation intent.
			projected := slices.Clone(state.Patterns)
			for i := range projected {
				projected[i].RefTypeNormalization = settings.ObservationNormalization{}
			}
			changes, err := settings.ReconcileEnvironmentPatterns(projected, p.DeploymentBranchPatterns)
			if err != nil || len(changes) != 0 {
				return nil, fmt.Errorf("custom deployment pattern identity metadata is incomplete or inconsistent")
			}
		}
		ui, err := EnvironmentPolicyUI(p)
		if err != nil {
			return nil, err
		}
		result = append(result, PreparedEnvironment{Environment: ui, ID: state.ID, Patterns: slices.Clone(state.Patterns)})
	}
	slices.SortFunc(result, func(a, b PreparedEnvironment) int {
		return cmp.Compare(settings.EnvironmentIdentity(a.Environment.Name), settings.EnvironmentIdentity(b.Environment.Name))
	})
	return result, nil
}
