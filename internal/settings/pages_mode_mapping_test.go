package settings

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestPagesModeMappingUsesClosedLiteralChoices(t *testing.T) {
	if PagesModePath != "repository.settings.pages.build_and_deployment.source" || DefaultPagesModeChoice != uimodel.DeployFromABranch {
		t.Fatal(PagesModePath, DefaultPagesModeChoice)
	}
	for _, test := range []struct {
		choice uimodel.PagesSourceChoice
		value  string
	}{
		{"github_actions", "workflow"}, {"deploy_from_a_branch", "legacy"},
	} {
		value, valid := PagesModeValueForChoice(test.choice)
		if !valid || value != test.value {
			t.Fatal(test, value, valid)
		}
		choice, valid := PagesModeChoiceForValue(test.value)
		if !valid || choice != test.choice {
			t.Fatal(test, choice, valid)
		}
	}
	for _, raw := range []string{"", "future", "LEGACY", " legacy", "github_actions"} {
		if choice, valid := PagesModeChoiceForValue(raw); valid || choice != "" {
			t.Fatal(raw, choice, valid)
		}
	}
	for _, choice := range []uimodel.PagesSourceChoice{"", "future", "workflow", "GitHub_Actions"} {
		if value, valid := PagesModeValueForChoice(choice); valid || value != "" {
			t.Fatal(choice, value, valid)
		}
	}
}
