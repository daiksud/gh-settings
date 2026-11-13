package settings

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"testing"
)

func TestEnvironmentPruneAndAllSelection(t *testing.T) {
	for _, scope := range []string{"environments", "all"} {
		p, err := ParsePruneScopes(scope)
		if err != nil || !p.Has("environments") {
			t.Fatalf("%s must select Environment pruning: %v", scope, err)
		}
		if scope == "all" {
			for _, member := range []string{"labels", "rulesets", "actions-variables", "agents-variables", "environment-variables", "actions-policies", "environments"} {
				if !p.Has(member) {
					t.Fatalf("all omitted %s", member)
				}
			}
		}
		empty, _ := NewEnvironmentDeclarations(nil)
		if p.Manages(Declaration{}) || !p.Manages(Declaration{Environments: &empty}) {
			t.Fatal("Environment collection presence was lost")
		}
	}
}

func TestEnvironmentPruneRetainsPolicyAndVariableOnlyParents(t *testing.T) {
	policies, _ := NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: "Keep"}})
	empty, _ := NewVariableDeclarations(nil)
	d := Declaration{Environments: &policies, EnvironmentVariables: []EnvironmentVariableDeclaration{{Environment: "keep", Variables: empty}, {Environment: "variables-only", Variables: empty}}}
	got := EnvironmentRetainedNames(d)
	if len(got) != 2 || EnvironmentIdentity(got[0]) != EnvironmentIdentity("keep") || got[1] != "variables-only" {
		t.Fatal(got)
	}
	for _, token := range []string{"all,labels", "labels,all", "all,all", "environments,environments", " environments", "environments,"} {
		if _, err := ParsePruneScopes(token); err == nil {
			t.Fatal(token)
		}
	}
}

func TestEnvironmentDeleteRequiresCompleteBeforeOnly(t *testing.T) {
	timer, flag, reviewers := 0, false, []EnvironmentReviewer{}
	before := EnvironmentDeclaration{Name: "production", WaitTimer: &timer, PreventSelfReview: &flag, CanAdminsBypass: &flag, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}
	change, err := NewEnvironmentChange("delete", &before, nil)
	if err != nil || change.Before() == nil || change.After() != nil || change.Name() != "production" {
		t.Fatal(change, err)
	}
	for _, test := range []struct{ before, after *EnvironmentDeclaration }{{nil, nil}, {&before, &before}, {&EnvironmentDeclaration{Name: "production"}, nil}} {
		if _, err := NewEnvironmentChange("delete", test.before, test.after); err == nil {
			t.Fatal("incomplete deletion accepted", test)
		}
	}
	*change.Before().WaitTimer = 5
	if *change.Before().WaitTimer != 0 {
		t.Fatal("delete before state mutable")
	}
}

func TestEnvironmentPruneSnapshotValidationAndOwnership(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	target := ArchiveState{ID: 71, FullName: "owner/current"}
	timer, flag, reviewers := 0, false, []EnvironmentReviewer{}
	policy := EnvironmentDeclaration{Name: "old", WaitTimer: &timer, PreventSelfReview: &flag, CanAdminsBypass: &flag, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}
	state := EnvironmentState{CustomRules: []EnvironmentCustomRule{}, ID: 9, Policy: policy}
	variables, _ := NewVariableInventory([]RepositoryVariable{{Name: "KEY", Value: ""}})
	secrets := []EnvironmentSecretMetadata{{Name: "SECRET", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z"}}
	candidate, err := NewEnvironmentPruneCandidate(repo, target, state, variables, secrets, true)
	if err != nil {
		t.Fatal(err)
	}
	timer = 10
	secrets[0].Name = "EDITED"
	change, err := candidate.Change()
	if err != nil || *change.Before().WaitTimer != 0 || candidate.secrets[0].Name != "SECRET" {
		t.Fatal(change, err)
	}
	empty, _ := NewEnvironmentPruneObservation(repo, target, nil)
	if empty.Equal(EnvironmentPruneObservation{}) {
		t.Fatal("missing observation equals acquired empty")
	}
	for _, candidates := range [][]EnvironmentPruneCandidate{{{}}, {candidate, candidate}} {
		if _, err := NewEnvironmentPruneObservation(repo, target, candidates); err == nil {
			t.Fatal("invalid snapshot accepted")
		}
	}
	for _, test := range []struct {
		state   EnvironmentState
		vars    VariableInventory
		secrets []EnvironmentSecretMetadata
		custom  bool
	}{{EnvironmentState{CustomRules: []EnvironmentCustomRule{}, ID: 9, Policy: EnvironmentDeclaration{Name: "old"}}, variables, secrets, true}, {state, VariableInventory{}, secrets, true}, {state, variables, nil, true}, {state, variables, secrets, false}, {state, variables, []EnvironmentSecretMetadata{{Name: "SECRET", CreatedAt: "wrong", UpdatedAt: "wrong"}}, true}} {
		if _, err := NewEnvironmentPruneCandidate(repo, target, test.state, test.vars, test.secrets, test.custom); err == nil {
			t.Fatal("incomplete evidence accepted")
		}
	}
}
