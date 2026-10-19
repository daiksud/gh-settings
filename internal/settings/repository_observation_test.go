package settings

import (
	"reflect"
	"testing"
)

func TestRepositoryObservationDeclaration(t *testing.T) {
	empty, description, homepage := "", "Example", "https://example.com"
	enabled, disabled := true, false
	for _, test := range []struct {
		name  string
		state RepositoryState
		want  RepositoryDeclaration
	}{
		{
			name: "explicit zero values",
			want: RepositoryDeclaration{
				Description: &empty, Homepage: &empty,
				HasIssues: &disabled, HasProjects: &disabled, HasWiki: &disabled, HasDiscussions: &disabled,
			},
		},
		{
			name: "mixed values",
			state: RepositoryState{
				Description: description, Homepage: homepage,
				HasIssues: true, HasProjects: false, HasWiki: true, HasDiscussions: true,
			},
			want: RepositoryDeclaration{
				Description: &description, Homepage: &homepage,
				HasIssues: &enabled, HasProjects: &disabled, HasWiki: &enabled, HasDiscussions: &enabled,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.state.Declaration()
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("declaration=%+v, want %+v", got, test.want)
			}
		})
	}
}
