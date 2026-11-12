package settings

import "testing"

func TestVariablePruneInventoryAndPresence(t *testing.T) {
	empty, _ := NewVariableDeclarations(nil)
	if _, err := PrunedVariableChanges(VariableInventory{}, &empty); err == nil {
		t.Fatal("missing inventory accepted")
	}
	inventory, err := NewVariableInventory([]RepositoryVariable{{Name: "keep", Value: "same"}, {Name: "OLD", Value: ""}})
	if err != nil {
		t.Fatal(err)
	}
	if changes, err := PrunedVariableChanges(inventory, nil); err != nil || len(changes) != 0 {
		t.Fatal(changes, err)
	}
	desired, _ := NewVariableDeclarations([]RepositoryVariable{{Name: "KEEP", Value: "same"}})
	changes, err := PrunedVariableChanges(inventory, &desired)
	if err != nil || len(changes) != 1 || changes[0].Name() != "OLD" || changes[0].Operation() != "delete" {
		t.Fatal(changes, err)
	}
	before, _ := changes[0].Values()
	if before == nil || *before != "" {
		t.Fatal("empty existing value became absence")
	}
	if _, err := NewVariableInventory([]RepositoryVariable{{Name: "a"}, {Name: "A"}}); err == nil {
		t.Fatal("duplicate names accepted")
	}
	env, err := NewEnvironmentVariableInventory("prod/blue", 7, nil)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := env.EnvironmentID("PROD/blue"); err != nil || id != 7 {
		t.Fatal(id, err)
	}
	if _, err := env.EnvironmentID("other"); err == nil {
		t.Fatal("wrong parent accepted")
	}
	p, err := ParsePruneScopes("actions-variables,agents-variables,environment-variables")
	if err != nil || !p.Has("agents-variables") || !p.Manages(Declaration{EnvironmentVariables: []EnvironmentVariableDeclaration{{Environment: "prod", Variables: empty}}}) {
		t.Fatal(p, err)
	}
}
