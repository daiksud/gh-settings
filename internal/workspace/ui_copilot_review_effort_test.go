package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"reflect"
	"testing"
)

func TestCopilotReviewEffortYAML(t *testing.T) {
	for _, value := range []string{"default", "lite", "balanced"} {
		for _, sibling := range []string{"", ", use_custom_instructions: false"} {
			data := []byte("version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: {code_review: {review_effort_level: " + value + sibling + "}}}}\n")
			ui, err := DecodeConfig(data)
			if err != nil {
				t.Fatal(err)
			}
			d, err := github.DeclarationFromUI(ui)
			if err != nil || d.CopilotReviewEffort == nil || *d.CopilotReviewEffort != value || d.Empty() || (d.CopilotInstructions != nil) != (sibling != "") {
				t.Fatal(d, err)
			}
			encoded, err := EncodeConfig(ui)
			if err != nil {
				t.Fatal(err)
			}
			again, err := DecodeConfig(encoded)
			if err != nil || !reflect.DeepEqual(ui, again) {
				t.Fatalf("%s %v", encoded, err)
			}
		}
	}
	for _, value := range []string{"null", "false", "12", "[]", "{}"} {
		if _, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: {code_review: {review_effort_level: " + value + "}}}}\n")); err == nil {
			t.Fatal("accepted invalid type", value)
		}
	}
	for _, value := range []string{"max", "inherit", "Balanced", "''", "unknown"} {
		ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {copilot: {code_review: {review_effort_level: " + value + ", use_custom_instructions: false}}}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		if d, err := github.DeclarationFromUI(ui); err == nil || !d.Empty() {
			t.Fatal("accepted invalid effort or returned partial declaration", value, d, err)
		}
	}
}
