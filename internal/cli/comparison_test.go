package cli

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRenderComparison(t *testing.T) {
	tests := []struct {
		name       string
		comparison settings.Comparison
		want       string
	}{
		{
			name: "five fields",
			comparison: settings.Comparison{Managed: 5, Changes: []settings.Change{
				{Field: "repository.description", Before: "old", After: ""},
				{Field: "repository.homepage", Before: "old-url", After: ""},
				{Field: "repository.has_issues", Before: true, After: false},
				{Field: "repository.has_projects", Before: true, After: false},
				{Field: "repository.has_wiki", Before: true, After: false},
			}},
			want: "settings.repository.description: \"old\" → \"\"\nsettings.repository.homepage: \"old-url\" → \"\"\nsettings.repository.has_issues: true → false\nsettings.repository.has_projects: true → false\nsettings.repository.has_wiki: true → false\n",
		},
		{
			name: "escaped strings",
			comparison: settings.Comparison{Managed: 1, Changes: []settings.Change{
				{Field: "repository.description", Before: "old", After: "quote\"\n\t\\\x1b"},
			}},
			want: "settings.repository.description: \"old\" → \"quote\\\"\\n\\t\\\\\\x1b\"\n",
		},
		{
			name: "no differences", comparison: settings.Comparison{Managed: 2},
			want: "No differences for managed repository settings.\n",
		},
		{name: "unmanaged", want: "No managed repository settings.\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := string(RenderComparison(test.comparison))
			if got != test.want {
				t.Fatalf("output=%q, want %q", got, test.want)
			}
		})
	}
}

func TestRenderTopicsComparison(t *testing.T) {
	got := string(RenderComparison(settings.Comparison{Managed: 1, Changes: []settings.Change{{Field: "topics.names", Before: []string{"old"}, After: []string{}}}}))
	if got != "settings.topics.names: [\"old\"] → []\n" {
		t.Fatalf("output=%q", got)
	}
}
