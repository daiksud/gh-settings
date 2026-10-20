package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"net/http"
	"reflect"
	"testing"
)

func forkPtr(v bool) *bool { return &v }
func TestForkReadsRequireApplicableCompleteState(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	complete := `{"run_workflows_from_fork_pull_requests":true,"send_write_tokens_to_workflows":false,"send_secrets_and_variables":true,"require_approval_for_fork_pr_workflows":false}`
	for _, private := range []bool{false, true} {
		for _, visibility := range []string{"public", "private", "internal", "", "null"} {
			calls := 0
			client := &ForkWorkflowsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" {
					t.Fatal(r.Method)
				}
				if r.URL.Path == "/repos/owner/current" {
					body := fmt.Sprintf(`{"visibility":%q}`, visibility)
					if visibility == "null" {
						body = `{"visibility":null}`
					}
					return actionsResponse(r, 200, body), nil
				}
				want := "/repos/owner/current/actions/permissions/fork-pr-contributor-approval"
				body := `{"approval_policy":"first_time_contributors"}`
				if private {
					want = "/repos/owner/current/actions/permissions/fork-pr-workflows-private-repos"
					body = complete
				}
				if r.URL.Path != want {
					t.Fatal("wrong family", r.URL)
				}
				return actionsResponse(r, 200, body), nil
			}).rest}
			var err error
			if private {
				_, err = client.ReadPrivate(repo)
			} else {
				_, err = client.ReadApproval(repo)
			}
			valid := visibility == "public" && !private || visibility == "private" && private
			if (err == nil) != valid || !valid && calls != 1 {
				t.Fatal(private, visibility, calls, err)
			}
		}
	}
	for _, private := range []bool{false, true} {
		bodies := []string{`{}`, `null`, `{"approval_policy":null}`, `{"approval_policy":false}`, `{"approval_policy":"never"}`, `{"Approval_policy":"first_time_contributors"}`}
		if private {
			bodies = []string{`{}`, `null`, `{"run_workflows_from_fork_pull_requests":false}`, `{"run_workflows_from_fork_pull_requests":null,"send_write_tokens_to_workflows":false,"send_secrets_and_variables":true,"require_approval_for_fork_pr_workflows":false}`, `{"run_workflows_from_fork_pull_requests":true,"send_write_tokens_to_workflows":"false","send_secrets_and_variables":true,"require_approval_for_fork_pr_workflows":false}`, `{"run_workflows_from_fork_pull_requests":true,"send_write_tokens_to_workflows":false,"send_secrets_and_variables":true,"Require_approval_for_fork_pr_workflows":false}`}
		}
		for _, body := range bodies {
			client := &ForkWorkflowsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					v := "public"
					if private {
						v = "private"
					}
					return actionsResponse(r, 200, fmt.Sprintf(`{"visibility":%q}`, v)), nil
				}
				return actionsResponse(r, 200, body), nil
			}).rest}
			var err error
			if private {
				_, err = client.ReadPrivate(repo)
			} else {
				_, err = client.ReadApproval(repo)
			}
			if err == nil {
				t.Fatal(private, body)
			}
		}
	}
}
func TestPrivateForkDeclaredPayloadPrerequisitesAndReadback(t *testing.T) {
	for _, mode := range []string{"success", "matching", "missing required declaration", "latent", "pre-read", "write", "enforced", "companion drift", "readback", "visibility drift"} {
		t.Run(mode, func(t *testing.T) {
			values := [4]bool{true, false, true, false}
			writes := 0
			d := settings.PrivateForkDeclaration{Run: forkPtr(true), Secrets: forkPtr(false)}
			switch mode {
			case "matching":
				d.Secrets = forkPtr(true)
			case "missing required declaration":
				d.Run = nil
			case "latent":
				values[0] = false
			}
			client := &ForkWorkflowsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					v := "private"
					if mode == "visibility drift" && writes > 0 {
						v = "public"
					}
					return actionsResponse(r, 200, fmt.Sprintf(`{"visibility":%q}`, v)), nil
				}
				if r.URL.Path != "/repos/owner/current/actions/permissions/fork-pr-workflows-private-repos" {
					t.Fatal("unrelated endpoint", r.URL)
				}
				if r.Method == "GET" {
					if mode == "pre-read" || mode == "readback" && writes > 0 {
						return actionsResponse(r, 404, `{}`), nil
					}
					fields := map[string]bool{}
					for i, key := range settings.PrivateForkKeys() {
						fields[key] = values[i]
					}
					b, _ := json.Marshal(fields)
					return actionsResponse(r, 200, string(b)), nil
				}
				if r.Method != "PUT" {
					t.Fatal(r.Method)
				}
				writes++
				var payload map[string]bool
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(payload, map[string]bool{"run_workflows_from_fork_pull_requests": true, "send_secrets_and_variables": false}) {
					t.Fatal("unowned token/approval replayed", payload)
				}
				if mode == "write" {
					return actionsResponse(r, 422, `{}`), nil
				}
				if mode != "enforced" {
					values[2] = false
				}
				if mode == "companion drift" {
					values[1] = true
				}
				return actionsResponse(r, 204, ""), nil
			}).rest}
			wrote, err := client.UpdatePrivate(repository.Repository{Owner: "owner", Name: "current"}, d)
			good := mode == "success" || mode == "matching"
			if (err == nil) != good || wrote != (mode == "success") {
				t.Fatal(wrote, err)
			}
			pre := mode == "pre-read" || mode == "missing required declaration" || mode == "latent"
			var marker interface{ ReadFailedBeforeWrite() }
			if errors.As(err, &marker) != pre {
				t.Fatal("wrong certainty", err)
			}
			want := 1
			if pre || mode == "matching" {
				want = 0
			}
			if writes != want {
				t.Fatal("wrong writes", writes, want)
			}
			if mode == "success" {
				wrote, err = client.UpdatePrivate(repository.Repository{Owner: "owner", Name: "current"}, d)
				if wrote || err != nil || writes != 1 {
					t.Fatal("second update wrote", wrote, err, writes)
				}
			}
		})
	}
}
func TestForkApprovalExactWriteAndFailureCertainty(t *testing.T) {
	for _, mode := range []string{"success", "pre-read", "write", "enforced", "readback", "visibility drift"} {
		t.Run(mode, func(t *testing.T) {
			policy := "first_time_contributors"
			writes := 0
			client := &ForkWorkflowsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/repos/owner/current" {
					v := "public"
					if writes > 0 && mode == "visibility drift" {
						v = "private"
					}
					return actionsResponse(r, 200, fmt.Sprintf(`{"visibility":%q}`, v)), nil
				}
				if r.URL.Path != "/repos/owner/current/actions/permissions/fork-pr-contributor-approval" {
					t.Fatal(r.URL)
				}
				if r.Method == "GET" {
					if mode == "pre-read" || mode == "readback" && writes > 0 {
						return actionsResponse(r, 403, `{}`), nil
					}
					return actionsResponse(r, 200, fmt.Sprintf(`{"approval_policy":%q}`, policy)), nil
				}
				if r.Method != "PUT" {
					t.Fatal(r.Method)
				}
				writes++
				var payload map[string]string
				json.NewDecoder(r.Body).Decode(&payload)
				if !reflect.DeepEqual(payload, map[string]string{"approval_policy": "all_external_contributors"}) {
					t.Fatal(payload)
				}
				if mode == "write" {
					return actionsResponse(r, 422, `{}`), nil
				}
				if mode != "enforced" {
					policy = "all_external_contributors"
				}
				return actionsResponse(r, 204, ""), nil
			}).rest}
			repo := repository.Repository{Owner: "owner", Name: "current"}
			wrote, err := client.UpdateApproval(repo, "all_external_contributors")
			if (err == nil) != (mode == "success") || wrote != (mode == "success") {
				t.Fatal(wrote, err)
			}
			var marker interface{ ReadFailedBeforeWrite() }
			if errors.As(err, &marker) != (mode == "pre-read") {
				t.Fatal("certainty", err)
			}
			if mode == "success" {
				wrote, err = client.UpdateApproval(repo, policy)
				if wrote || err != nil || writes != 1 {
					t.Fatal(wrote, err, writes)
				}
			}
		})
	}
}
