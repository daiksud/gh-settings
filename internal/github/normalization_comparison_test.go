package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestWorkflowNormalizedComparisonRemainsActionable(t *testing.T) {
	observed := NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "FUTURE"})
	for _, desired := range []string{"read", "write"} {
		managed, changes, err := settings.CompareWorkflowPermissions(observed, settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &desired})
		if err != nil || managed != 1 || len(changes) != 1 {
			t.Fatalf("raw out-of-UI value became equality: %d %+v %v", managed, changes, err)
		}
		if changes[0].Normalization() != observed.Normalization {
			t.Fatal("comparison lost observation provenance")
		}
		before, after := changes[0].Values()
		if before != "read" || after != desired || settings.ValidateActionsPolicyChanges(changes) != nil {
			t.Fatal("invalid normalized comparison", before, after)
		}
	}
	managed, changes, err := settings.CompareWorkflowPermissions(observed, settings.ActionsPolicyDeclaration{CanApprovePullRequestReviews: actionsPointer(false)})
	if err != nil || managed != 1 || len(changes) != 0 {
		t.Fatalf("omitted permission became managed: %d %+v %v", managed, changes, err)
	}
	managed, changes, err = settings.CompareWorkflowPermissions(observed, settings.ActionsPolicyDeclaration{CanApprovePullRequestReviews: actionsPointer(true)})
	if err != nil || managed != 1 || len(changes) != 1 || changes[0].Field() != "actions.can_approve_pull_request_reviews" || changes[0].Normalization() != (settings.ObservationNormalization{}) {
		t.Fatalf("unmanaged permission leaked into approval change: %d %+v %v", managed, changes, err)
	}
	before, after := changes[0].Values()
	if before != false || after != true {
		t.Fatal("approval-only values lost", before, after)
	}
	managed, changes, err = settings.CompareWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "read"}, settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("read")})
	if err != nil || managed != 1 || len(changes) != 0 {
		t.Fatalf("exact observed match became drift: %d %+v %v", managed, changes, err)
	}
	if _, err := settings.NewActionsPolicyChange("actions.default_workflow_permissions", "read", "read"); err == nil {
		t.Fatal("ordinary equal comparison accepted without provenance")
	}
}

func TestWorkflowNormalizationComparisonRejectsForgedProvenance(t *testing.T) {
	valid := NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "FUTURE"}).Normalization
	for _, test := range []struct {
		field, before, after string
		normalization        settings.ObservationNormalization
	}{
		{"actions.allowed_actions", "all", "all", valid},
		{"actions.default_workflow_permissions", "read", "read", settings.ObservationNormalization{}},
		{"actions.default_workflow_permissions", "write", "read", valid},
		{"actions.default_workflow_permissions", "read", "FUTURE", valid},
		{"actions.default_workflow_permissions", "read", "read", settings.ObservationNormalization{Path: valid.Path, RawValue: "read", Fallback: valid.Fallback}},
		{"actions.default_workflow_permissions", "read", "read", settings.ObservationNormalization{Path: "unmanaged", RawValue: "FUTURE", Fallback: valid.Fallback}},
		{"actions.default_workflow_permissions", "read", "read", settings.ObservationNormalization{Path: valid.Path, RawValue: "FUTURE", Fallback: "write"}},
	} {
		if _, err := settings.NewActionsPolicyChangeWithNormalization(test.field, test.before, test.after, test.normalization); err == nil {
			t.Fatalf("forged comparison accepted: %+v", test)
		}
	}
}
