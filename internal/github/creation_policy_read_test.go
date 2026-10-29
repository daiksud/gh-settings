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

func TestSelectiveCreationPrerequisitesAcquireOnlyManagedPolicies(t *testing.T) {
	all, yes := "ALL", true
	for _, test := range []struct {
		name     string
		desired  settings.RepositoryDeclaration
		fields   []string
		response map[string]any
		want     map[settings.CreationPolicyControl]string
	}{
		{"PR", settings.RepositoryDeclaration{PullRequestCreationPolicy: &all}, []string{"id", "nameWithOwner", "pullRequestCreationPolicy"}, map[string]any{"pullRequestCreationPolicy": "FUTURE"}, map[settings.CreationPolicyControl]string{settings.PullRequestPermissionsControl: "FUTURE"}},
		{"Issue empty", settings.RepositoryDeclaration{IssueCreationPolicy: &all}, []string{"id", "nameWithOwner", "issueCreationPolicy"}, map[string]any{"issueCreationPolicy": ""}, map[settings.CreationPolicyControl]string{settings.IssuePermissionsControl: ""}},
		{"both", settings.RepositoryDeclaration{PullRequestCreationPolicy: &all, IssueCreationPolicy: &all}, []string{"id", "nameWithOwner", "pullRequestCreationPolicy", "issueCreationPolicy"}, map[string]any{"pullRequestCreationPolicy": " \t\n ", "issueCreationPolicy": "COLLABORATORS_ONLY"}, map[settings.CreationPolicyControl]string{settings.PullRequestPermissionsControl: " \t\n ", settings.IssuePermissionsControl: "COLLABORATORS_ONLY"}},
		{"identity only", settings.RepositoryDeclaration{HasPullRequests: &yes}, []string{"id", "nameWithOwner"}, nil, nil},
		{"unrelated REST", settings.RepositoryDeclaration{HasIssues: &yes}, nil, nil, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := prAccessTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				var request struct {
					Query     string
					Variables map[string]any
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				if r.Method != "POST" || r.URL.Path != "/graphql" || !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) || !strings.Contains(request.Query, "followRenames: false") || strings.Contains(request.Query, "mutation") {
					t.Fatal("unexpected acquisition request", r.Method, r.URL, request)
				}
				start := strings.Index(request.Query, "repository(")
				if start < 0 {
					t.Fatal("missing repository selection", request.Query)
				}
				start += strings.Index(request.Query[start:], "{")
				end := start + strings.Index(request.Query[start:], "}")
				if fields := strings.Fields(request.Query[start+1 : end]); !reflect.DeepEqual(fields, test.fields) {
					t.Fatal("query acquired unmanaged fields", fields, test.fields)
				}
				if len(test.fields) == 2 && request.Query != repositoryAccessIdentityQuery {
					t.Fatal("identity-only contract changed", request.Query)
				}
				fields := map[string]any{"id": "repo-id", "nameWithOwner": "owner/current"}
				for name, value := range test.response {
					fields[name] = value
				}
				body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": fields}})
				return labelsResponse(r, 200, string(body)), nil
			}))
			id, got, err := client.readAccessPrerequisites(repository.Repository{Owner: "owner", Name: "current"}, test.desired)
			wantID := "repo-id"
			wantCalls := 1
			if len(test.fields) == 0 {
				wantID = ""
				wantCalls = 0
			}
			if err != nil || id != wantID || len(got) != len(test.want) || calls != wantCalls {
				t.Fatal("selective acquisition failed", id, got, err, calls)
			}
			for control, raw := range test.want {
				value, valid := got.rawValue(control)
				if !valid || value != raw {
					t.Fatal("raw token lost", control, value, valid)
				}
				_, canonical := settings.CreationPolicyChoiceForValue(raw)
				if (got[control].Normalization == (settings.ObservationNormalization{})) != canonical {
					t.Fatal("canonical/fallback distinction lost", got)
				}
				*got[control].Value = "caller mutation"
			}
			if len(test.want) > 0 {
				_, again, err := client.readAccessPrerequisites(repository.Repository{Owner: "owner", Name: "current"}, test.desired)
				if err != nil {
					t.Fatal(err)
				}
				for control, raw := range test.want {
					if value, valid := again.rawValue(control); !valid || value != raw {
						t.Fatal("caller mutation contaminated next acquisition", again)
					}
				}
			}
		})
	}
}

func TestSelectiveCreationPrerequisitesRejectPartialOrUnboundState(t *testing.T) {
	all := "ALL"
	for _, test := range []struct {
		name, body string
		status     int
		transport  bool
	}{
		{"missing policy", `{"data":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","pullRequestCreationPolicy":"FUTURE"}}}`, 200, false},
		{"null policy", `{"data":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","pullRequestCreationPolicy":"FUTURE","issueCreationPolicy":null}}}`, 200, false},
		{"boolean policy", `{"data":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","pullRequestCreationPolicy":"FUTURE","issueCreationPolicy":false}}}`, 200, false},
		{"number policy", `{"data":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","pullRequestCreationPolicy":"FUTURE","issueCreationPolicy":0}}}`, 200, false},
		{"empty identity", `{"data":{"repository":{"id":" ","nameWithOwner":"owner/current","pullRequestCreationPolicy":"FUTURE","issueCreationPolicy":"ALL"}}}`, 200, false},
		{"missing identity", `{"data":{"repository":{"nameWithOwner":"owner/current","pullRequestCreationPolicy":"FUTURE","issueCreationPolicy":"ALL"}}}`, 200, false},
		{"wrong target", `{"data":{"repository":{"id":"repo-id","nameWithOwner":"other/repo","pullRequestCreationPolicy":"FUTURE","issueCreationPolicy":"ALL"}}}`, 200, false},
		{"partial GraphQL", `{"data":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","pullRequestCreationPolicy":"FUTURE","issueCreationPolicy":"ALL"}},"errors":[{"message":"denied"}]}`, 200, false},
		{"missing repository", `{"data":{}}`, 200, false},
		{"null repository", `{"data":{"repository":null}}`, 200, false},
		{"bad JSON", `bad`, 200, false},
		{"HTTP forbidden", `{}`, 403, false},
		{"transport", "", 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := prAccessTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				if test.transport {
					return nil, errors.New("fixture transport failed")
				}
				return labelsResponse(r, test.status, test.body), nil
			}))
			id, got, err := client.readAccessPrerequisites(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{PullRequestCreationPolicy: &all, IssueCreationPolicy: &all})
			if err == nil || id != "" || len(got) != 0 {
				t.Fatal("failed acquisition returned partial/unbound state", id, got, err)
			}
		})
	}
}
