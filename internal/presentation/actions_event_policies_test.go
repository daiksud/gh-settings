package presentation

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestActionsActorPolicyLine(t *testing.T) {
	actors := []settings.ActionsPolicyActor{{Type: "User", ID: 71}, {Type: "RepositoryRole", ID: 4}, {Type: "RepositoryRole", ID: 2}, {Type: "RepositoryRole", ID: 5}}
	policy := settings.ActionsEventPolicy{Name: "actors", Enforcement: "active", Actors: &actors}
	change, err := settings.NewActionsEventPolicyChange(nil, policy)
	if err != nil {
		t.Fatal(err)
	}
	line := ActionsEventPolicyLine(change)
	for _, want := range []string{`restrict_events.enabled=false`, `restrict_actors.allowed_actors=`, `{"user":71}`, `{"role":"write"}`, `{"role":"maintain"}`, `{"role":"repository_admin"}`} {
		if !strings.Contains(line, want) {
			t.Fatalf("missing %s in %s", want, line)
		}
	}
	empty := []settings.ActionsPolicyActor{}
	policy.Actors = &empty
	policy.Events = []string{"push"}
	change, err = settings.NewActionsEventPolicyChange(nil, policy)
	if err != nil {
		t.Fatal(err)
	}
	line = ActionsEventPolicyLine(change)
	if !strings.Contains(line, `restrict_actors.allowed_actors=[]`) || !strings.Contains(line, `restrict_events.allowed_events=["push"]`) {
		t.Fatal(line)
	}
	policy.Actors = nil
	change, err = settings.NewActionsEventPolicyChange(nil, policy)
	if err != nil {
		t.Fatal(err)
	}
	if line := ActionsEventPolicyLine(change); !strings.Contains(line, `restrict_actors.enabled=false`) {
		t.Fatal(line)
	}
}

func TestValidateActionsPolicyRestrictionRemoval(t *testing.T) {
	for _, removeActors := range []bool{false, true} {
		for _, changeMetadata := range []bool{false, true} {
			actors := []settings.ActionsPolicyActor{{Type: "User", ID: 41}}
			before := settings.ActionsEventPolicy{ID: 7, Name: "policy", Enforcement: "active", Events: []string{"push"}, Actors: &actors}
			declaration := before
			if removeActors {
				declaration.Actors = nil
				declaration.DisableActors = true
			} else {
				declaration.Events = nil
				declaration.DisableEvents = true
			}
			if changeMetadata {
				declaration.Name = "renamed"
				declaration.Enforcement = "disabled"
			}
			change, err := settings.NewActionsEventPolicyChange(&before, declaration)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateActionsEventPolicyChanges([]settings.ActionsEventPolicyChange{change}); err != nil {
				t.Fatal(err)
			}
			line := ActionsEventPolicyLine(change)
			disabled := "restrict_events.enabled=false"
			if removeActors {
				disabled = "restrict_actors.enabled=false"
			}
			if !strings.Contains(line, disabled) {
				t.Fatal(line)
			}
		}
	}
}
