package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const PagesModePath = "repository.settings.pages.build_and_deployment.source"
const DefaultPagesModeChoice = uimodel.DefaultPagesSource

var pagesModeChoices = [...]struct {
	choice uimodel.PagesSourceChoice
	value  string
}{
	{uimodel.GitHubActions, "workflow"},
	{uimodel.DeployFromABranch, "legacy"},
}

func PagesModeValueForChoice(choice uimodel.PagesSourceChoice) (string, bool) {
	for _, option := range pagesModeChoices {
		if option.choice == choice {
			return option.value, true
		}
	}
	return "", false
}

func PagesModeChoiceForValue(value string) (uimodel.PagesSourceChoice, bool) {
	for _, option := range pagesModeChoices {
		if option.value == value {
			return option.choice, true
		}
	}
	return "", false
}
