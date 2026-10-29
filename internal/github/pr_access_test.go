package github

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func prAccessTestClient(t *testing.T, transport http.RoundTripper) *RepositoryAccessClient {
	t.Helper()
	gql, err := api.NewGraphQLClient(api.ClientOptions{Host: "github.com", AuthToken: "dummy-token", LogIgnoreEnv: true, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	if err != nil {
		t.Fatal(err)
	}
	return &RepositoryAccessClient{graphql: gql}
}
func TestReadPRAccessCompleteAndRejectsUnknownData(t *testing.T) {
	const valid = `{"data":{"repository":{"hasPullRequestsEnabled":false,"pullRequestCreationPolicy":"ALL"}}}`
	for _, body := range []string{valid, strings.Replace(valid, `"hasPullRequestsEnabled":false,`, "", 1), strings.Replace(valid, `false`, `null`, 1), strings.Replace(valid, `false`, `"false"`, 1), strings.Replace(valid, `"ALL"`, `null`, 1), strings.Replace(valid, `"ALL"`, `false`, 1), strings.Replace(valid, `"ALL"`, `0`, 1), `{"data":{"repository":null}}`, `{"data":{"repository":{"hasPullRequestsEnabled":false,"pullRequestCreationPolicy":"ALL"}},"errors":[{"message":"partial data denied"}]}`} {
		client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			data, _ := io.ReadAll(req.Body)
			var request struct {
				Query     string
				Variables map[string]any
			}
			json.Unmarshal(data, &request)
			if req.Method != "POST" || req.URL.Path != "/graphql" || strings.Contains(request.Query, "mutation") || !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) {
				t.Fatal(string(data))
			}
			return labelsResponse(req, 200, body), nil
		}))
		got, err := client.ReadPRAccess(repository.Repository{Owner: "owner", Name: "current"})
		if body == valid {
			if err != nil || got.HasPullRequests || got.PullRequestCreationPolicy != "ALL" {
				t.Fatal(got, err)
			}
		} else if err == nil || got != (settings.PRAccessState{}) {
			t.Fatal("incomplete policy accepted", body, got, err)
		}
	}
}
func TestPRAccessWriteOnlyDeclaredValues(t *testing.T) {
	no := false
	all, limited := "ALL", "COLLABORATORS_ONLY"
	for _, desired := range []settings.RepositoryDeclaration{{HasPullRequests: &no}, {PullRequestCreationPolicy: &all}, {PullRequestCreationPolicy: &limited}, {HasPullRequests: &no, PullRequestCreationPolicy: &limited}} {
		calls := 0
		client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			data, _ := io.ReadAll(req.Body)
			var request struct {
				Query     string
				Variables map[string]any
			}
			json.Unmarshal(data, &request)
			if calls == 1 || calls == 3 {
				if strings.Contains(request.Query, "mutation") || !reflect.DeepEqual(request.Variables, map[string]any{"owner": "owner", "name": "current"}) {
					t.Fatal(string(data))
				}
				fields := map[string]any{"id": "opaque-id", "nameWithOwner": "OWNER/CURRENT"}
				if desired.PullRequestCreationPolicy != nil {
					value := "ALL"
					if calls == 3 {
						value = *desired.PullRequestCreationPolicy
					}
					fields["pullRequestCreationPolicy"] = value
				}
				body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": fields}})
				return labelsResponse(req, 200, string(body)), nil
			}
			want := map[string]any{"repositoryId": "opaque-id"}
			response := map[string]any{"id": "opaque-id", "nameWithOwner": "owner/current"}
			if desired.HasPullRequests != nil {
				want["hasPullRequestsEnabled"] = false
				response["hasPullRequestsEnabled"] = false
			} else if strings.Contains(request.Query, "hasPullRequestsEnabled") {
				t.Fatal("unmanaged enabled field requested")
			}
			if desired.PullRequestCreationPolicy != nil {
				want["pullRequestCreationPolicy"] = *desired.PullRequestCreationPolicy
				response["pullRequestCreationPolicy"] = *desired.PullRequestCreationPolicy
			} else if strings.Contains(request.Query, "pullRequestCreationPolicy") {
				t.Fatal("unmanaged companion requested")
			}
			if calls != 2 || !strings.Contains(request.Query, "mutation($input: UpdateRepositoryInput!)") || !reflect.DeepEqual(request.Variables, map[string]any{"input": want}) {
				t.Fatal("excess mutation values", string(data), want)
			}
			body, _ := json.Marshal(map[string]any{"data": map[string]any{"updateRepository": map[string]any{"repository": response}}})
			return labelsResponse(req, 200, string(body)), nil
		}))
		wantCalls := 2
		if desired.PullRequestCreationPolicy != nil {
			wantCalls = 3
		}
		if err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, desired); err != nil || calls != wantCalls {
			t.Fatal(calls, err)
		}
	}
	client := prAccessTestClient(t, transportFunc(func(*http.Request) (*http.Response, error) { t.Fatal("empty policy requested API"); return nil, nil }))
	if err := client.ApplyRepositoryAccess(repository.Repository{}, settings.RepositoryDeclaration{}); err != nil {
		t.Fatal(err)
	}
}
func TestPRAccessIdentityAndMutationFailures(t *testing.T) {
	const lookup = `{"data":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current","pullRequestCreationPolicy":"ALL"}}}`
	const applied = `{"data":{"updateRepository":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current","hasPullRequestsEnabled":false,"pullRequestCreationPolicy":"ALL"}}}}`
	for _, tt := range []struct {
		name, first, second string
		calls               int
	}{
		{"identity missing", `{"data":{"repository":{"nameWithOwner":"owner/current"}}}`, applied, 1},
		{"identity another repository", strings.Replace(lookup, "owner/current", "other/repo", 1), applied, 1},
		{"identity null", `{"data":{"repository":null}}`, applied, 1},
		{"identity partial error", `{"data":{"repository":{"id":"opaque-id","nameWithOwner":"owner/current"}},"errors":[{"message":"lookup denied"}]}`, applied, 1},
		{"mutation null", lookup, `{"data":{"updateRepository":null}}`, 2},
		{"ignored false", lookup, strings.Replace(applied, "false", "true", 1), 2},
		{"missing value", lookup, strings.Replace(applied, `"hasPullRequestsEnabled":false,`, "", 1), 2},
		{"unknown policy", lookup, strings.Replace(applied, `"ALL"`, `"UNKNOWN"`, 1), 2},
		{"another ID", lookup, strings.Replace(applied, "opaque-id", "another-id", 1), 2},
		{"another target", lookup, strings.Replace(applied, "owner/current", "other/repo", 1), 2},
		{"partial mutation error", lookup, strings.TrimSuffix(applied, "}") + `,"errors":[{"message":"write failed after partial data"}]}`, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				body := tt.first
				if calls > 1 {
					body = tt.second
				}
				return labelsResponse(req, 200, body), nil
			}))
			no := false
			policy := "ALL"
			err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{HasPullRequests: &no, PullRequestCreationPolicy: &policy})
			if err == nil || calls != tt.calls || (calls == 2 && !strings.Contains(err.Error(), "writes may already have applied")) {
				t.Fatal(calls, err)
			}
		})
	}
}
func TestPRAccessReadSelectionAndPartialDiscard(t *testing.T) {
	fields := observationFields(t, "repository.has_pull_requests")
	calls := 0
	observed, err := observeRepository(repository.Repository{}, fields, nil, nil, func(repository.Repository) (settings.PRAccessState, error) {
		calls++
		return settings.PRAccessState{PullRequestCreationPolicy: "ALL"}, nil
	}, nil, nil)
	if err != nil || calls != 1 || len(observed.Fields()) != 2 {
		t.Fatal(observed, calls, err)
	}
	cause := errors.New("PR read denied")
	observed, err = observeRepository(repository.Repository{}, observationFields(t, "repository.has_issues", "repository.pull_request_creation_policy"), func(repository.Repository) (settings.RepositoryState, error) {
		return settings.RepositoryState{HasIssues: true}, nil
	}, nil, func(repository.Repository) (settings.PRAccessState, error) { return settings.PRAccessState{}, cause }, nil, nil)
	if err != cause || len(observed.Fields()) != 0 {
		t.Fatal("earlier comparison leaked", observed, err)
	}
}

