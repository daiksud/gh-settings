package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
	"testing"
)

func TestRepositoryVariablesExportRetainsActionsGeneral(t *testing.T) {
	// The base About/General values satisfy the ordinary export projection contract.
	empty := ""
	topics := []string{}
	variables, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "DISPLAY_NAME", Value: "日本語-v2"}, {Name: "SECOND", Value: "present"}})
	days := 7
	ui, err := UIFromExportDeclaration(settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &empty}, Topics: settings.TopicsDeclaration{Names: &topics}, Variables: &variables, Actions: settings.ActionsPolicyDeclaration{RetentionDays: &days}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := workspace.EncodeConfig(ui)
	if err != nil {
		t.Fatal(err)
	}
	again, err := workspace.DecodeConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	d, err := DeclarationFromUI(again)
	if err != nil || d.Variables == nil || d.Variables.Values()[0].Value != "日本語-v2" || d.Actions.RetentionDays == nil || *d.Actions.RetentionDays != 7 {
		t.Fatalf("%s %v", data, err)
	}
}
