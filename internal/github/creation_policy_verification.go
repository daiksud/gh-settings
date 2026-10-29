package github

import "github.com/daiksud/gh-settings/internal/settings"

type repositoryCreationObservations map[settings.CreationPolicyControl]preparedCreationPolicy

func (observed repositoryCreationObservations) rawValue(control settings.CreationPolicyControl) (string, bool) {
	value, present := observed[control]
	if !present || value.Value == nil || control.Field() == (settings.FieldID{}) {
		return "", false
	}
	if value.Normalization == (settings.ObservationNormalization{}) {
		_, valid := settings.CreationPolicyChoiceForValue(*value.Value)
		return *value.Value, valid
	}
	if _, err := settings.NewCreationPolicyChangeWithNormalization(control.Field(), *value.Value, *value.Value, value.Normalization); err != nil {
		return "", false
	}
	return value.Normalization.RawValue, true
}

func repositoryCreationPoliciesMatch(actual repositoryCreationObservations, desired settings.RepositoryDeclaration) bool {
	for _, entry := range []struct {
		control settings.CreationPolicyControl
		desired *string
	}{
		{settings.PullRequestPermissionsControl, desired.PullRequestCreationPolicy},
		{settings.IssuePermissionsControl, desired.IssueCreationPolicy},
	} {
		if entry.desired == nil {
			continue
		}
		if _, valid := settings.CreationPolicyChoiceForValue(*entry.desired); !valid {
			return false
		}
		raw, valid := actual.rawValue(entry.control)
		if !valid || raw != *entry.desired {
			return false
		}
	}
	return true
}
