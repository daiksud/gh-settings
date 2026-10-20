package github

import (
	"encoding/json"
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestReadIssueCreationPolicyCompleteAndSelective(t *testing.T) {
	for _, value := range []string{`"ALL"`, `"COLLABORATORS_ONLY"`, `null`, `"UNKNOWN"`, `"all"`, `false`, `0`} {
		client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			data, _ := io.ReadAll(req.Body)
			var request struct {
				Query     string
				Variables map[string]any
			}
			json.Unmarshal(data, &request)
			if req.Method != "POST" || req.URL.Path != "/graphql" || !strings.Contains(request.Query, "followRenames: false") || strings.Contains(request.Query, "mutation") || strings.Contains(request.Query, "hasPullRequestsEnabled") || strings.Contains(request.Query, "hasIssuesEnabled") || !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) {
				t.Fatal("unrelated read or rename following", string(data))
			}
			return labelsResponse(req, 200, `{"data":{"repository":{"issueCreationPolicy":`+value+`}}}`), nil
		}))
		state, err := client.ReadIssueCreationPolicy(repository.Repository{Owner: "owner", Name: "current"})
		valid := value == `"ALL"` || value == `"COLLABORATORS_ONLY"`
		if (err == nil) != valid || (valid && state.Policy != strings.Trim(value, `"`)) || (!valid && state != (settings.IssueCreationPolicyState{})) {
			t.Fatal(value, state, err)
		}
	}
	for _, body := range []string{`{"data":{"repository":{}}}`, `{"data":{"repository":null}}`, `{"data":{"repository":{"issueCreationPolicy":"ALL"}},"errors":[{"message":"policy hidden"}]}`} {
		client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, body), nil }))
		state, err := client.ReadIssueCreationPolicy(repository.Repository{})
		if err == nil || state != (settings.IssueCreationPolicyState{}) {
			t.Fatal("partial policy accepted", state, err)
		}
	}
}
func TestIssuePolicyWriteOnlyDeclaredValuesAndSharedMutation(t *testing.T) {
	no := false
	all, limited := "ALL", "COLLABORATORS_ONLY"
	for _, desired := range []settings.RepositoryDeclaration{{IssueCreationPolicy: &all}, {IssueCreationPolicy: &limited}, {HasPullRequests: &no, PullRequestCreationPolicy: &limited, IssueCreationPolicy: &all}} {
		calls := 0
		client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			data, _ := io.ReadAll(req.Body)
			var request struct {
				Query     string
				Variables map[string]any
			}
			json.Unmarshal(data, &request)
			if calls == 1 {
				if !strings.Contains(request.Query, "followRenames: false") || strings.Contains(request.Query, "issueCreationPolicy") || strings.Contains(request.Query, "hasIssuesEnabled") {
					t.Fatal("companion lookup or rename following", string(data))
				}
				return labelsResponse(req, 200, `{"data":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current"}}}`), nil
			}
			want := map[string]any{"repositoryId": "opaque-id", "issueCreationPolicy": *desired.IssueCreationPolicy}
			result := map[string]any{"id": "opaque-id", "nameWithOwner": "owner/current", "issueCreationPolicy": *desired.IssueCreationPolicy}
			if desired.HasPullRequests != nil {
				want["hasPullRequestsEnabled"] = false
				result["hasPullRequestsEnabled"] = false
			}
			if desired.PullRequestCreationPolicy != nil {
				want["pullRequestCreationPolicy"] = *desired.PullRequestCreationPolicy
				result["pullRequestCreationPolicy"] = *desired.PullRequestCreationPolicy
			}
			if calls != 2 || strings.Contains(request.Query, "hasIssuesEnabled") || !reflect.DeepEqual(request.Variables, map[string]any{"input": want}) {
				t.Fatal("inferred availability or excess mutation", string(data), want)
			}
			body, _ := json.Marshal(map[string]any{"data": map[string]any{"updateRepository": map[string]any{"repository": result}}})
			return labelsResponse(req, 200, string(body)), nil
		}))
		if err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, desired); err != nil || calls != 2 {
			t.Fatal(calls, err)
		}
	}
}
func TestIssuePolicyMutationRequiresMatchingResponse(t *testing.T) {
	for _, value := range []string{`"ALL"`, `null`, `"UNKNOWN"`, `false`, `MISSING`, `PARTIAL_ERROR`} {
		calls := 0
		client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return labelsResponse(req, 200, `{"data":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current"}}}`), nil
			}
			field := `,"issueCreationPolicy":` + value
			if value == "MISSING" {
				field = ""
			}
			suffix := ""
			if value == "PARTIAL_ERROR" {
				field = `,"issueCreationPolicy":"COLLABORATORS_ONLY"`
				suffix = `,"errors":[{"message":"partial mutation denied"}]`
			}
			return labelsResponse(req, 200, `{"data":{"updateRepository":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current"`+field+`}}}`+suffix+`}`), nil
		}))
		policy := "COLLABORATORS_ONLY"
		err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{IssueCreationPolicy: &policy})
		if err == nil || calls != 2 || !strings.Contains(err.Error(), "writes may already have applied") {
			t.Fatal(value, calls, err)
		}
		if value == "PARTIAL_ERROR" && !strings.Contains(err.Error(), "partial mutation denied") {
			t.Fatal("fixture did not exercise GraphQL partial errors", err)
		}
	}
	invalid := "UNKNOWN"
	client := prAccessTestClient(t, transportFunc(func(*http.Request) (*http.Response, error) { t.Fatal("invalid policy reached API"); return nil, nil }))
	if err := client.ApplyRepositoryAccess(repository.Repository{}, settings.RepositoryDeclaration{IssueCreationPolicy: &invalid}); err == nil {
		t.Fatal("invalid enum accepted")
	}
}
func TestIssuePolicyObservationRoutingAndPartialDiscard(t *testing.T) {
	observed, err := observeRepository(repository.Repository{}, observationFields(t, "repository.issue_creation_policy"), nil, nil, nil, func(repository.Repository) (settings.IssueCreationPolicyState, error) {
		return settings.IssueCreationPolicyState{Policy: "ALL"}, nil
	})
	if err != nil || len(observed.Fields()) != 1 {
		t.Fatal(observed, err)
	}
	cause := errors.New("Issue policy hidden")
	observed, err = observeRepository(repository.Repository{}, observationFields(t, "repository.has_issues", "repository.issue_creation_policy"), func(repository.Repository) (settings.RepositoryState, error) {
		return settings.RepositoryState{HasIssues: true}, nil
	}, nil, nil, func(repository.Repository) (settings.IssueCreationPolicyState, error) {
		return settings.IssueCreationPolicyState{}, cause
	})
	if err != cause || len(observed.Fields()) != 0 {
		t.Fatal("partial comparison escaped", observed, err)
	}
}
