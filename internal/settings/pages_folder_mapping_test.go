package settings

import (
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"testing"
)

func TestPagesFolderMappingKeepsClosedChoices(t *testing.T) {
	if PagesFolderPath != "repository.settings.pages.build_and_deployment.select_folder" || DefaultPagesFolderChoice != "root" {
		t.Fatal("folder identity/default changed")
	}
	for _, test := range []struct {
		choice uimodel.PagesFolderChoice
		value  string
	}{{"root", "/"}, {"docs", "/docs"}} {
		value, valid := PagesFolderValueForChoice(test.choice)
		if !valid || value != test.value {
			t.Fatal(test, value, valid)
		}
		choice, valid := PagesFolderChoiceForValue(test.value)
		if !valid || choice != test.choice {
			t.Fatal(test, choice, valid)
		}
	}
	for _, choice := range []uimodel.PagesFolderChoice{"", "/", "/docs", "Root", "future"} {
		if _, valid := PagesFolderValueForChoice(choice); valid {
			t.Fatal("unsupported UI choice", choice)
		}
	}
	for _, value := range []string{"", "root", "docs", "/other", " /docs"} {
		if _, valid := PagesFolderChoiceForValue(value); valid {
			t.Fatal("unsupported API path", value)
		}
	}
}
