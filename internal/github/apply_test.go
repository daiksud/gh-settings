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

func TestApplyRepositorySettings(t *testing.T) {
	empty, escaped, homepage := "", "quote\"\n\\", "https://example.com"
	disabled, enabled := false, true
	for _, test := range []struct {
		name    string
		desired settings.RepositoryDeclaration
		want    map[string]any
	}{
		{"partial", settings.RepositoryDeclaration{HasIssues: &disabled}, map[string]any{"has_issues": false}},
		{"five zero values", settings.RepositoryDeclaration{Description: &empty, Homepage: &empty, HasIssues: &disabled, HasProjects: &disabled, HasWiki: &disabled}, map[string]any{"description": "", "homepage": "", "has_issues": false, "has_projects": false, "has_wiki": false}},
		{"strings and true", settings.RepositoryDeclaration{Description: &escaped, Homepage: &homepage, HasWiki: &enabled}, map[string]any{"description": escaped, "homepage": homepage, "has_wiki": true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := writeRepositoryFixtureClient(t, test.want, func(req *http.Request) (*http.Response, error) {
				calls++
				var got map[string]any
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if req.Method != http.MethodPatch || req.URL.Path != "/repos/owner/current" || !reflect.DeepEqual(got, test.want) {
					t.Fatalf("request=%s %s body=%v, want PATCH current body=%v", req.Method, req.URL.Path, got, test.want)
				}
				return &http.Response{Request: req, StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{}`))}, nil
			})
			_, err := client.ApplyRepository(repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, test.desired)
			if err != nil || calls != 1 {
				t.Fatalf("error=%v calls=%d, want one PATCH", err, calls)
			}
		})
	}
}

func TestApplyRepositoryFailures(t *testing.T) {
	for _, test := range []struct {
		name, body, want string
		status           int
		transportError   error
	}{
		{name: "permission", body: `{"message":"Resource not accessible"}`, status: 403, want: "403"},
		{name: "server validation", body: `{"message":"Validation Failed"}`, status: 422, want: "422"},
		{name: "redirect", body: `{}`, status: 307, want: "307"},
		{name: "bad response", body: `not json`, status: 200, want: "apply repository settings"},
		{name: "transport", transportError: errors.New("network unavailable"), want: "network unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			patches := 0
			client := writeRepositoryFixtureClient(t, map[string]any{"has_issues": false}, func(req *http.Request) (*http.Response, error) {
				patches++
				if req.Method != http.MethodPatch || req.URL.Path != "/repos/owner/current" {
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL)
				}
				if test.transportError != nil {
					return nil, test.transportError
				}
				response := labelsResponse(req, test.status, test.body)
				if test.status == 307 {
					response.Header.Set("Location", "https://other.test/repository")
				}
				return response, nil
			})
			value := false
			wrote, err := client.ApplyRepository(repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, settings.RepositoryDeclaration{HasIssues: &value})
			var prerequisite interface{ ReadFailedBeforeWrite() }
			if err == nil || !strings.Contains(err.Error(), test.want) || wrote || patches != 1 || errors.As(err, &prerequisite) {
				t.Fatalf("wrote=%v patches=%d error=%v, want uncertain write error %q", wrote, patches, err, test.want)
			}
		})
	}
}

func writeTestClient(t *testing.T, transport transportFunc) *Client {
	t.Helper()
	client, err := newWriteClient(api.ClientOptions{Host: "github.com", AuthToken: "test-token", LogIgnoreEnv: true, Transport: transport})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// Payload/acknowledgment tests use independent complete read fixtures around
// their existing PATCH assertions. Reconciliation failures and exact query
// selection are exercised separately with explicit request sequences.
func writeRepositoryFixtureClient(t *testing.T, patch map[string]any, transport transportFunc) *Client {
	t.Helper()
	base := map[string]any{"description": "before", "homepage": "https://before.test", "default_branch": "main", "has_issues": true, "has_projects": true, "has_wiki": true, "has_discussions": false, "web_commit_signoff_required": false, "is_template": false}
	merge := map[string]any{"mergeCommitAllowed": true, "squashMergeAllowed": true, "rebaseMergeAllowed": true, "deleteBranchOnMerge": false, "autoMergeAllowed": false, "allowUpdateBranch": false, "mergeCommitTitle": "FUTURE_TITLE", "mergeCommitMessage": "FUTURE_MESSAGE", "squashMergeCommitTitle": "FUTURE_TITLE", "squashMergeCommitMessage": "FUTURE_MESSAGE"}
	wire := map[string]string{"allow_merge_commit": "mergeCommitAllowed", "allow_squash_merge": "squashMergeAllowed", "allow_rebase_merge": "rebaseMergeAllowed", "delete_branch_on_merge": "deleteBranchOnMerge", "allow_auto_merge": "autoMergeAllowed", "allow_update_branch": "allowUpdateBranch", "merge_commit_title": "mergeCommitTitle", "merge_commit_message": "mergeCommitMessage", "squash_merge_commit_title": "squashMergeCommitTitle", "squash_merge_commit_message": "squashMergeCommitMessage"}
	for field, value := range patch {
		target, name := base, field
		if graphql, ok := wire[field]; ok {
			target, name = merge, graphql
		}
		if value, ok := value.(bool); ok {
			target[name] = !value
		} else {
			target[name] = "before"
		}
	}
	return writeTestClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method == "GET" && req.URL.Path == "/repos/owner/current" {
			body, _ := json.Marshal(base)
			return labelsResponse(req, 200, string(body)), nil
		}
		if req.Method == "POST" && req.URL.Path == "/graphql" {
			body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": merge}})
			return labelsResponse(req, 200, string(body)), nil
		}
		if req.Method == "PATCH" {
			for field, value := range patch {
				if name, ok := wire[field]; ok {
					merge[name] = value
				} else {
					base[field] = value
				}
			}
		}
		return transport(req)
	})
}
