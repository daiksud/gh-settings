package settings

import "testing"

func TestEnvironmentVariableSelectionIdentity(t *testing.T) {
	values, err := CanonicalEnvironmentVariableSelections([]EnvironmentVariableSelection{{Environment: "Production", Name: "display_name"}, {Environment: "Staging", Name: "DISPLAY_NAME"}})
	if err != nil || len(values) != 2 || values[0].Environment != "Production" || values[0].Name != "DISPLAY_NAME" {
		t.Fatalf("selection: %#v %v", values, err)
	}
	for _, values := range [][]EnvironmentVariableSelection{
		{{Environment: "Production", Name: "a"}, {Environment: "production", Name: "A"}},
		{{Environment: "", Name: "A"}}, {{Environment: " prod", Name: "A"}}, {{Environment: "prod", Name: "GITHUB_A"}},
	} {
		if _, err := CanonicalEnvironmentVariableSelections(values); err == nil {
			t.Fatalf("accepted %#v", values)
		}
	}
}

func TestEnvironmentVariableChangeAndPresence(t *testing.T) {
	value, err := NewVariableChange("value", nil, "non-sensitive")
	if err != nil {
		t.Fatal(err)
	}
	change, err := NewEnvironmentVariableChange("Production", value)
	if err != nil || change.Environment != "Production" || change.Change.Name() != "VALUE" {
		t.Fatalf("change %#v %v", change, err)
	}
	if _, err := NewEnvironmentVariableChange("", value); err == nil {
		t.Fatal("accepted missing environment")
	}
	if _, err := NewEnvironmentVariableChange("Production", VariableChange{}); err == nil {
		t.Fatal("accepted missing change")
	}
	if !(Comparison{EnvironmentVariables: []EnvironmentVariableChange{change}}).HasChanges() {
		t.Fatal("lost variable change")
	}
	variables, _ := NewVariableDeclarations([]RepositoryVariable{{Name: "A", Value: "value"}})
	if (Declaration{EnvironmentVariables: []EnvironmentVariableDeclaration{{Environment: "Production", Variables: variables}}}).Empty() {
		t.Fatal("lost declarations")
	}
	if !(Declaration{EnvironmentVariables: []EnvironmentVariableDeclaration{{Environment: "Production"}}}).Empty() {
		t.Fatal("empty values are not a no-op")
	}
}
