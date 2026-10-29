package actions_test

import (
	"encoding/json"
	"html"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/actions"
)

func TestSummaryDisplaysCurrentPathsWithoutChangingPrivateProtocol(t *testing.T) {
	report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 3, Changes: []actions.PlanChange{
		{Field: "repository.description", Before: "<before>", After: "line\n<after>"},
		{Field: "repository.homepage", Before: "old-url", After: ""},
		{Field: "repository.has_issues", Before: true, After: false},
	}}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{`"field":"repository.description"`, `"field":"repository.homepage"`, `"field":"repository.has_issues"`} {
		if !strings.Contains(string(data), raw) {
			t.Fatalf("private protocol field changed: %s", data)
		}
	}
	content, err := actions.RenderSummary(report, summaryMetadata())
	if err != nil {
		t.Fatal(err)
	}
	want := "repository.about.description: \"<before>\" → \"line\\n<after>\"\nrepository.about.website: \"old-url\" → \"\"\nrepository.settings.general.features.issues: true → false\n"
	if !strings.Contains(html.UnescapeString(content), want) || strings.Contains(content, "<before>") || strings.Contains(content, "<after>") {
		t.Fatalf("Summary paths/escaping changed: %s", content)
	}
}
