package settings

import "testing"

func TestNormalizedForkApprovalChangeIsClosedAndActionable(t *testing.T) {
	for _, after := range []string{"first_time_contributors_new_to_github", "first_time_contributors", "all_external_contributors"} {
		record := ObservationNormalization{Path: "repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors", RawValue: "FUTURE", Fallback: "require_approval_for_first_time_contributors"}
		change, err := NewActionsPolicyChangeWithNormalization("actions.fork_pr_approval_policy", "first_time_contributors", after, record)
		before, actual := change.Values()
		if err != nil || change.Field() != "actions.fork_pr_approval_policy" || before != "first_time_contributors" || actual != after || change.Normalization() != record {
			t.Fatal("valid normalized write rejected/lost provenance", change, err)
		}
		if err := ValidateActionsPolicyChanges([]ActionsPolicyChange{change}); err != nil {
			t.Fatal("closed normalized change failed reconstruction", err)
		}
		record.RawValue = "caller mutation"
		if change.Normalization().RawValue != "FUTURE" {
			t.Fatal("change borrowed record")
		}
	}
	if _, err := NewActionsPolicyChange("actions.fork_pr_approval_policy", "first_time_contributors", "first_time_contributors"); err == nil {
		t.Fatal("ordinary equal-value restriction changed")
	}
}

func TestNormalizedForkChangeRejectsForgedAndCrossFieldEvidence(t *testing.T) {
	fork := ObservationNormalization{Path: ForkApprovalPath, RawValue: "FUTURE", Fallback: "require_approval_for_first_time_contributors"}
	workflow := ObservationNormalization{Path: WorkflowPermissionsPath, RawValue: "FUTURE", Fallback: "read_repository_contents_and_packages_permissions"}
	for _, test := range []struct {
		field         string
		before, after any
		record        ObservationNormalization
	}{
		{"actions.fork_pr_approval_policy", "first_time_contributors", "first_time_contributors", workflow},
		{"actions.default_workflow_permissions", "read", "write", fork},
		{"actions.can_approve_pull_request_reviews", false, true, fork},
		{"actions.fork_pr_approval_policy", "all_external_contributors", "first_time_contributors", fork},
		{"actions.fork_pr_approval_policy", false, "first_time_contributors", fork},
		{"actions.fork_pr_approval_policy", "first_time_contributors", true, fork},
		{"actions.fork_pr_approval_policy", "first_time_contributors", "require_approval_for_all_external_contributors", fork},
		{"actions.fork_pr_approval_policy", "first_time_contributors", "first_time_contributors", ObservationNormalization{Path: ForkApprovalPath, RawValue: "all_external_contributors", Fallback: "require_approval_for_first_time_contributors"}},
	} {
		change, err := NewActionsPolicyChangeWithNormalization(test.field, test.before, test.after, test.record)
		if err == nil || change.Field() != "" || change.Normalization() != (ObservationNormalization{}) {
			t.Fatal("forged normalization authorized change", test, change, err)
		}
	}
}
