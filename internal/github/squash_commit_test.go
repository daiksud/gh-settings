package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestSquashCommitReadRejectsIncompleteEnums(t *testing.T) {
	base := map[string]any{"mergeCommitAllowed": false, "squashMergeAllowed": true, "rebaseMergeAllowed": false, "deleteBranchOnMerge": true, "autoMergeAllowed": false, "allowUpdateBranch": false, "squashMergeCommitTitle": "PR_TITLE", "squashMergeCommitMessage": "BLANK", "mergeCommitTitle": "MERGE_MESSAGE", "mergeCommitMessage": "PR_TITLE"}
	for _, field := range []string{"squashMergeCommitTitle", "squashMergeCommitMessage"} {
		for _, value := range []any{nil, true, float64(0), "missing"} {
			t.Run(field+"/"+stringValue(value), func(t *testing.T) {
				fields := make(map[string]any)
				for k, v := range base {
					fields[k] = v
				}
				fields[field] = value
				if value == "missing" {
					delete(fields, field)
				}
				body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": fields}})
				gql, err := api.NewGraphQLClient(api.ClientOptions{Host: "github.com", AuthToken: "fixture", LogIgnoreEnv: true, Transport: transportFunc(func(r *http.Request) (*http.Response, error) { return labelsResponse(r, 200, string(body)), nil })})
				if err != nil {
					t.Fatal(err)
				}
				got, err := (&MergeMethodsClient{graphql: gql}).ReadMergeMethods(repository.Repository{Owner: "owner", Name: "current"})
				if err == nil || !strings.Contains(err.Error(), field) || got != (settings.MergeMethodsState{}) {
					t.Fatalf("got=%+v err=%v", got, err)
				}
			})
		}
	}
}
func stringValue(v any) string { b, _ := json.Marshal(v); return string(b) }

func TestSquashCommitDeclaredOnlyPATCH(t *testing.T) {
	for _, title := range []string{"PR_TITLE", "COMMIT_OR_PR_TITLE"} {
		for _, message := range []string{"", "PR_BODY", "COMMIT_MESSAGES", "BLANK"} {
			desired := settings.RepositoryDeclaration{SquashMergeCommitTitle: &title}
			want := map[string]any{"squash_merge_commit_title": title}
			if message != "" {
				desired.SquashMergeCommitMessage = &message
				want["squash_merge_commit_message"] = message
			}
			calls := 0
			client := writeTestClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == "POST" && r.URL.Path == "/graphql" {
					actualTitle, actualMessage := "FUTURE_TITLE", "FUTURE_MESSAGE"
					if calls > 0 {
						actualTitle = title
						if message != "" {
							actualMessage = message
						}
					}
					body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": map[string]any{"squashMergeCommitTitle": actualTitle, "squashMergeCommitMessage": actualMessage}}})
					return labelsResponse(r, 200, string(body)), nil
				}
				calls++
				var got map[string]any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if r.Method != "PATCH" || r.URL.Path != "/repos/owner/current" || !reflect.DeepEqual(got, want) {
					t.Fatalf("request=%s %s payload=%v want=%v", r.Method, r.URL, got, want)
				}
				return labelsResponse(r, 200, `{}`), nil
			}))
			if err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, desired); err != nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
		}
	}
}
