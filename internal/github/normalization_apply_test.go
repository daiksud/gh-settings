package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestWorkflowNormalizedApplyVerifiesRawReadback(t *testing.T) {
	for _, test := range []struct {
		name        string
		desired     settings.ActionsPolicyDeclaration
		actualRaw   string
		wantPayload map[string]any
		wantError   bool
		warnings    int
	}{
		{"fallback retained", settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("read")}, "read", map[string]any{"default_workflow_permissions": "read"}, false, 1},
		{"different choice retained", settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("write")}, "write", map[string]any{"default_workflow_permissions": "write"}, false, 1},
		{"fallback not confirmed", settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("read")}, "FUTURE", map[string]any{"default_workflow_permissions": "read"}, true, 2},
		{"different choice not confirmed", settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("write")}, "FUTURE", map[string]any{"default_workflow_permissions": "write"}, true, 2},
		{"omitted permission preserved", settings.ActionsPolicyDeclaration{CanApprovePullRequestReviews: actionsPointer(false)}, "FUTURE", map[string]any{"can_approve_pull_request_reviews": false}, false, 0},
		{"raw companion changed", settings.ActionsPolicyDeclaration{CanApprovePullRequestReviews: actionsPointer(false)}, "OTHER", map[string]any{"can_approve_pull_request_reviews": false}, true, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			var operations []string
			var warnings []settings.ObservationNormalization
			written := false
			client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/repos/owner/current/actions/permissions/workflow" {
					t.Fatal("unrelated policy requested", req.URL)
				}
				operations = append(operations, req.Method)
				if req.Method == "PUT" {
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil || !reflect.DeepEqual(payload, test.wantPayload) {
						t.Fatal("fallback or companion replayed in payload", payload, err)
					}
					written = true
					return actionsResponse(req, 204, ""), nil
				}
				if req.Method != "GET" {
					t.Fatal("unexpected mutation", req.Method)
				}
				raw, approval := "FUTURE", true
				if written {
					raw = test.actualRaw
					if test.desired.CanApprovePullRequestReviews != nil {
						approval = *test.desired.CanApprovePullRequestReviews
					}
				}
				return actionsResponse(req, 200, fmt.Sprintf(`{"default_workflow_permissions":%q,"can_approve_pull_request_reviews":%t}`, raw, approval)), nil
			})
			client.NormalizationWarning = func(normalization settings.ObservationNormalization) { warnings = append(warnings, normalization) }
			err := client.ApplyWorkflowPermissions(repository.Repository{Owner: "owner", Name: "current"}, test.desired)
			if (err != nil) != test.wantError || !reflect.DeepEqual(operations, []string{"GET", "PUT", "GET"}) || len(warnings) != test.warnings {
				t.Fatalf("raw verification/diagnostics failed: operations=%v warnings=%+v error=%v", operations, warnings, err)
			}
			if err != nil {
				var preflightFailure interface{ ReadFailedBeforeWrite() }
				if errors.As(err, &preflightFailure) || !strings.Contains(err.Error(), "may already have applied") {
					t.Fatal("post-write uncertainty mislabeled", err)
				}
			}
			for _, warning := range warnings {
				if warning.Path != "repository.settings.actions.general.workflow_permissions.default_permissions" || warning.RawValue != "FUTURE" || warning.Fallback != "read_repository_contents_and_packages_permissions" {
					t.Fatal("raw warning provenance changed", warning)
				}
			}
		})
	}
}
