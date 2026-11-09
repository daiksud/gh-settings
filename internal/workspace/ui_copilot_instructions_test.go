package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"reflect"
	"testing"
)

func TestCopilotInstructionsYAMLRoundTrip(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		t.Run(value, func(t *testing.T) {
			data := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: {code_review: {use_custom_instructions: " + value + "}}}}\n")
			ui, err := DecodeConfig(data)
			if err != nil {
				t.Fatal(err)
			}
			desired, err := github.DeclarationFromUI(ui)
			if err != nil || desired.CopilotInstructions == nil || *desired.CopilotInstructions != (value == "true") || desired.Empty() {
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
func TestCopilotInstructionsYAMLBoundary(t *testing.T) {
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
	for _, body := range []string{"null", "false", "{code_review: null}", "{code_review: {use_custom_instructions: null}}", "{code_review: {use_custom_instructions: 'false'}}", "{code_review: {use_custom_instructions: 0}}", "{code_review: {review_effort: balanced}}", "{code_review: {mcp_enabled: false}}"} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: " + body + "}}\n")); err == nil {
			t.Fatal("accepted invalid/unadopted setting", body)
		}
	}
}
