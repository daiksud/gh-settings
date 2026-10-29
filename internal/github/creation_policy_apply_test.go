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

func TestCreationPolicyApplyVerifiesFreshRawAndIdentity(t *testing.T) {
	for _, control := range []settings.CreationPolicyControl{settings.IssuePermissionsControl, settings.PullRequestPermissionsControl} {
		for _, test := range []struct {
			name, before, desired, actual, id string
			wantError                         bool
			warnings                          int
		}{
			{"canonical matching", "ALL", "ALL", "ALL", "repo-id", false, 0},
			{"canonical changing", "COLLABORATORS_ONLY", "ALL", "ALL", "repo-id", false, 0},
			{"fallback verified", "FUTURE", "ALL", "ALL", "repo-id", false, 1},
			{"different verified", "FUTURE", "COLLABORATORS_ONLY", "COLLABORATORS_ONLY", "repo-id", false, 1},
			{"fallback unconfirmed", "FUTURE", "ALL", "FUTURE", "repo-id", true, 2},
			{"different unconfirmed", "FUTURE", "COLLABORATORS_ONLY", "FUTURE", "repo-id", true, 2},
			{"repository recreated", "FUTURE", "ALL", "ALL", "other-id", true, 1},
		} {
			t.Run(string(control)+"/"+test.name, func(t *testing.T) {
				graphqlField := "issueCreationPolicy"
				desired := settings.RepositoryDeclaration{IssueCreationPolicy: &test.desired}
				if control == settings.PullRequestPermissionsControl {
					graphqlField = "pullRequestCreationPolicy"
					desired = settings.RepositoryDeclaration{PullRequestCreationPolicy: &test.desired}
				}
				var operations []string
				var warnings []settings.ObservationNormalization
				client := prAccessTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
					var request struct {
						Query     string
						Variables map[string]any
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Fatal(err)
					}
					if strings.Contains(request.Query, "mutation") {
						operations = append(operations, "mutation")
						want := map[string]any{"input": map[string]any{"repositoryId": "repo-id", graphqlField: test.desired}}
						if !reflect.DeepEqual(request.Variables, want) {
							t.Fatal("fallback or unmanaged field replayed", request.Variables, want)
						}
						body, _ := json.Marshal(map[string]any{"data": map[string]any{"updateRepository": map[string]any{"repository": map[string]any{"id": "repo-id", "nameWithOwner": "owner/current", graphqlField: test.desired}}}})
						return labelsResponse(r, 200, string(body)), nil
					}
					operations = append(operations, "query")
					if !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) {
						t.Fatal("readback target changed", request.Variables)
					}
					raw, id := test.before, "repo-id"
					if len(operations) > 1 {
						raw, id = test.actual, test.id
					}
					body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": map[string]any{"id": id, "nameWithOwner": "owner/current", graphqlField: raw}}})
					return labelsResponse(r, 200, string(body)), nil
				}))
				client.NormalizationWarning = func(n settings.ObservationNormalization) { warnings = append(warnings, n) }
				err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, desired)
				if (err != nil) != test.wantError || !reflect.DeepEqual(operations, []string{"query", "mutation", "query"}) || len(warnings) != test.warnings {
					t.Fatal("raw/identity verification or scoped diagnostics failed", operations, warnings, err)
				}
				for _, warning := range warnings {
					if warning.Path != control.Path() || warning.RawValue != "FUTURE" || warning.Fallback != "all_users" {
						t.Fatal("wrong policy warning", warning)
					}
				}
				if err != nil {
					var pre interface{ ReadFailedBeforeWrite() }
					if errors.As(err, &pre) || !strings.Contains(err.Error(), "may already have applied") {
						t.Fatal("post-write failure mislabeled", err)
					}
				}
			})
		}
	}
}

func TestCreationPolicyApplyAcquisitionAndMutationFailures(t *testing.T) {
	for _, mode := range []string{"preflight partial", "preflight null", "postflight partial", "postflight null", "mutation denied"} {
		t.Run(mode, func(t *testing.T) {
			calls, warnings := 0, 0
			client := prAccessTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 2 {
					if mode == "mutation denied" {
						return labelsResponse(r, 403, `{"message":"fixture denied"}`), nil
					}
					return labelsResponse(r, 200, `{"data":{"updateRepository":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","issueCreationPolicy":"ALL"}}}}`), nil
				}
				if calls == 1 && strings.HasPrefix(mode, "preflight") || calls == 3 && strings.HasPrefix(mode, "postflight") {
					if strings.HasSuffix(mode, "null") {
						return labelsResponse(r, 200, `{"data":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","issueCreationPolicy":null}}}`), nil
					}
					return labelsResponse(r, 200, `{"data":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","issueCreationPolicy":"FUTURE"}},"errors":[{"message":"fixture partial denied"}]}`), nil
				}
				return labelsResponse(r, 200, `{"data":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","issueCreationPolicy":"FUTURE"}}}`), nil
			}))
			client.NormalizationWarning = func(settings.ObservationNormalization) { warnings++ }
			policy := "ALL"
			err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{IssueCreationPolicy: &policy})
			var pre interface{ ReadFailedBeforeWrite() }
			wantPre := strings.HasPrefix(mode, "preflight")
			wantCalls, wantWarnings := 3, 1
			if wantPre {
				wantCalls, wantWarnings = 1, 0
			} else if mode == "mutation denied" {
				wantCalls = 2
			}
			if err == nil || errors.As(err, &pre) != wantPre || calls != wantCalls || warnings != wantWarnings {
				t.Fatal("failure boundary lost", mode, calls, warnings, err)
			}
		})
	}
}

func TestAccessOnlyCheckboxDoesNotAcquireOrWarnPolicies(t *testing.T) {
	calls, warnings := 0, 0
	client := prAccessTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		var request struct {
			Query     string
			Variables map[string]any
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if calls == 1 {
			if request.Query != repositoryAccessIdentityQuery {
				t.Fatal("unmanaged policy acquired", request.Query)
			}
			return labelsResponse(r, 200, `{"data":{"repository":{"id":"repo-id","nameWithOwner":"owner/current"}}}`), nil
		}
		if !reflect.DeepEqual(request.Variables, map[string]any{"input": map[string]any{"repositoryId": "repo-id", "hasPullRequestsEnabled": false}}) {
			t.Fatal("unmanaged policy replayed", request.Variables)
		}
		return labelsResponse(r, 200, `{"data":{"updateRepository":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","hasPullRequestsEnabled":false}}}}`), nil
	}))
	client.NormalizationWarning = func(settings.ObservationNormalization) { warnings++ }
	no := false
	if err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{HasPullRequests: &no}); err != nil || calls != 2 || warnings != 0 {
		t.Fatal("checkbox-only boundary changed", calls, warnings, err)
	}
}
