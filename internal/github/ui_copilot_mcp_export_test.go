package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
	"testing"
)

func TestCopilotMCPExportProjection(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		empty := ""
		topics := []string{}
		days := 7
		ui, err := UIFromExportDeclaration(settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &empty}, Topics: settings.TopicsDeclaration{Names: &topics}, Actions: settings.ActionsPolicyDeclaration{RetentionDays: &days}, CopilotMCP: &enabled})
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
		if err != nil || d.CopilotMCP == nil || *d.CopilotMCP != enabled || d.Actions.RetentionDays == nil || *d.Actions.RetentionDays != 7 {
			t.Fatalf("%s %v", data, err)
		}
	}
}
