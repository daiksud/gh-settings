package settings

import (
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"go.yaml.in/yaml/v4"
)

func TestArchiveFieldOmissionAndBooleanValidation(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	if (RepositoryDeclaration{}).Archived != nil || len(AllRepositoryFields()) != RepositoryFieldCount()-1 {
		t.Fatal("archive must be opt-in, not part of ordinary export")
	}
	for _, archived := range []bool{false, true} {
		d := RepositoryDeclaration{Archived: &archived}
		if d.Empty() || !d.BaseEmpty() || !ValidRepositoryFieldValue("repository.archived", archived) {
			t.Fatal(d)
		}
		field, err := ParseFieldID("repository.archived")
		if err != nil || field.Path() != "repository.archived" {
			t.Fatal(field, err)
		}
		got, err := NewBoolChange(field, !archived, archived)
		if err != nil {
			t.Fatal(err)
		}
		got.Visit(func(before, after bool) {
			if before == after || after != archived {
				t.Fatal(before, after)
			}
		}, func(string, string) { t.Fatal("string change") }, func([]string, []string) { t.Fatal("topic change") })
		state := ArchiveState{ID: 42, FullName: "owner/current", Archived: archived}
		if err := state.Validate(repo); err != nil {
			t.Fatal(err)
		}
		bytes, err := yaml.Marshal(map[string]any{"archived": archived})
		if err != nil || len(bytes) == 0 {
			t.Fatal(err)
		}
	}
	for _, bad := range []ArchiveState{{}, {ID: -1, FullName: "owner/current"}, {ID: 1, FullName: "wrong/repo"}, {ID: 1, FullName: "owner/current/extra"}} {
		if err := bad.Validate(repo); err == nil {
			t.Fatal("accepted", bad)
		}
	}
	if _, err := ParseFieldID("repository.archive"); err == nil {
		t.Fatal("accepted unsupported key")
	}
	if reflect.DeepEqual(RepositoryDeclaration{Archived: new(bool)}, RepositoryDeclaration{}) {
		t.Fatal("false lost")
	}
}
