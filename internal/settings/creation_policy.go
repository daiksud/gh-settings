package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

type CreationPolicyControl string

const (
	IssuePermissionsControl       CreationPolicyControl = "issue_permissions"
	PullRequestPermissionsControl CreationPolicyControl = "pull_request_permissions"
)

func (control CreationPolicyControl) Field() FieldID {
	switch control {
	case IssuePermissionsControl:
		return FieldID{path: "repository.issue_creation_policy"}
	case PullRequestPermissionsControl:
		return FieldID{path: "repository.pull_request_creation_policy"}
	default:
		return FieldID{}
	}
}

func creationPolicyControlForField(field FieldID) (CreationPolicyControl, bool) {
	for _, control := range [...]CreationPolicyControl{IssuePermissionsControl, PullRequestPermissionsControl} {
		if control.Field() == field {
			return control, true
		}
	}
	return "", false
}

func (control CreationPolicyControl) Path() string {
	switch control {
	case IssuePermissionsControl, PullRequestPermissionsControl:
		return "repository.settings.general.features." + string(control) + ".creation_allowed_by"
	default:
		return ""
	}
}

func (control CreationPolicyControl) DefaultChoice() (uimodel.CreationAllowedBy, bool) {
	switch control {
	case IssuePermissionsControl:
		return uimodel.DefaultIssueCreationAllowedBy, true
	case PullRequestPermissionsControl:
		return uimodel.DefaultPullRequestCreationAllowedBy, true
	default:
		return "", false
	}
}

var creationPolicies = [...]struct {
	choice uimodel.CreationAllowedBy
	value  string
}{
	{uimodel.AllUsers, "ALL"},
	{uimodel.CollaboratorsOnly, "COLLABORATORS_ONLY"},
}

func CreationPolicyValueForChoice(choice uimodel.CreationAllowedBy) (string, bool) {
	for _, option := range creationPolicies {
		if option.choice == choice {
			return option.value, true
		}
	}
	return "", false
}

func CreationPolicyChoiceForValue(value string) (uimodel.CreationAllowedBy, bool) {
	for _, option := range creationPolicies {
		if option.value == value {
			return option.choice, true
		}
	}
	return "", false
}
