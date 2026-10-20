package github

import (
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

func TestReadRepositorySettings(t *testing.T) {
	for _, test := range []struct {
		name, body string
		want       settings.RepositoryState
	}{
		{"five values and extra fields", `{"default_branch":"main","description":"Example repository","homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false,"id":123,"private":true}`, settings.RepositoryState{Description: "Example repository", Homepage: "https://example.com", HasIssues: true, HasWiki: true}},
		{"zero values", `{"default_branch":"main","description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, settings.RepositoryState{}},
		{"enabled projects", `{"default_branch":"main","description":"","homepage":"","has_issues":false,"has_projects":true,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, settings.RepositoryState{HasProjects: true}},
		{"null strings", `{"default_branch":"main","description":null,"homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, settings.RepositoryState{}},
		{"null description", `{"default_branch":"main","description":null,"homepage":"","has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, settings.RepositoryState{}},
		{"null homepage", `{"default_branch":"main","description":"","homepage":null,"has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, settings.RepositoryState{}},
		{"null description with values", `{"default_branch":"main","description":null,"homepage":"https://example.com","has_issues":true,"has_projects":false,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, settings.RepositoryState{Homepage: "https://example.com", HasIssues: true, HasWiki: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.want.DefaultBranch = "main"
			client := readTestClient(t, func(req *http.Request) (*http.Response, error) {
				return &http.Response{Request: req, StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})
			got, err := client.ReadRepository(repository.Repository{Host: "github.com", Owner: "owner", Name: "current"})
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("state=%+v error=%v, want %+v", got, err, test.want)
			}
		})
	}
}

func TestReadRepositoryFailures(t *testing.T) {
	for _, test := range []struct {
		name, body, want string
		status           int
		transportError   error
	}{
		{name: "forbidden", body: `{"message":"Resource not accessible"}`, status: 403, want: "403"},
		{name: "malformed", body: "not json", status: 200, want: "read repository settings"},
		{name: "empty response", body: `{}`, status: 200, want: "missing description"},
		{name: "missing description", body: `{"homepage":null,"has_issues":true,"has_projects":false,"has_wiki":true,"has_discussions":false}`, status: 200, want: "missing description"},
		{name: "missing homepage", body: `{"default_branch":"main","description":null,"has_issues":true,"has_projects":false,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, status: 200, want: "missing homepage"},
		{name: "missing issues", body: `{"default_branch":"main","description":"","homepage":"","has_projects":false,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false}`, status: 200, want: "missing has_issues"},
		{name: "missing projects", body: `{"default_branch":"main","description":"","homepage":"","has_issues":true,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, status: 200, want: "missing has_projects"},
		{name: "missing wiki", body: `{"default_branch":"main","description":"ok","homepage":"","has_issues":true,"has_projects":false}`, status: 200, want: "missing has_wiki"},
		{name: "wrong description type", body: `{"description":123}`, status: 200, want: "decode description"},
		{name: "wrong homepage type", body: `{"default_branch":"main","description":"","homepage":123}`, status: 200, want: "decode homepage"},
		{name: "wrong issues type", body: `{"default_branch":"main","description":"ok","homepage":"","has_issues":"true","has_projects":false,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, status: 200, want: "decode has_issues"},
		{name: "wrong projects type", body: `{"default_branch":"main","description":"","homepage":"","has_issues":false,"has_projects":"false","has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, status: 200, want: "decode has_projects"},
		{name: "wrong wiki type", body: `{"default_branch":"main","description":"","homepage":"","has_issues":false,"has_projects":true,"has_wiki":"true","has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, status: 200, want: "decode has_wiki"},
		{name: "null issues", body: `{"default_branch":"main","description":null,"homepage":null,"has_issues":null}`, status: 200, want: "null for has_issues"},
		{name: "null projects", body: `{"default_branch":"main","description":"","homepage":"","has_issues":true,"has_projects":null,"has_wiki":true,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, status: 200, want: "null for has_projects"},
		{name: "null wiki", body: `{"default_branch":"main","description":"","homepage":"","has_issues":true,"has_projects":false,"has_wiki":null,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false}`, status: 200, want: "null for has_wiki"},
		{name: "network", transportError: errors.New("network unavailable"), want: "network unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := readTestClient(t, func(req *http.Request) (*http.Response, error) {
				if test.transportError != nil {
					return nil, test.transportError
				}
				return &http.Response{Request: req, StatusCode: test.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})
			got, err := client.ReadRepository(repository.Repository{Host: "github.com", Owner: "owner", Name: "current"})
			if err == nil || !strings.Contains(err.Error(), test.want) || !reflect.DeepEqual(got, settings.RepositoryState{}) {
				t.Fatalf("state=%+v error=%v, want zero state and %q", got, err, test.want)
			}
		})
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func readTestClient(t *testing.T, transport transportFunc) *Client {
	t.Helper()
	rest, err := api.NewRESTClient(api.ClientOptions{
		Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true,
		Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet || req.URL.Path != "/repos/owner/current" {
				t.Errorf("unexpected request: %s %s", req.Method, req.URL)
			}
			return transport(req)
		}),
	})
	if err != nil {
		t.Fatal(err)
	}
	return &Client{rest: rest}
}
