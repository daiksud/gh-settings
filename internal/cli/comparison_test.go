package cli

import (
	"strings"
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
				comparisonChange(t, "repository.description", "old", ""),
				comparisonChange(t, "repository.homepage", "old-url", ""),
				comparisonChange(t, "repository.has_issues", true, false),
				comparisonChange(t, "repository.has_projects", true, false),
				comparisonChange(t, "repository.has_wiki", true, false),
			}},
			want: "settings.repository.description: \"old\" → \"\"\nsettings.repository.homepage: \"old-url\" → \"\"\nsettings.repository.has_issues: true → false\nsettings.repository.has_projects: true → false\nsettings.repository.has_wiki: true → false\n",
		},
		{
			name: "escaped strings",
			comparison: settings.Comparison{Managed: 1, Changes: []settings.Change{
				comparisonChange(t, "repository.description", "old", "quote\"\n\t\\\x1b"),
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
	got := string(RenderComparison(settings.Comparison{Managed: 1, Changes: []settings.Change{comparisonChange(t, "topics.names", []string{"old"}, []string{})}}))
	if got != "settings.topics.names: [\"old\"] → []\n" {
		t.Fatalf("output=%q", got)
	}
}

func TestRenderLabelsComparison(t *testing.T) {
	empty, color, id := "", "abcdef", int64(1)
	op := labelOperation(t)
	got := string(RenderComparison(settings.Comparison{Managed: 1, Labels: labelSequence(t,
		op(settings.NewLabelCreate(settings.LabelDeclaration{Name: "new", Color: &color})),
		op(settings.NewLabelUpdate(settings.LabelState{ID: 1, Name: "a", Color: "abcdef", Description: "old\ntext"}, settings.LabelDeclaration{ID: &id, Name: "a", Color: &color, Description: &empty})),
	)}))
	want := "settings.labels: create \"new\" color=\"abcdef\" description=\"\"\nsettings.labels: update \"a\" (id 1) description: \"old\\ntext\" → \"\"\n"
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestRenderMergeMethodComparison(t *testing.T) {
	got := string(RenderComparison(settings.Comparison{Managed: 3, Changes: []settings.Change{
		comparisonChange(t, "repository.allow_merge_commit", true, false),
		comparisonChange(t, "repository.allow_squash_merge", false, true),
		comparisonChange(t, "repository.allow_rebase_merge", true, false),
	}}))
	want := "settings.repository.allow_merge_commit: true → false\nsettings.repository.allow_squash_merge: false → true\nsettings.repository.allow_rebase_merge: true → false\n"
	if got != want {
		t.Fatalf("got=%q", got)
	}
}

func comparisonChange(t *testing.T, path string, before, after any) settings.Change {
	t.Helper()
	field, err := settings.ParseFieldID(path)
	if err != nil {
		t.Fatal(err)
	}
	var change settings.Change
	switch before := before.(type) {
	case bool:
		change, err = settings.NewBoolChange(field, before, after.(bool))
	case string:
		change, err = settings.NewStringChange(field, before, after.(string))
	case []string:
		if field != settings.TopicsField() {
			t.Fatal("topic values require the topics field")
		}
		change, err = settings.NewTopicsChange(before, after.([]string))
	default:
		t.Fatalf("unsupported comparison fixture type %T", before)
	}
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func labelOperation(t *testing.T) func(settings.LabelChange, error) settings.LabelChange {
	t.Helper()
	return func(change settings.LabelChange, err error) settings.LabelChange {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return change
	}
}

func labelSequence(t *testing.T, changes ...settings.LabelChange) settings.LabelChanges {
	t.Helper()
	result, err := settings.NewLabelChanges(changes)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestArchiveOrderIsExplicitInCLIPlan(t *testing.T) {
	for _, tc := range []struct {
		before, after bool
		want          string
	}{
		{true, false, "Lifecycle order: unarchive before other settings."},
		{false, true, "Lifecycle order: archive after other settings."},
	} {
		content := string(RenderComparison(settings.Comparison{Managed: 1, Changes: []settings.Change{comparisonChange(t, "repository.archived", tc.before, tc.after)}}))
		if !strings.Contains(content, tc.want) {
			t.Fatal(content, tc.want)
		}
	}
}
