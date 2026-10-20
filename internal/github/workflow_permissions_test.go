package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestWorkflowPermissionsCompleteReads(t *testing.T) {
	for _, body := range []string{`{"default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`, `{"default_workflow_permissions":"write","can_approve_pull_request_reviews":true}`, `{}`, `null`, `{"default_workflow_permissions":"unknown","can_approve_pull_request_reviews":false}`, `{"default_workflow_permissions":"read","can_approve_pull_request_reviews":null}`, `{"default_workflow_permissions":null,"can_approve_pull_request_reviews":false}`} {
		client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method != "GET" || req.URL.Path != "/repos/owner/current/actions/permissions/workflow" {
				t.Fatal(req.Method, req.URL)
			}
			return actionsResponse(req, 200, body), nil
		})
		state, err := client.ReadWorkflowPermissions(repository.Repository{Owner: "owner", Name: "current"})
		valid := strings.Contains(body, `"read"`) || strings.Contains(body, `"write"`)
		valid = valid && !strings.Contains(body, "null")
		if (err == nil) != valid || (err != nil && !reflect.DeepEqual(state, settings.WorkflowPermissionsState{})) {
			t.Fatal(body, state, err)
		}
	}
}

func TestWorkflowPermissionsDeclaredOnlyPUTAndCompanionVerification(t *testing.T) {
	for _, test := range []struct {
		name string
		d    settings.ActionsPolicyDeclaration
		want map[string]any
	}{
		{"read", settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("read")}, map[string]any{"default_workflow_permissions": "read"}},
		{"write", settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("write")}, map[string]any{"default_workflow_permissions": "write"}},
		{"deny", settings.ActionsPolicyDeclaration{CanApprovePullRequestReviews: actionsPointer(false)}, map[string]any{"can_approve_pull_request_reviews": false}},
		{"allow", settings.ActionsPolicyDeclaration{CanApprovePullRequestReviews: actionsPointer(true)}, map[string]any{"can_approve_pull_request_reviews": true}},
		{"both", settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("write"), CanApprovePullRequestReviews: actionsPointer(true)}, map[string]any{"default_workflow_permissions": "write", "can_approve_pull_request_reviews": true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := map[string]any{"default_workflow_permissions": "read", "can_approve_pull_request_reviews": false}
			var writes []map[string]any
			client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/repos/owner/current/actions/permissions/workflow" {
					t.Fatal("unrelated endpoint", req.URL)
				}
				if req.Method == "PUT" {
					var body map[string]any
					json.NewDecoder(req.Body).Decode(&body)
					writes = append(writes, body)
					for k, v := range body {
						state[k] = v
					}
					return actionsResponse(req, 204, ""), nil
				}
				if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				body, _ := json.Marshal(state)
				return actionsResponse(req, 200, string(body)), nil
			})
			for i := 0; i < 2; i++ {
				if err := client.ApplyWorkflowPermissions(repository.Repository{Owner: "owner", Name: "current"}, test.d); err != nil {
					t.Fatal(err)
				}
			}
			if len(writes) != 2 || !reflect.DeepEqual(writes[0], test.want) || !reflect.DeepEqual(writes[1], test.want) {
				t.Fatal(writes)
			}
		})
	}
	for _, failure := range []string{"reset companion", "missing readback", "mismatching value"} {
		t.Run(failure, func(t *testing.T) {
			written := false
			client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method == "PUT" {
					written = true
					return actionsResponse(req, 204, ""), nil
				}
				body := `{"default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`
				if written {
					switch failure {
					case "reset companion":
						body = `{"default_workflow_permissions":"write","can_approve_pull_request_reviews":true}`
					case "missing readback":
						body = `{}`
					}
				}
				return actionsResponse(req, 200, body), nil
			})
			err := client.ApplyWorkflowPermissions(repository.Repository{}, settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("write")})
			if err == nil || !strings.Contains(err.Error(), "may already have applied") {
				t.Fatal(err)
			}
			var readFailure interface{ ReadFailedBeforeWrite() }
			if errors.As(err, &readFailure) {
				t.Fatal("post-write verification mislabeled as a preflight read failure", err)
			}
		})
	}
}

func TestWorkflowPermissionsErrorsAndOmission(t *testing.T) {
	for _, method := range []string{"GET", "PUT"} {
		for _, status := range []int{301, 403, 404, 409, 422} {
			writes := 0
			client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
				if req.Method == "PUT" {
					writes++
				}
				if req.Method == method {
					return actionsResponse(req, status, `{"message":"denied"}`), nil
				}
				return actionsResponse(req, 200, `{"default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`), nil
			})
			err := client.ApplyWorkflowPermissions(repository.Repository{}, settings.ActionsPolicyDeclaration{CanApprovePullRequestReviews: actionsPointer(true)})
			var readFailure interface{ ReadFailedBeforeWrite() }
			if errors.As(err, &readFailure) != (method == "GET") {
				t.Fatal("incorrect pre-write/write stage", method, status, err)
			}
			want := 1
			if method == "GET" {
				want = 0
			}
			if err == nil || writes != want {
				t.Fatal(method, status, writes, err)
			}
		}
	}
	client := actionsClient(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("omitted/invalid policy invoked API")
		return nil, nil
	})
	if err := client.ApplyWorkflowPermissions(repository.Repository{}, settings.ActionsPolicyDeclaration{}); err != nil {
		t.Fatal(err)
	}
	if err := client.ApplyWorkflowPermissions(repository.Repository{}, settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("unknown")}); err == nil {
		t.Fatal("invalid policy accepted")
	}
	if err := client.ApplyActionsPolicy(repository.Repository{}, settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: actionsPointer("read")}); err != nil {
		t.Fatal("workflow-only declaration reached execution API", err)
	}
}
