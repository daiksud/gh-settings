package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const EnvironmentRefTypePath = "repository.settings.environments.deployment_branch_or_tag_rules.ref_type"
const DefaultEnvironmentRefType = uimodel.DefaultEnvironmentRefType

func EnvironmentRefTypeValueForChoice(choice uimodel.EnvironmentRefType) (string, bool) {
	switch choice {
	case uimodel.EnvironmentBranch:
		return "branch", true
	case uimodel.EnvironmentTag:
		return "tag", true
	default:
		return "", false
	}
}

func EnvironmentRefTypeChoiceForValue(value string) (uimodel.EnvironmentRefType, bool) {
	switch value {
	case "branch":
		return uimodel.EnvironmentBranch, true
	case "tag":
		return uimodel.EnvironmentTag, true
	default:
		return "", false
	}
}
