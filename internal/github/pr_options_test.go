package github

import (
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"net/http"
	"strings"
	"testing"
)

func TestPROptionsCompleteGraphQLRead(t *testing.T) {
	for _, field := range []string{"autoMergeAllowed", "allowUpdateBranch"} {
		for _, value := range []any{true, false, nil, "false", 0, "missing"} {
			fields := map[string]any{"mergeCommitAllowed": false, "squashMergeAllowed": true, "rebaseMergeAllowed": false, "deleteBranchOnMerge": true, "autoMergeAllowed": true, "allowUpdateBranch": true, "squashMergeCommitTitle": "COMMIT_OR_PR_TITLE", "squashMergeCommitMessage": "COMMIT_MESSAGES", "mergeCommitTitle": "MERGE_MESSAGE", "mergeCommitMessage": "PR_TITLE"}
			fields[field] = value
			if value == "missing" {
				delete(fields, field)
			}
			body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": fields}})
			calls := 0
			gql, err := api.NewGraphQLClient(api.ClientOptions{Host: "github.com", AuthToken: "fixture", LogIgnoreEnv: true, Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return labelsResponse(r, 200, string(body)), nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			got, err := (&MergeMethodsClient{graphql: gql}).ReadMergeMethods(repository.Repository{Owner: "owner", Name: "current"})
			valid := value == true || value == false
			if calls != 1 || valid != (err == nil) {
				t.Fatalf("%s=%v got=%+v err=%v calls=%d", field, value, got, err, calls)
			}
			if valid {
				before := got.AllowAutoMerge
				if field == "allowUpdateBranch" {
					before = got.AllowUpdateBranch
				}
				if before != value {
					t.Fatalf("lost bool %+v", got)
				}
			} else if got != (settings.MergeMethodsState{}) || !strings.Contains(err.Error(), field) {
				t.Fatalf("partial=%+v err=%v", got, err)
			}
		}
	}
}
