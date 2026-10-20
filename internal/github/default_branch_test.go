package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestDefaultBranchReadRequiresValidString(t *testing.T) {
	for _, value := range []string{`"main"`, `"release/v2"`, `""`, `"a..b"`, "null", "false", "0", "[]", "{}", "missing"} {
		t.Run(value, func(t *testing.T) {
			body := `{"description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false,"is_template":false`
			if value != "missing" {
				body += `,"default_branch":` + value
			}
			body += "}"
			client := readTestClient(t, func(r *http.Request) (*http.Response, error) { return labelsResponse(r, 200, body), nil })
			got, err := client.ReadRepository(repository.Repository{Owner: "owner", Name: "current"})
			if value == `"main"` || value == `"release/v2"` {
				var branch string
				if err := json.Unmarshal([]byte(value), &branch); err != nil {
					t.Fatal(err)
				}
				if err != nil || got.DefaultBranch != branch {
					t.Fatalf("state=%+v error=%v", got, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "default_branch") || got != (settings.RepositoryState{}) {
				t.Fatalf("invalid read accepted: %+v error=%v", got, err)
			}
		})
	}
}

func TestDefaultBranchPATCHScopeAndResponse(t *testing.T) {
	branch, disabled := "release/v2", false
	for _, test := range []struct {
		name              string
		desired           settings.RepositoryDeclaration
		payload           map[string]any
		response, failure string
		status            int
	}{
		{"omitted", settings.RepositoryDeclaration{HasIssues: &disabled}, map[string]any{"has_issues": false}, `{}`, "", 200},
		{"existing branch", settings.RepositoryDeclaration{DefaultBranch: &branch}, map[string]any{"default_branch": "release/v2"}, `{"default_branch":"release/v2"}`, "", 200},
		{"mixed", settings.RepositoryDeclaration{DefaultBranch: &branch, HasIssues: &disabled}, map[string]any{"default_branch": "release/v2", "has_issues": false}, `{"default_branch":"release/v2"}`, "", 200},
		{"discussions too", settings.RepositoryDeclaration{DefaultBranch: &branch, HasDiscussions: &disabled}, map[string]any{"default_branch": "release/v2", "has_discussions": false}, `{"default_branch":"release/v2","has_discussions":false}`, "", 200},
		{"missing", settings.RepositoryDeclaration{DefaultBranch: &branch}, map[string]any{"default_branch": "release/v2"}, `{}`, "lacks valid default_branch", 200},
		{"null", settings.RepositoryDeclaration{DefaultBranch: &branch}, map[string]any{"default_branch": "release/v2"}, `{"default_branch":null}`, "lacks valid default_branch", 200},
		{"wrong type", settings.RepositoryDeclaration{DefaultBranch: &branch}, map[string]any{"default_branch": "release/v2"}, `{"default_branch":false}`, "lacks valid default_branch", 200},
		{"empty", settings.RepositoryDeclaration{DefaultBranch: &branch}, map[string]any{"default_branch": "release/v2"}, `{"default_branch":""}`, "lacks valid default_branch", 200},
		{"invalid", settings.RepositoryDeclaration{DefaultBranch: &branch}, map[string]any{"default_branch": "release/v2"}, `{"default_branch":"a..b"}`, "lacks valid default_branch", 200},
		{"ignored", settings.RepositoryDeclaration{DefaultBranch: &branch}, map[string]any{"default_branch": "release/v2"}, `{"default_branch":"main"}`, "expected", 200},
		{"forbidden", settings.RepositoryDeclaration{DefaultBranch: &branch}, map[string]any{"default_branch": "release/v2"}, `{}`, "403", 403},
		{"branch not found", settings.RepositoryDeclaration{DefaultBranch: &branch}, map[string]any{"default_branch": "release/v2"}, `{"message":"branch not found"}`, "422", 422},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := writeTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if r.Method != "PATCH" || r.URL.Path != "/repos/owner/current" || !reflect.DeepEqual(payload, test.payload) {
					t.Fatalf("request=%s %s payload=%v, want %v", r.Method, r.URL, payload, test.payload)
				}
				return labelsResponse(r, test.status, test.response), nil
			}))
			err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, test.desired)
			if calls != 1 || test.failure == "" && err != nil || test.failure != "" && (err == nil || !strings.Contains(err.Error(), test.failure)) {
				t.Fatalf("calls=%d error=%v want %q", calls, err, test.failure)
			}
		})
	}
}
