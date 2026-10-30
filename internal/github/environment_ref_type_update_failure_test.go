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

func TestEnvironmentRefTypeUpdateRequiresCanonicalCreationAcknowledgment(t *testing.T) {
	for _, acknowledgment := range []string{
		`{"id":8,"type":"future","name":"main"}`,
		`{"id":8,"name":"main"}`,
		`{"id":8,"type":"tag","name":"main"}`,
	} {
		t.Run(acknowledgment, func(t *testing.T) {
			var history []string
			created, warnings := false, 0
			client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
				base := "/repos/owner/current/environments/production"
				if req.Method == "GET" {
					responses := map[string]string{
						"/repos/owner/current":                `{"id":71,"full_name":"owner/current","archived":false}`,
						"/repos/owner/current/environments":   `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`,
						base:                                  `{"id":1,"name":"production","protection_rules":[],"can_admins_bypass":false,"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}`,
						base + "/deployment_protection_rules": customEnvironmentEmpty,
						base + "/deployment-branch-policies":  `{"total_count":1,"branch_policies":[{"id":7,"type":"future","name":"main"}]}`,
					}
					if body, ok := responses[req.URL.Path]; ok && !created {
						return actionsResponse(req, 200, body), nil
					}
					t.Fatal("read after unconfirmed creation", req.URL)
				}
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				switch {
				case req.Method == "PUT" && req.URL.Path == base:
					want := map[string]any{"wait_timer": float64(0), "prevent_self_review": false, "can_admins_bypass": false, "reviewers": []any{}, "deployment_branch_policy": map[string]any{"protected_branches": false, "custom_branch_policies": true}}
					if !reflect.DeepEqual(payload, want) || warnings != 1 {
						t.Fatal("literal PUT or warning", payload, warnings)
					}
					history = append(history, "PUT")
					return actionsResponse(req, 200, `{"id":1,"name":"production","protection_rules":[],"can_admins_bypass":false,"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}`), nil
				case req.Method == "POST" && req.URL.Path == base+"/deployment-branch-policies":
					if !reflect.DeepEqual(payload, map[string]any{"type": "branch", "name": "main"}) || warnings != 2 {
						t.Fatal("literal POST or fresh warning", payload, warnings)
					}
					history = append(history, "POST")
					created = true
					return actionsResponse(req, 200, acknowledgment), nil
				default:
					t.Fatal("unexpected write", req.Method, req.URL)
					return nil, nil
				}
			}).rest, RefTypeNormalizationWarning: func(string, settings.EnvironmentPatternState) { warnings++ }}
			zero, off, reviewers := 0, false, []settings.EnvironmentReviewer{}
			patterns := []settings.EnvironmentPattern{{Type: "branch", Name: "main"}}
			desired := settings.EnvironmentDeclaration{Name: "production", WaitTimer: &zero, PreventSelfReview: &off, CanAdminsBypass: &off, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true, DeploymentBranchPolicy: &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}, DeploymentBranchPatterns: &patterns}
			written, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, desired)
			var pre interface{ ReadFailedBeforeWrite() }
			if err == nil || !strings.Contains(err.Error(), "write may have applied") || !strings.Contains(err.Error(), "earlier writes remain applied") || errors.As(err, &pre) || written || !created || !reflect.DeepEqual(history, []string{"PUT", "POST"}) || warnings != 2 {
				t.Fatal("unconfirmed creation advanced or hid earlier effects", written, created, history, warnings, err)
			}
		})
	}
}
