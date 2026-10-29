package cli

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestComparisonDisplaysCurrentAboutAndGeneralPaths(t *testing.T) {
	comparison := settings.Comparison{Managed: 3, Changes: []settings.Change{
		comparisonChange(t, "repository.description", "<before>", "line\n<after>"),
		comparisonChange(t, "repository.homepage", "old-url", ""),
		comparisonChange(t, "repository.has_issues", true, false),
	}}
	want := "repository.about.description: \"<before>\" → \"line\\n<after>\"\nrepository.about.website: \"old-url\" → \"\"\nrepository.settings.general.features.issues: true → false\n"
	if got := string(RenderComparison(comparison)); got != want {
		t.Fatalf("display=%q; want %q", got, want)
	}
	if comparison.Changes[0].Field().Path() != "repository.description" || comparison.Changes[2].Field().Path() != "repository.has_issues" {
		t.Fatal("presentation changed internal field identity")
	}
}
