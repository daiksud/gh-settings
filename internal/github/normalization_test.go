package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestWorkflowPermissionNormalizationRetainsRawObservation(t *testing.T) {
	got := NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "FUTURE", CanApprovePullRequestReviews: false})
	if got.DefaultWorkflowPermissions != "read" || got.Normalization.Path != "repository.settings.actions.general.workflow_permissions" || got.Normalization.RawValue != "FUTURE" || got.Normalization.Fallback != "read_repository_contents_and_packages_permissions" || got.CanApprovePullRequestReviews {
		t.Fatalf("normalization lost effective value or raw provenance: %+v", got)
	}
	if again := NormalizeWorkflowPermissions(got); again != got {
		t.Fatalf("repeated projection lost original provenance: %+v", again)
	}
	for _, raw := range []string{"read", "write", ""} {
		source := settings.WorkflowPermissionsState{DefaultWorkflowPermissions: raw, CanApprovePullRequestReviews: false}
		result := NormalizeWorkflowPermissions(source)
		if raw != "" && result != source {
			t.Fatalf("supported observation changed: %+v", result)
		}
		if raw == "" && (result.DefaultWorkflowPermissions != "read" || result.Normalization.Path == "" || result.Normalization.RawValue != "") {
			t.Fatalf("empty typed token lost its provenance: %+v", result)
		}
	}
}

func TestWorkflowPermissionReadbackUsesRawEvidence(t *testing.T) {
	for _, test := range []struct {
		name           string
		permission     *string
		approval       *bool
		actual         string
		want           bool
		beforeApproval bool
		actualApproval bool
	}{
		{"explicit fallback retained", actionsPointer("read"), nil, "read", true, false, false},
		{"fallback projection is not confirmation", actionsPointer("read"), nil, "FUTURE", false, false, false},
		{"different choice retained", actionsPointer("write"), nil, "write", true, false, false},
		{"different choice unconfirmed", actionsPointer("write"), nil, "OTHER", false, false, false},
		{"omitted permission preserved", nil, actionsPointer(false), "FUTURE", true, false, false},
		{"omitted raw companion changed", nil, actionsPointer(false), "OTHER", false, false, false},
		{"explicit false approval retained", nil, actionsPointer(false), "FUTURE", true, true, false},
		{"explicit false approval not retained", nil, actionsPointer(false), "FUTURE", false, true, true},
		{"omitted approval changed", actionsPointer("read"), nil, "read", false, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "FUTURE", CanApprovePullRequestReviews: test.beforeApproval})
			actual := NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: test.actual, CanApprovePullRequestReviews: test.actualApproval})
			got := workflowPermissionsMatch(before, actual, settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: test.permission, CanApprovePullRequestReviews: test.approval})
			if got != test.want {
				t.Fatalf("readback verdict=%v, want %v", got, test.want)
			}
		})
	}
}
