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

func TestTemplateReadRequiresBoolean(t *testing.T) {
	for _, value := range []string{"true", "false", "null", `"false"`, "0", "[]", "{}", "missing"} {
		t.Run(value, func(t *testing.T) {
			body := `{"default_branch":"main","description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false,"web_commit_signoff_required":false`
			if value != "missing" {
				body += `,"is_template":` + value
			}
			body += "}"
			client := readTestClient(t, func(r *http.Request) (*http.Response, error) { return labelsResponse(r, 200, body), nil })
			got, err := client.ReadRepository(repository.Repository{Owner: "owner", Name: "current"})
			if value == "true" || value == "false" {
				if err != nil || got.IsTemplate != (value == "true") {
					t.Fatalf("state=%+v error=%v", got, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "is_template") || got != (settings.RepositoryState{}) {
				t.Fatalf("incomplete read accepted: state=%+v error=%v", got, err)
			}
		})
	}
}

func TestTemplatePATCHDeclaredValuesOnly(t *testing.T) {
	disabled, enabled := false, true
	for _, test := range []struct {
		name    string
		desired settings.RepositoryDeclaration
		payload map[string]any
		status  int
	}{
		{"omitted", settings.RepositoryDeclaration{HasIssues: &disabled}, map[string]any{"has_issues": false}, 200},
		{"disabled", settings.RepositoryDeclaration{IsTemplate: &disabled}, map[string]any{"is_template": false}, 200},
		{"enabled", settings.RepositoryDeclaration{IsTemplate: &enabled}, map[string]any{"is_template": true}, 200},
		{"mixed", settings.RepositoryDeclaration{HasIssues: &disabled, IsTemplate: &enabled}, map[string]any{"has_issues": false, "is_template": true}, 200},
		{"forbidden", settings.RepositoryDeclaration{IsTemplate: &disabled}, map[string]any{"is_template": false}, 403},
		{"rejected", settings.RepositoryDeclaration{IsTemplate: &enabled}, map[string]any{"is_template": true}, 422},
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
				return labelsResponse(r, test.status, `{}`), nil
			}))
			err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, test.desired)
			if calls != 1 || (err != nil) != (test.status != 200) {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
		})
	}
}
