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

func TestRenderLabelsComparison(t *testing.T) {
	empty := ""
	got := string(RenderComparison(settings.Comparison{Managed: 1, Labels: []settings.LabelChange{
		{Operation: "create", Name: "new", After: &settings.LabelDeclaration{Name: "new"}},
		{Operation: "update", Name: "a", Before: &settings.LabelState{Name: "a", Color: "abcdef", Description: "old\ntext"}, After: &settings.LabelDeclaration{Name: "a", Description: &empty}},
		{Operation: "delete", Name: "old", Before: &settings.LabelState{Name: "old", Color: "abcdef"}},
	}}))
	want := "settings.labels: create \"new\"\nsettings.labels: update \"a\" description: \"old\\ntext\" → \"\"\nsettings.labels: delete \"old\" (removes issue/PR associations)\n"
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestRenderMergeMethodComparison(t *testing.T) {
	got := string(RenderComparison(settings.Comparison{Managed: 3, Changes: []settings.Change{
		{Field: "repository.allow_merge_commit", Before: true, After: false},
		{Field: "repository.allow_squash_merge", Before: false, After: true},
		{Field: "repository.allow_rebase_merge", Before: true, After: false},
	}}))
	want := "settings.repository.allow_merge_commit: true → false\nsettings.repository.allow_squash_merge: false → true\nsettings.repository.allow_rebase_merge: true → false\n"
	if got != want {
		t.Fatalf("got=%q", got)
	}
}
