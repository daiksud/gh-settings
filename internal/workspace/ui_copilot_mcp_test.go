package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"reflect"
	"strings"
	"testing"
)

func TestCopilotMCPYAMLRoundTrip(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		t.Run(value, func(t *testing.T) {
			data := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: {code_review: {allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: " + value + "}}}}\n")
			ui, err := DecodeConfig(data)
			if err != nil {
				t.Fatal(err)
			}
			desired, err := github.DeclarationFromUI(ui)
			if err != nil || desired.CopilotMCP == nil || *desired.CopilotMCP != (value == "true") || desired.Empty() {
				t.Fatal(desired, err)
			}
			encoded, err := EncodeConfig(ui)
			if err != nil {
				t.Fatal(err)
			}
			again, err := DecodeConfig(encoded)
			if err != nil || !reflect.DeepEqual(ui, again) {
				t.Fatalf("%s %v", encoded, err)
			}
		})
	}
}
func TestCopilotMCPYAMLBoundary(t *testing.T) {
	for _, settings := range []string{"{}", "{copilot: {}}", "{copilot: {code_review: {}}}"} {
		ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: " + settings + "}\n"))
		if err != nil {
			t.Fatal(err)
		}
		d, err := github.DeclarationFromUI(ui)
		if err != nil || !d.Empty() {
			t.Fatal(d, err)
		}
	}
	for _, body := range []string{"null", "false", "{code_review: null}", "{code_review: {allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: null}}", "{code_review: {allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: 'false'}}", "{code_review: {allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: 0}}", "{code_review: {review_effort: balanced}}", "{code_review: {mcp_enabled: false}}"} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: " + body + "}}\n")); err == nil {
			t.Fatal("accepted invalid/unadopted setting", body)
		}
	}
}

func TestCopilotMCPIndependentDeclarations(t *testing.T) {
	for _, siblings := range []string{"", ", use_custom_instructions: false", ", review_effort_level: balanced", ", use_custom_instructions: true, review_effort_level: default"} {
		data := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: {code_review: {allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests: false" + siblings + "}}}}\n")
		ui, err := DecodeConfig(data)
		if err != nil {
			t.Fatal(err)
		}
		d, err := github.DeclarationFromUI(ui)
		if err != nil || d.CopilotMCP == nil || *d.CopilotMCP {
			t.Fatal(d, err)
		}
		if (d.CopilotInstructions != nil) != strings.Contains(siblings, "use_custom_instructions") || (d.CopilotReviewEffort != nil) != strings.Contains(siblings, "review_effort_level") {
			t.Fatal("coupled controls", d)
		}
	}
}
