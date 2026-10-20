package settings

import (
	"reflect"
	"testing"
)

func TestDefaultBranchNames(t *testing.T) {
	for _, name := range []string{"main", "release/v2", "feature/日本語", "@", "foo./bar", "refs/heads/main"} {
		if !ValidDefaultBranch(name) {
			t.Errorf("rejected %q", name)
		}
	}
	for _, name := range []string{"", "HEAD", "-main", "/main", "main/", "a//b", "a..b", "a@{b", ".hidden", "a/.hidden", "a.lock/b", "a/b.lock", "main.", "a b", "a\tb", "a\nb", "a\x7fb", "a~b", "a^b", "a:b", "a?b", "a*b", "a[b", "a\\b", string([]byte{255})} {
		if ValidDefaultBranch(name) {
			t.Errorf("accepted %q", name)
		}
	}
	set, enum, ok := RepositoryStringSetter("repository.default_branch")
	if !ok || enum {
		t.Fatalf("setter found=%t enum=%t", ok, enum)
	}
	var desired RepositoryDeclaration
	if err := set(&desired, "release/v2"); err != nil || desired.DefaultBranch == nil || *desired.DefaultBranch != "release/v2" {
		t.Fatalf("declaration=%+v error=%v", desired, err)
	}
	before := desired
	if err := set(&desired, ""); err == nil || desired != before {
		t.Fatalf("invalid value changed declaration: %+v error=%v", desired, err)
	}
	for _, value := range []any{nil, false, 1, "", "a..b"} {
		if ValidRepositoryFieldValue("repository.default_branch", value) {
			t.Errorf("accepted %#v", value)
		}
	}
}

func TestDefaultBranchComparison(t *testing.T) {
	branch := "release/v2"
	current := RepositoryState{DefaultBranch: "main"}
	if got := CompareRepository(current, RepositoryDeclaration{}); got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatalf("omitted field managed: %+v", got)
	}
	got := CompareRepository(current, RepositoryDeclaration{DefaultBranch: &branch})
	want := []Change{{field: FieldID{path: "repository.default_branch"}, values: stringPair{before: "main", after: "release/v2"}}}
	if got.Managed != 1 || !reflect.DeepEqual(got.Changes, want) {
		t.Fatalf("comparison=%+v", got)
	}
	got = CompareRepository(RepositoryState{DefaultBranch: branch}, RepositoryDeclaration{DefaultBranch: &branch})
	if got.Managed != 1 || len(got.Changes) != 0 {
		t.Fatalf("equal branch=%+v", got)
	}
}
