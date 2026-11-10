package settings

import (
	"reflect"
	"testing"
)

func TestActionsWorkflowTargetValidation(t *testing.T) {
	for _, target := range []*ActionsWorkflowTarget{
		{}, {Include: []string{""}}, {Include: []string{" "}}, {Include: []string{"a\nb"}},
		{Include: []string{"a", "a"}}, {Exclude: []string{"a", "a"}},
		{Include: []string{"~ALL", "a"}}, {Exclude: []string{"~ALL"}},
	} {
		if ValidateActionsWorkflowTarget(target) == nil {
			t.Fatalf("accepted %#v", target)
		}
	}
	for _, target := range []*ActionsWorkflowTarget{nil, {Include: []string{"~ALL"}}, {Exclude: []string{".github/workflows/skip*.yml"}}, {Include: []string{".github/workflows/*.yml"}, Exclude: []string{"**/skip.yml"}}} {
		if err := ValidateActionsWorkflowTarget(target); err != nil {
			t.Fatal(err)
		}
	}
	if !EqualActionsWorkflowTargets(nil, &ActionsWorkflowTarget{}) || !EqualActionsWorkflowTargets(nil, &ActionsWorkflowTarget{Include: []string{"~ALL"}}) {
		t.Fatal("all representations differ")
	}
}

func TestActionsWorkflowTargetPreservationResetAndOwnership(t *testing.T) {
	before := eventPolicy(7, "events")
	before.TargetWorkflows = &ActionsWorkflowTarget{Include: []string{"z", "a"}, Exclude: []string{"skip"}}
	desired := eventPolicy(7, "events")
	if changes, err := ReconcileActionsPolicies(eventObservation(t, before), eventDeclarations(t, desired)); err != nil || len(changes) != 0 {
		t.Fatalf("omitted targeting changed policy: %v %v", changes, err)
	}
	desired.Enforcement = "disabled"
	changes, err := ReconcileActionsPolicies(eventObservation(t, before), eventDeclarations(t, desired))
	if err != nil || len(changes) != 1 || !EqualActionsWorkflowTargets(changes[0].After().TargetWorkflows, before.TargetWorkflows) {
		t.Fatal(changes, err)
	}
	before.TargetWorkflows.Include[0] = "mutated"
	if changes[0].Before().TargetWorkflows.Include[1] != "z" {
		t.Fatal("target aliased")
	}
	desired.TargetWorkflows = &ActionsWorkflowTarget{Include: []string{"~ALL"}}
	changes, err = ReconcileActionsPolicies(eventObservation(t, before), eventDeclarations(t, desired))
	if err != nil || len(changes) != 1 || !EqualActionsWorkflowTargets(changes[0].After().TargetWorkflows, nil) {
		t.Fatal(changes, err)
	}
	after := changes[0].After()
	after.TargetWorkflows.Include[0] = "mutated"
	if !reflect.DeepEqual(changes[0].After().TargetWorkflows.Include, []string{"~ALL"}) {
		t.Fatal("change target aliased")
	}
	exported, err := ExportActionsPolicies(eventObservation(t, eventPolicy(7, "all")))
	if err != nil || exported.Values()[0].TargetWorkflows == nil || !EqualActionsWorkflowTargets(exported.Values()[0].TargetWorkflows, nil) {
		t.Fatal(exported, err)
	}
}
