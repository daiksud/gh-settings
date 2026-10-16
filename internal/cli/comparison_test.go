package cli

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRenderRepositoryComparison(t *testing.T) {
	tests := []struct {
		name       string
		comparison settings.RepositoryComparison
		want       string
	}{
		{
			name: "five fields",
			comparison: settings.RepositoryComparison{Managed: 5, Changes: []settings.RepositoryChange{
				{Field: "description", Before: "old", After: ""},
				{Field: "homepage", Before: "old-url", After: ""},
				{Field: "has_issues", Before: true, After: false},
				{Field: "has_projects", Before: true, After: false},
				{Field: "has_wiki", Before: true, After: false},
			}},
			want: "settings.repository.description: \"old\" → \"\"\nsettings.repository.homepage: \"old-url\" → \"\"\nsettings.repository.has_issues: true → false\nsettings.repository.has_projects: true → false\nsettings.repository.has_wiki: true → false\n",
		},
		{
			name: "escaped strings",
			comparison: settings.RepositoryComparison{Managed: 1, Changes: []settings.RepositoryChange{
				{Field: "description", Before: "old", After: "quote\"\n\t\\\x1b"},
			}},
			want: "settings.repository.description: \"old\" → \"quote\\\"\\n\\t\\\\\\x1b\"\n",
		},
		{
			name: "no differences", comparison: settings.RepositoryComparison{Managed: 2},
			want: "No differences for managed repository settings.\n",
		},
		{name: "unmanaged", want: "No managed repository settings.\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := string(RenderRepositoryComparison(test.comparison))
			if got != test.want {
				t.Fatalf("output=%q, want %q", got, test.want)
			}
		})
	}
}