func TestInvalidPRAccessWriteFailsBeforeLookup(t *testing.T) {
	client := prAccessTestClient(t, transportFunc(func(*http.Request) (*http.Response, error) { t.Fatal("invalid enum reached API"); return nil, nil }))
	value := "UNKNOWN"
	if err := client.ApplyRepositoryAccess(repository.Repository{}, settings.RepositoryDeclaration{PullRequestCreationPolicy: &value}); err == nil {
		t.Fatal("unknown policy accepted")
	}
}

func TestPRAccessQueriesRejectRenamedRepository(t *testing.T) {
	for _, operation := range []string{"read", "lookup"} {
		t.Run(operation, func(t *testing.T) {
			calls := 0
			client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				data, _ := io.ReadAll(req.Body)
				var request struct{ Query string }
				json.Unmarshal(data, &request)
				if strings.Contains(request.Query, "followRenames: false") {
					return labelsResponse(req, 200, `{"data":{"repository":null},"errors":[{"message":"stale checkout origin"}]}`), nil
				}
				if operation == "lookup" {
					t.Error("identity lookup followed repository rename")
					return labelsResponse(req, 200, `{"data":{"repository":{"id":"renamed-id","nameWithOwner":"owner/renamed"}}}`), nil
				}
				// GitHub's default would silently resolve the old name to this policy.
				return labelsResponse(req, 200, `{"data":{"repository":{"hasPullRequestsEnabled":false,"pullRequestCreationPolicy":"ALL"}}}`), nil
			}))
			var err error
			repo := repository.Repository{Owner: "owner", Name: "current"}
			if operation == "read" {
				_, err = client.ReadPRAccess(repo)
			} else {
				no := false
				err = client.ApplyRepositoryAccess(repo, settings.RepositoryDeclaration{HasPullRequests: &no})
			}
			if err == nil || calls != 1 || !strings.Contains(err.Error(), "stale checkout origin") {
				t.Fatal("renamed repository accepted", calls, err)
			}
		})
	}
}
