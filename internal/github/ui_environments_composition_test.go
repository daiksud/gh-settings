package github

import (
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestUIEnvironmentExactPUTAndAPIOmissions(t *testing.T) {
	before := `{"id":7,"name":"Production","can_admins_bypass":true,"protection_rules":[{"type":"wait_timer","wait_timer":30},{"type":"required_reviewers","prevent_self_review":true,"reviewers":[{"type":"User","reviewer":{"id":9,"type":"User"}}]}],"deployment_branch_policy":{"protected_branches":true,"custom_branch_policies":false}}`
	after := `{"id":7,"name":"Production","can_admins_bypass":false,"protection_rules":[],"deployment_branch_policy":null}`
	for _, test := range []struct {
		ui      uimodel.Environment
		payload string
	}{
		{uimodel.Environment{Name: "Production", DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{AllowAdministratorsToBypassConfiguredProtectionRules: environmentUIPtr(false)}}, `{"can_admins_bypass":false}`},
		{uimodel.Environment{Name: "Production", DeploymentProtectionRules: &uimodel.EnvironmentDeploymentProtectionRules{WaitTimer: environmentUIPtr(0), PreventSelfReview: environmentUIPtr(false), AllowAdministratorsToBypassConfiguredProtectionRules: environmentUIPtr(false), RequiredReviewers: environmentUIPtr([]uimodel.EnvironmentReviewer{})}, DeploymentBranchesAndTags: environmentUIPtr(uimodel.NoRestriction)}, `{"wait_timer":0,"prevent_self_review":false,"can_admins_bypass":false,"reviewers":[],"deployment_branch_policy":null}`},
	} {
		written := false
		writes := 0
		client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/repos/owner/current":
				if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				return labelsResponse(req, 200, `{"id":1,"full_name":"owner/current","archived":false}`), nil
			case "/repos/owner/current/environments":
				if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				return labelsResponse(req, 200, `{"total_count":2,"environments":[{"id":7,"name":"Production"},{"id":8,"name":"Unlisted"}]}`), nil
			case "/repos/owner/current/environments/Production":
				if req.Method == "PUT" {
					var got, want any
					if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal([]byte(test.payload), &want); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatal("replayed omitted fields or wrong explicit values", got, want)
					}
					written = true
					writes++
					return labelsResponse(req, 200, after), nil
				}
				if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				body := before
				if written {
					body = after
				}
				return labelsResponse(req, 200, body), nil
			case "/repos/owner/current/environments/Production/deployment_protection_rules":
				return labelsResponse(req, 200, `{"total_count":0,"custom_deployment_protection_rules":[]}`), nil
			default:
				t.Fatal("unlisted or unrelated resource accessed", req.Method, req.URL)
			}
			return nil, nil
		}).rest}
		repo := repository.Repository{Owner: "owner", Name: "current"}
		raw, err := client.Read(repo, []string{"Production"})
		if err != nil {
			t.Fatal(err)
		}
		view, err := EnvironmentsObservation(raw)
		if err != nil || len(view) != 1 || view[0].ID != 7 || *view[0].Environment.DeploymentProtectionRules.WaitTimer != 30 || *(*view[0].Environment.DeploymentProtectionRules.RequiredReviewers)[0].Person != 9 || *view[0].Environment.DeploymentBranchesAndTags != "protected_branches_only" {
			t.Fatal(view, err)
		}
		d, err := EnvironmentDeclaration(test.ui)
		if err != nil {
			t.Fatal(err)
		}
		changed, err := client.Update(repo, d)
		if err != nil || !changed || writes != 1 {
			t.Fatal(changed, writes, err)
		}
		raw, err = client.Read(repo, []string{"Production"})
		if err != nil {
			t.Fatal(err)
		}
		view, err = EnvironmentsObservation(raw)
		if err != nil || *view[0].Environment.DeploymentProtectionRules.WaitTimer != 0 || len(*view[0].Environment.DeploymentProtectionRules.RequiredReviewers) != 0 {
			t.Fatal("API-controlled omission was replayed", view, err)
		}
	}
}

