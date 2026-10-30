package github

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestCompleteMergeReadNormalizesTypedMenuPairs(t *testing.T) {
	for _, test := range []struct {
		title, message string
		normalized     bool
	}{
		{"PR_TITLE", "BLANK", false},
		{"FUTURE_TITLE", "FUTURE_MESSAGE", true},
		{"", "", true},
	} {
		fields := map[string]any{
			"mergeCommitAllowed": true, "squashMergeAllowed": true, "rebaseMergeAllowed": false, "deleteBranchOnMerge": false, "autoMergeAllowed": false, "allowUpdateBranch": false,
			"mergeCommitTitle": test.title, "mergeCommitMessage": test.message, "squashMergeCommitTitle": test.title, "squashMergeCommitMessage": test.message,
		}
		body, _ := json.Marshal(map[string]any{"data": map[string]any{"repository": fields}})
		gql, err := api.NewGraphQLClient(api.ClientOptions{Host: "github.com", AuthToken: "fixture", LogIgnoreEnv: true, Transport: transportFunc(func(r *http.Request) (*http.Response, error) { return labelsResponse(r, 200, string(body)), nil })})
		if err != nil {
			t.Fatal(err)
		}
		got, err := (&MergeMethodsClient{graphql: gql}).ReadMergeMethods(repository.Repository{Owner: "owner", Name: "current"})
		if err != nil || !got.AllowMergeCommit || !got.AllowSquashMerge || got.AllowRebaseMerge || got.DeleteBranchOnMerge || got.AllowAutoMerge || got.AllowUpdateBranch {
			t.Fatal("complete GET failed or booleans changed", got, err)
		}
		if test.normalized {
			if got.MergeCommitTitle != "MERGE_MESSAGE" || got.MergeCommitMessage != "PR_TITLE" || got.SquashMergeCommitTitle != "COMMIT_OR_PR_TITLE" || got.SquashMergeCommitMessage != "COMMIT_MESSAGES" {
				t.Fatal("wrong effective pairs", got)
			}
			for menu, n := range map[settings.CommitMessageMenu]settings.CommitMessageNormalization{settings.MergeCommitMenu: got.MergeCommitNormalization, settings.SquashCommitMenu: got.SquashCommitNormalization} {
				if n.Path != menu.Path() || n.RawTitle != test.title || n.RawMessage != test.message || n.Fallback != "default_message" {
					t.Fatal("lost original pair", menu, n)
				}
			}
		} else if got.MergeCommitTitle != test.title || got.MergeCommitMessage != test.message || got.SquashMergeCommitTitle != test.title || got.SquashMergeCommitMessage != test.message || got.MergeCommitNormalization != (settings.CommitMessageNormalization{}) || got.SquashCommitNormalization != (settings.CommitMessageNormalization{}) {
			t.Fatal("canonical pair changed", got)
		}
	}
}
