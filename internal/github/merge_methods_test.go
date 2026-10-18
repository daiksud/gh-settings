package github

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestReadMergeMethodsGraphQLBoundary(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
		valid      bool
	}{
		{"values", `{"data":{"repository":{"mergeCommitAllowed":false,"squashMergeAllowed":true,"rebaseMergeAllowed":false,"deleteBranchOnMerge":true,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES"}}}`, 200, true},
		{"all false", `{"data":{"repository":{"mergeCommitAllowed":false,"squashMergeAllowed":false,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES"}}}`, 200, true},
		{"permission", `{"data":{"repository":null},"errors":[{"message":"Resource not accessible by integration"}]}`, 200, false},
		{"partial data with errors", `{"data":{"repository":{"mergeCommitAllowed":true,"squashMergeAllowed":true,"rebaseMergeAllowed":true,"deleteBranchOnMerge":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES"}},"errors":[{"message":"denied"}]}`, 200, false},
		{"missing repository", `{"data":{}}`, 200, false},
		{"null repository", `{"data":{"repository":null}}`, 200, false},
		{"missing field", `{"data":{"repository":{"mergeCommitAllowed":false,"squashMergeAllowed":true}}}`, 200, false},
		{"null field", `{"data":{"repository":{"mergeCommitAllowed":null,"squashMergeAllowed":true,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES"}}}`, 200, false},
		{"string field", `{"data":{"repository":{"mergeCommitAllowed":"false","squashMergeAllowed":true,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES"}}}`, 200, false},
		{"number field", `{"data":{"repository":{"mergeCommitAllowed":0,"squashMergeAllowed":true,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES"}}}`, 200, false},
		{"missing deletion field", `{"data":{"repository":{"mergeCommitAllowed":false,"squashMergeAllowed":true,"rebaseMergeAllowed":false}}}`, 200, false},
		{"null deletion field", `{"data":{"repository":{"mergeCommitAllowed":false,"squashMergeAllowed":true,"rebaseMergeAllowed":false,"deleteBranchOnMerge":null,"squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES"}}}`, 200, false},
		{"wrong deletion type", `{"data":{"repository":{"mergeCommitAllowed":false,"squashMergeAllowed":true,"rebaseMergeAllowed":false,"deleteBranchOnMerge":"false","squashMergeCommitTitle":"COMMIT_OR_PR_TITLE","squashMergeCommitMessage":"COMMIT_MESSAGES"}}}`, 200, false},
		{"bad JSON", `not json`, 200, false},
		{"empty response", ``, 204, false},
		{"forbidden", `{"message":"denied"}`, 403, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			gql, err := api.NewGraphQLClient(api.ClientOptions{Host: "github.com", AuthToken: "fixture-token", LogIgnoreEnv: true, Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				var body struct {
					Query     string         `json:"query"`
					Variables map[string]any `json:"variables"`
				}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if req.Method != "POST" || req.URL.Path != "/graphql" || body.Query != mergeMethodsQuery || !reflect.DeepEqual(body.Variables, map[string]any{"owner": "owner", "name": "current"}) || strings.Contains(body.Query, "mutation") {
					t.Fatalf("request=%s %s body=%+v", req.Method, req.URL, body)
				}
				return &http.Response{Request: req, StatusCode: test.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			client := &MergeMethodsClient{graphql: gql}
			got, err := client.ReadMergeMethods(repository.Repository{Owner: "owner", Name: "current"})
			if calls != 1 || test.valid != (err == nil) || (!test.valid && got != (settings.MergeMethodsState{})) {
				t.Fatalf("state=%+v error=%v calls=%d", got, err, calls)
			}
			if test.name == "values" && got != (settings.MergeMethodsState{AllowSquashMerge: true, DeleteBranchOnMerge: true, SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES"}) {
				t.Fatalf("state=%+v", got)
			}
		})
	}
}

func TestApplyMergeMethodPayload(t *testing.T) {
	yes, no := true, false
	for _, test := range []struct {
		desired settings.RepositoryDeclaration
		want    map[string]any
	}{
		{settings.RepositoryDeclaration{AllowMergeCommit: &no}, map[string]any{"allow_merge_commit": false}},
		{settings.RepositoryDeclaration{AllowSquashMerge: &no}, map[string]any{"allow_squash_merge": false}},
		{settings.RepositoryDeclaration{AllowRebaseMerge: &yes}, map[string]any{"allow_rebase_merge": true}},
		{settings.RepositoryDeclaration{DeleteBranchOnMerge: &no}, map[string]any{"delete_branch_on_merge": false}},
		{settings.RepositoryDeclaration{DeleteBranchOnMerge: &yes}, map[string]any{"delete_branch_on_merge": true}},
		{settings.RepositoryDeclaration{HasIssues: &no, AllowMergeCommit: &no, AllowSquashMerge: &yes, AllowRebaseMerge: &no}, map[string]any{"has_issues": false, "allow_merge_commit": false, "allow_squash_merge": true, "allow_rebase_merge": false}},
	} {
		calls := 0
		client := writeTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			var body map[string]any
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if req.Method != "PATCH" || req.URL.Path != "/repos/owner/current" || !reflect.DeepEqual(body, test.want) {
				t.Fatalf("request=%s %s body=%v want=%v", req.Method, req.URL, body, test.want)
			}
			return labelsResponse(req, 200, `{}`), nil
		}))
		if err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, test.desired); err != nil || calls != 1 {
			t.Fatalf("calls=%d err=%v", calls, err)
		}
	}
}