func TestUIEnvironmentTypedPatternCRUDAndPartialFailure(t *testing.T) {
	before := `{"id":7,"name":"Production","can_admins_bypass":true,"protection_rules":[],"deployment_branch_policy":null}`
	after := `{"id":7,"name":"Production","can_admins_bypass":true,"protection_rules":[],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}`
	for _, failTag := range []bool{false, true} {
		modeWritten, createdBranch, removed := false, false, false
		operations := []string{}
		client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/repos/owner/current":
				return labelsResponse(req, 200, `{"id":1,"full_name":"owner/current","archived":false}`), nil
			case "/repos/owner/current/environments":
				return labelsResponse(req, 200, `{"total_count":2,"environments":[{"id":7,"name":"Production"},{"id":8,"name":"Unlisted"}]}`), nil
			case "/repos/owner/current/environments/Production":
				if req.Method == "PUT" {
					var got map[string]any
					if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
						t.Fatal(err)
					}
					want := map[string]any{"deployment_branch_policy": map[string]any{"protected_branches": false, "custom_branch_policies": true}}
					if !reflect.DeepEqual(got, want) {
						t.Fatal(got)
					}
					operations = append(operations, "mode")
					modeWritten = true
					return labelsResponse(req, 200, after), nil
				}
				if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				body := before
				if modeWritten {
					body = after
				}
				return labelsResponse(req, 200, body), nil
			case "/repos/owner/current/environments/Production/deployment_protection_rules":
				return labelsResponse(req, 200, `{"total_count":0,"custom_deployment_protection_rules":[]}`), nil
			case "/repos/owner/current/environments/Production/deployment-branch-policies":
				if req.Method == "GET" {
					if removed {
						return labelsResponse(req, 200, `{"total_count":2,"branch_policies":[{"id":41,"type":"branch","name":"main"},{"id":42,"type":"tag","name":"release/*"}]}`), nil
					}
					return labelsResponse(req, 200, `{"total_count":1,"branch_policies":[{"id":31,"type":"branch","name":"obsolete"}]}`), nil
				}
				if req.Method != "POST" || !modeWritten {
					t.Fatal(req.Method, "pattern created before mode")
				}
				var got map[string]any
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if !createdBranch {
					if !reflect.DeepEqual(got, map[string]any{"type": "branch", "name": "main"}) {
						t.Fatal(got)
					}
					operations = append(operations, "branch")
					createdBranch = true
					return labelsResponse(req, 200, `{"id":41,"type":"branch","name":"main"}`), nil
				}
				if !reflect.DeepEqual(got, map[string]any{"type": "tag", "name": "release/*"}) {
					t.Fatal(got)
				}
				operations = append(operations, "tag")
				if failTag {
					return labelsResponse(req, 422, `{"message":"rejected"}`), nil
				}
				return labelsResponse(req, 200, `{"id":42,"type":"tag","name":"release/*"}`), nil
			case "/repos/owner/current/environments/Production/deployment-branch-policies/31":
				if req.Method == "GET" {
					operations = append(operations, "identity")
					return labelsResponse(req, 200, `{"id":31,"type":"branch","name":"obsolete"}`), nil
				}
				if req.Method != "DELETE" {
					t.Fatal(req.Method)
				}
				operations = append(operations, "remove")
				removed = true
				return labelsResponse(req, 204, ""), nil
			default:
				t.Fatal("unlisted or unrelated resource accessed", req.Method, req.URL)
			}
			return nil, nil
		}).rest}
		d, err := EnvironmentDeclaration(uimodel.Environment{Name: "Production", DeploymentBranchesAndTags: environmentUIPtr(uimodel.SelectedBranchesAndTags), DeploymentBranchOrTagRules: environmentUIPtr([]uimodel.EnvironmentDeploymentRule{{RefType: "tag", NamePattern: "release/*"}, {RefType: "branch", NamePattern: "main"}})})
		if err != nil {
			t.Fatal(err)
		}
		changed, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, d)
		if failTag {
			if err == nil || changed || !modeWritten || !createdBranch || removed || !strings.Contains(err.Error(), "earlier writes remain applied") || !reflect.DeepEqual(operations, []string{"mode", "branch", "tag"}) {
				t.Fatal(changed, operations, err)
			}
			continue
		}
		if err != nil || !changed || !reflect.DeepEqual(operations, []string{"mode", "branch", "tag", "identity", "remove"}) {
			t.Fatal(changed, operations, err)
		}
		raw, err := client.Read(repository.Repository{Owner: "owner", Name: "current"}, []string{"Production"})
		if err != nil {
			t.Fatal(err)
		}
		view, err := EnvironmentsObservation(raw)
		if err != nil || len(view) != 1 || view[0].ID != 7 || len(view[0].Patterns) != 2 || view[0].Patterns[0].ID != 41 || view[0].Patterns[1].ID != 42 || len(*view[0].Environment.DeploymentBranchOrTagRules) != 2 {
			t.Fatal(view, err)
		}
	}
}

func TestUIEnvironmentUnavailableAndUnknownAcquisition(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
	}{
		{404, `{"message":"Not Found"}`}, {403, `{"message":"Forbidden"}`},
		{200, `{"id":7,"name":"Production","can_admins_bypass":true,"protection_rules":[{"type":"future"}],"deployment_branch_policy":null}`},
		{200, `{"id":7,"name":"Production","can_admins_bypass":true,"protection_rules":[],"deployment_branch_policy":null,"future_setting":true}`},
	} {
		client := &EnvironmentsClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			if req.Method != "GET" {
				t.Fatal("failed acquisition wrote")
			}
			if req.URL.Path == "/repos/owner/current/environments" {
				return labelsResponse(req, 200, `{"total_count":1,"environments":[{"id":7,"name":"Production"}]}`), nil
			}
			return labelsResponse(req, test.status, test.body), nil
		}).rest}
		raw, err := client.Read(repository.Repository{Owner: "owner", Name: "current"}, nil)
		if err == nil {
			t.Fatal("unavailable/unknown accepted")
		}
		view, err := EnvironmentsObservation(raw)
		if err == nil || len(view) != 0 {
			t.Fatal("failed acquisition defaulted", view, err)
		}
	}
}
