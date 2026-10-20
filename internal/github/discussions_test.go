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

func TestDiscussionsReadRequiresBoolean(t *testing.T) {
	for _, value := range []string{"true", "false", "null", `"false"`, "0", "[]", "missing"} {
		body := `{"default_branch":"main","description":"","homepage":"","has_issues":false,"has_projects":false,"has_wiki":false,"web_commit_signoff_required":false,"is_template":false`
		if value != "missing" {
			body += `,"has_discussions":` + value
		}
		body += "}"
		client := readTestClient(t, func(r *http.Request) (*http.Response, error) { return labelsResponse(r, 200, body), nil })
		got, err := client.ReadRepository(repository.Repository{Owner: "owner", Name: "current"})
		valid := value == "true" || value == "false"
		if valid {
			if err != nil || got.HasDiscussions != (value == "true") {
				t.Fatalf("value=%s got=%+v err=%v", value, got, err)
			}
		} else if err == nil || !strings.Contains(err.Error(), "has_discussions") || got != (settings.RepositoryState{}) {
			t.Fatalf("incomplete read accepted: value=%s got=%+v err=%v", value, got, err)
		}
	}
}

func TestDiscussionsPATCHVerifiesDeclaredValue(t *testing.T) {
	for _, desired := range []bool{false, true} {
		for _, test := range []struct {
			body, failure string
			status        int
		}{
			{body: `{"has_discussions":false}`, failure: "mismatch"},
			{body: `{"has_discussions":true}`, failure: "mismatch"},
			{body: `{}`, failure: "lacks boolean"},
			{body: `{"has_discussions":null}`, failure: "lacks boolean"},
			{body: `{"has_discussions":"false"}`, failure: "lacks boolean"},
			{body: `{"message":"Resource not accessible"}`, status: 403, failure: "403"},
		} {
			calls := 0
			client := writeTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if r.Method != "PATCH" || r.URL.Path != "/repos/owner/current" || !reflect.DeepEqual(payload, map[string]any{"has_discussions": desired}) {
					t.Fatalf("request=%s %s payload=%v", r.Method, r.URL, payload)
				}
				status := test.status
				if status == 0 {
					status = 200
				}
				return labelsResponse(r, status, test.body), nil
			}))
			err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{HasDiscussions: &desired})
			matching := test.body == `{"has_discussions":false}` && !desired || test.body == `{"has_discussions":true}` && desired
			if calls != 1 || matching && err != nil || !matching && err == nil {
				t.Fatalf("desired=%t response=%s calls=%d err=%v", desired, test.body, calls, err)
			}
			if !matching && test.failure != "mismatch" && !strings.Contains(err.Error(), test.failure) {
				t.Fatalf("error=%v want %s", err, test.failure)
			}
		}
	}
}
