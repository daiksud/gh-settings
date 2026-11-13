package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
	"testing"
)

func TestCopilotReviewEffortExportProjection(t *testing.T) {
	for _, value := range []string{"default", "lite", "balanced", "max"} {
		for _, instructions := range []*bool{nil, new(bool)} {
			empty := ""
			topics := []string{}
			source := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &empty, Homepage: &empty}, Topics: settings.TopicsDeclaration{Names: &topics}, CopilotReviewEffort: &value, CopilotInstructions: instructions}
			ui, err := UIFromExportDeclaration(source)
			if err != nil {
				t.Fatal(err)
			}
			data, err := workspace.EncodeConfig(ui)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := workspace.DecodeConfig(data)
			if err != nil {
				t.Fatal(err)
			}
			d, err := DeclarationFromUI(decoded)
			if err != nil || d.CopilotReviewEffort == nil || *d.CopilotReviewEffort != value || (d.CopilotInstructions != nil) != (instructions != nil) {
				t.Fatalf("%s %v", data, err)
			}
			source.CopilotReviewEffort = new(string)
			*source.CopilotReviewEffort = "future"
			if _, err := UIFromExportDeclaration(source); err == nil {
				t.Fatal("exported unsupported effort")
			}
		}
	}
}
