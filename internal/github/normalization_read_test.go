package github

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestWorkflowReadNormalizesOnlyCompleteTypedObservations(t *testing.T) {
	for _, raw := range []string{"read", "write", "FUTURE", ""} {
		client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method != "GET" || req.URL.Path != "/repos/owner/current/actions/permissions/workflow" {
				t.Fatal("read changed remote state or endpoint", req.Method, req.URL)
			}
			return actionsResponse(req, 200, fmt.Sprintf(`{"default_workflow_permissions":%q,"can_approve_pull_request_reviews":false}`, raw)), nil
		})
		state, err := client.ReadWorkflowPermissions(repository.Repository{Owner: "owner", Name: "current"})
		if err != nil || state.CanApprovePullRequestReviews {
			t.Fatal("complete typed GET failed or false approval lost", state, err)
		}
		if raw == "read" || raw == "write" {
			if state.DefaultWorkflowPermissions != raw || state.Normalization != (settings.ObservationNormalization{}) {
				t.Fatal("supported GET choice changed", state)
			}
		} else if state.DefaultWorkflowPermissions != "read" || state.Normalization.Path != "repository.settings.actions.general.workflow_permissions.default_permissions" || state.Normalization.RawValue != raw || state.Normalization.Fallback != "read_repository_contents_and_packages_permissions" {
			t.Fatal("unknown GET choice lost provenance", state)
		}
	}
	for _, body := range []string{`{}`, `null`, `{"can_approve_pull_request_reviews":false}`, `{"default_workflow_permissions":null,"can_approve_pull_request_reviews":false}`, `{"default_workflow_permissions":false,"can_approve_pull_request_reviews":false}`, `{"default_workflow_permissions":0,"can_approve_pull_request_reviews":false}`, `{"default_workflow_permissions":{},"can_approve_pull_request_reviews":false}`, `{"default_workflow_permissions":"FUTURE"}`, `{"default_workflow_permissions":"FUTURE","can_approve_pull_request_reviews":null}`, `{"default_workflow_permissions":"FUTURE","can_approve_pull_request_reviews":"false"}`} {
		client := actionsClient(t, func(req *http.Request) (*http.Response, error) { return actionsResponse(req, 200, body), nil })
		state, err := client.ReadWorkflowPermissions(repository.Repository{Owner: "owner", Name: "current"})
		if err == nil || state != (settings.WorkflowPermissionsState{}) {
			t.Fatalf("incomplete or malformed response became fallback: %s %+v %v", body, state, err)
		}
	}
}
