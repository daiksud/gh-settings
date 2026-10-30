package settings

import "github.com/daiksud/gh-settings/internal/settings/uimodel"

const PagesFolderPath = "repository.settings.pages.build_and_deployment.select_folder"
const DefaultPagesFolderChoice = uimodel.DefaultPagesFolder

var pagesFolderChoices = [...]struct {
	choice uimodel.PagesFolderChoice
	value  string
}{
	{uimodel.PagesRoot, "/"},
	{uimodel.PagesDocs, "/docs"},
}

func PagesFolderValueForChoice(choice uimodel.PagesFolderChoice) (string, bool) {
	for _, option := range pagesFolderChoices {
		if option.choice == choice {
			return option.value, true
		}
	}
	return "", false
}

func PagesFolderChoiceForValue(value string) (uimodel.PagesFolderChoice, bool) {
	for _, option := range pagesFolderChoices {
		if option.value == value {
			return option.choice, true
		}
	}
	return "", false
}
