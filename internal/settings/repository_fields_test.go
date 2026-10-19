package settings

import (
	"reflect"
	"testing"
)

func TestStringFieldDefinitions(t *testing.T) {
	for _, test := range []struct {
		name      string
		allowed   []string
		enum      bool
		want      func(string) RepositoryDeclaration
		wantError string
	}{
		{
			name: "repository.description", allowed: []string{"", "example value"},
			want: func(value string) RepositoryDeclaration { return RepositoryDeclaration{Description: &value} },
		},
		{
			name: "repository.homepage", allowed: []string{"", "example value"},
			want: func(value string) RepositoryDeclaration { return RepositoryDeclaration{Homepage: &value} },
		},
		{
			name: "repository.squash_merge_commit_title", allowed: []string{"PR_TITLE", "COMMIT_OR_PR_TITLE"}, enum: true,
			want:      func(value string) RepositoryDeclaration { return RepositoryDeclaration{SquashMergeCommitTitle: &value} },
			wantError: "squash_merge_commit_title must be PR_TITLE or COMMIT_OR_PR_TITLE",
		},
		{
			name: "repository.squash_merge_commit_message", allowed: []string{"PR_BODY", "COMMIT_MESSAGES", "BLANK"}, enum: true,
			want: func(value string) RepositoryDeclaration {
				return RepositoryDeclaration{SquashMergeCommitMessage: &value}
			},
			wantError: "squash_merge_commit_message must be PR_BODY, COMMIT_MESSAGES or BLANK",
		},
		{
			name: "repository.merge_commit_title", allowed: []string{"PR_TITLE", "MERGE_MESSAGE"}, enum: true,
			want:      func(value string) RepositoryDeclaration { return RepositoryDeclaration{MergeCommitTitle: &value} },
			wantError: "merge_commit_title must be PR_TITLE or MERGE_MESSAGE",
		},
		{
			name: "repository.merge_commit_message", allowed: []string{"PR_TITLE", "PR_BODY", "BLANK"}, enum: true,
			want:      func(value string) RepositoryDeclaration { return RepositoryDeclaration{MergeCommitMessage: &value} },
			wantError: "merge_commit_message must be PR_BODY, PR_TITLE or BLANK",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			set, enum, ok := RepositoryStringSetter(test.name)
			if !ok || set == nil || enum != test.enum {
				t.Fatalf("string lookup: found=%t enum=%t, want enum=%t", ok, enum, test.enum)
			}
			for _, value := range test.allowed {
				if !ValidRepositoryFieldValue(test.name, value) {
					t.Errorf("rejected %q", value)
				}
				var declaration RepositoryDeclaration
				if err := set(&declaration, value); err != nil || !reflect.DeepEqual(declaration, test.want(value)) {
					t.Errorf("value=%q assigned %+v, want %+v: %v", value, declaration, test.want(value), err)
				}
			}
			if !test.enum {
				return
			}
			for _, value := range []string{"", "UNKNOWN", "pr_title", " PR_TITLE"} {
				if ValidRepositoryFieldValue(test.name, value) {
					t.Errorf("accepted invalid enum %q", value)
				}
				disabled := false
				declaration := RepositoryDeclaration{HasIssues: &disabled}
				before := declaration
				err := set(&declaration, value)
				if err == nil || err.Error() != test.wantError || declaration != before {
					t.Errorf("invalid value=%q assigned %+v: %v; want unchanged declaration and %q", value, declaration, err, test.wantError)
				}
			}
		})
	}
}

func TestRepositoryStringSetterRejectsOtherNames(t *testing.T) {
	for _, name := range []string{"", "description", "repository.unknown", "repository.has_issues", "topics.names"} {
		if set, enum, ok := RepositoryStringSetter(name); ok || enum || set != nil {
			t.Errorf("unexpected string setter for %q", name)
		}
	}
}

func TestBooleanFieldDefinitions(t *testing.T) {
	for _, test := range []struct {
		name string
		want func(bool) RepositoryDeclaration
	}{
		{"repository.has_issues", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{HasIssues: &value} }},
		{"repository.has_projects", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{HasProjects: &value} }},
		{"repository.has_wiki", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{HasWiki: &value} }},
		{"repository.has_discussions", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{HasDiscussions: &value} }},
		{"repository.allow_merge_commit", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{AllowMergeCommit: &value} }},
		{"repository.allow_squash_merge", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{AllowSquashMerge: &value} }},
		{"repository.allow_rebase_merge", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{AllowRebaseMerge: &value} }},
		{"repository.delete_branch_on_merge", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{DeleteBranchOnMerge: &value} }},
		{"repository.allow_auto_merge", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{AllowAutoMerge: &value} }},
		{"repository.allow_update_branch", func(value bool) RepositoryDeclaration { return RepositoryDeclaration{AllowUpdateBranch: &value} }},
	} {
		for _, value := range []bool{false, true} {
			if !ValidRepositoryFieldValue(test.name, value) {
				t.Errorf("%s rejected native boolean %t", test.name, value)
			}
			set, ok := RepositoryBoolSetter(test.name)
			if !ok || set == nil {
				t.Fatalf("missing setter for %s", test.name)
			}
			var declaration RepositoryDeclaration
			set(&declaration, value)
			if !reflect.DeepEqual(declaration, test.want(value)) {
				t.Errorf("%s assigned %+v, want %+v", test.name, declaration, test.want(value))
			}
		}
	}
}

func TestRepositoryBoolSetterRejectsOtherNames(t *testing.T) {
	for _, name := range []string{"", "has_issues", "repository.unknown", "repository.description", "repository.merge_commit_title", "topics.names"} {
		if set, ok := RepositoryBoolSetter(name); ok || set != nil {
			t.Errorf("unexpected boolean setter for %q", name)
		}
	}
}

func TestRepositoryFieldValidationRejectsUnknownNamesAndTypes(t *testing.T) {
	for _, test := range []struct {
		name  string
		value any
	}{
		{"repository.unknown", false},
		{"repository.unknown", "value"},
		{"repository.has_issues", "false"},
		{"repository.has_issues", nil},
		{"repository.description", false},
		{"repository.description", nil},
		{"repository.description", float64(1)},
		{"repository.description", []string{}},
		{"topics.names", "value"},
	} {
		if ValidRepositoryFieldValue(test.name, test.value) {
			t.Errorf("%s accepted %T value %#v", test.name, test.value, test.value)
		}
	}
}
