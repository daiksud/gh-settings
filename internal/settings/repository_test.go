package settings

import "testing"

func TestRepositoryDeclarationEmpty(t *testing.T) {
	empty, disabled := "", false
	for _, test := range []struct {
		name        string
		declaration RepositoryDeclaration
		empty       bool
	}{
		{name: "omitted fields", empty: true},
		{name: "empty description", declaration: RepositoryDeclaration{Description: &empty}},
		{name: "empty homepage", declaration: RepositoryDeclaration{Homepage: &empty}},
		{name: "disabled issues", declaration: RepositoryDeclaration{HasIssues: &disabled}},
		{name: "disabled projects", declaration: RepositoryDeclaration{HasProjects: &disabled}},
		{name: "disabled wiki", declaration: RepositoryDeclaration{HasWiki: &disabled}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.declaration.Empty(); got != test.empty {
				t.Fatalf("Empty()=%v, want %v", got, test.empty)
			}
		})
	}
}
