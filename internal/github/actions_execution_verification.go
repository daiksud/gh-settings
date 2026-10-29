package github

import (
	"fmt"
	"reflect"

	"github.com/daiksud/gh-settings/internal/settings"
)

type actionsExecutionVerification struct {
	enabled  bool
	allowed  string
	sha      bool
	selected *settings.SelectedActionsState
}

func prepareActionsExecutionVerification(current settings.ActionsPolicyState, desired settings.ActionsPolicyDeclaration) (actionsExecutionVerification, error) {
	if _, err := settings.CompareActionsPolicy(current, desired); err != nil {
		return actionsExecutionVerification{}, err
	}
	raw, err := current.RawAllowedActions()
	if err != nil {
		return actionsExecutionVerification{}, err
	}
	expected := actionsExecutionVerification{enabled: current.Enabled, allowed: raw, sha: current.SHAPinningRequired}
	if desired.Enabled != nil {
		expected.enabled = *desired.Enabled
	}
	if desired.AllowedActions != nil {
		expected.allowed = *desired.AllowedActions
	}
	if desired.SHAPinningRequired != nil {
		expected.sha = *desired.SHAPinningRequired
	}
	if !desired.SelectedActions.Empty() {
		selected := settings.SelectedActionsState{}
		if current.Selected != nil {
			selected = *current.Selected
			selected.PatternsAllowed = append([]string{}, current.Selected.PatternsAllowed...)
		}
		if desired.SelectedActions.GitHubOwnedAllowed != nil {
			selected.GitHubOwnedAllowed = *desired.SelectedActions.GitHubOwnedAllowed
		}
		if desired.SelectedActions.VerifiedAllowed != nil {
			selected.VerifiedAllowed = *desired.SelectedActions.VerifiedAllowed
		}
		if desired.SelectedActions.PatternsAllowed != nil {
			selected.PatternsAllowed = append([]string{}, (*desired.SelectedActions.PatternsAllowed)...)
		}
		expected.selected = &selected
	}
	return expected, nil
}

func sameActionsExecutionVerification(expected actionsExecutionVerification, actual settings.ActionsPolicyState) (bool, error) {
	raw, err := actual.RawAllowedActions()
	if err != nil {
		return false, err
	}
	if actual.Enabled != expected.enabled || raw != expected.allowed || actual.SHAPinningRequired != expected.sha {
		return false, nil
	}
	if expected.selected != nil {
		if actual.Selected == nil {
			return false, fmt.Errorf("selected Actions readback was not acquired")
		}
		return reflect.DeepEqual(*actual.Selected, *expected.selected), nil
	}
	return true, nil
}
