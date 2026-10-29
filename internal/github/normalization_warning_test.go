package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestWorkflowNormalizationWarningsRespectManagedIntent(t *testing.T) {
	for _, test := range []struct {
		name, raw string
		desired   settings.ActionsPolicyDeclaration
		want      int
	}{
		{"managed fallback", "FUTURE", settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("read")}, 1},
		{"managed different choice", "FUTURE", settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("write")}, 1},
		{"exact choice", "read", settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("read")}, 0},
		{"omitted permission", "FUTURE", settings.ActionsPolicyDeclaration{}, 0},
		{"explicit false approval only", "FUTURE", settings.ActionsPolicyDeclaration{CanApprovePullRequestReviews: actionsPointer(false)}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: test.raw})
			var warnings []settings.ObservationNormalization
			client := ActionsPolicyClient{NormalizationWarning: func(normalization settings.ObservationNormalization) { warnings = append(warnings, normalization) }}
			client.reportWorkflowNormalization(state, test.desired)
			if len(warnings) != test.want || (len(warnings) == 1 && warnings[0] != state.Normalization) {
				t.Fatalf("warning disclosed unmanaged values or lost provenance: %+v", warnings)
			}
		})
	}
}
