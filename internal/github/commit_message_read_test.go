package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestSelectiveCommitMessageRead(t *testing.T) {
	title, message, description := "PR_TITLE", "BLANK", "scalar"
	for _, test := range []struct {
		name    string
		desired settings.RepositoryDeclaration
		fields  []string
		body    string
		menus   []settings.CommitMessageMenu
	}{
		{"squash", settings.RepositoryDeclaration{SquashMergeCommitTitle: &title}, []string{"squashMergeCommitTitle", "squashMergeCommitMessage"}, `{"data":{"repository":{"squashMergeCommitTitle":"FUTURE_TITLE","squashMergeCommitMessage":""}}}`, []settings.CommitMessageMenu{settings.SquashCommitMenu}},
		{"merge", settings.RepositoryDeclaration{MergeCommitMessage: &message}, []string{"mergeCommitTitle", "mergeCommitMessage"}, `{"data":{"repository":{"mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}}}`, []settings.CommitMessageMenu{settings.MergeCommitMenu}},
		{"both", settings.RepositoryDeclaration{SquashMergeCommitTitle: &title, MergeCommitMessage: &message}, []string{"squashMergeCommitTitle", "squashMergeCommitMessage", "mergeCommitTitle", "mergeCommitMessage"}, `{"data":{"repository":{"squashMergeCommitTitle":"FUTURE_TITLE","squashMergeCommitMessage":"","mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}}}`, []settings.CommitMessageMenu{settings.SquashCommitMenu, settings.MergeCommitMenu}},
		{"scalar", settings.RepositoryDeclaration{Description: &description}, nil, "", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			gql, err := api.NewGraphQLClient(api.ClientOptions{Host: "github.com", AuthToken: "fixture", LogIgnoreEnv: true, Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				var body struct {
					Query     string
					Variables map[string]any
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if r.Method != "POST" || r.URL.Path != "/graphql" || !reflect.DeepEqual(body.Variables, map[string]any{"owner": "owner", "name": "current"}) || !strings.Contains(body.Query, "followRenames: false") || strings.Contains(body.Query, "mutation") {
					t.Fatalf("unexpected query: %s %s %+v", r.Method, r.URL, body)
				}
				repositoryStart := strings.Index(body.Query, "repository(")
				if repositoryStart < 0 {
					t.Fatal("query lacks repository selection", body.Query)
				}
				start := strings.Index(body.Query[repositoryStart:], "{") + repositoryStart
				end := strings.Index(body.Query[start:], "}") + start
				if got := strings.Fields(body.Query[start+1 : end]); !reflect.DeepEqual(got, test.fields) {
					t.Fatalf("selected fields = %v, want %v", got, test.fields)
				}
				return labelsResponse(r, 200, test.body), nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			got, err := (&MergeMethodsClient{graphql: gql}).readCommitMessages(repository.Repository{Owner: "owner", Name: "current"}, test.desired)
			if err != nil || len(got) != len(test.menus) || (len(test.menus) == 0 && calls != 0) || (len(test.menus) > 0 && calls != 1) {
				t.Fatalf("got=%+v err=%v calls=%d", got, err, calls)
			}
			for _, menu := range test.menus {
				pair, valid := got.rawPair(menu)
				want := settings.CommitMessagePair{Title: "MERGE_MESSAGE", Message: "PR_TITLE"}
				if menu == settings.SquashCommitMenu {
					want = settings.CommitMessagePair{Title: "FUTURE_TITLE", Message: ""}
				}
				if !valid || pair != want || (menu == settings.SquashCommitMenu && got[menu].Normalization.Path != menu.Path()) || (menu == settings.MergeCommitMenu && got[menu].Normalization != (settings.CommitMessageNormalization{})) {
					t.Fatalf("raw=%+v valid=%v prepared=%+v", pair, valid, got[menu])
				}
			}
		})
	}
}

func TestSelectiveCommitMessageReadRejectsIncompleteAcquisition(t *testing.T) {
	title := "PR_TITLE"
	desired := settings.RepositoryDeclaration{SquashMergeCommitTitle: &title, MergeCommitTitle: &title}
	for _, test := range []struct {
		name, body     string
		status         int
		transportError bool
	}{
		{"missing member", `{"data":{"repository":{"squashMergeCommitTitle":"FUTURE","squashMergeCommitMessage":"","mergeCommitTitle":"MERGE_MESSAGE"}}}`, 200, false},
		{"null member", `{"data":{"repository":{"squashMergeCommitTitle":"FUTURE","squashMergeCommitMessage":"","mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":null}}}`, 200, false},
		{"boolean member", `{"data":{"repository":{"squashMergeCommitTitle":"FUTURE","squashMergeCommitMessage":"","mergeCommitTitle":true,"mergeCommitMessage":"PR_TITLE"}}}`, 200, false},
		{"numeric member", `{"data":{"repository":{"squashMergeCommitTitle":"FUTURE","squashMergeCommitMessage":"","mergeCommitTitle":2,"mergeCommitMessage":"PR_TITLE"}}}`, 200, false},
		{"partial GraphQL", `{"data":{"repository":{"squashMergeCommitTitle":"FUTURE","squashMergeCommitMessage":"","mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}},"errors":[{"message":"denied"}]}`, 200, false},
		{"missing repository", `{"data":{}}`, 200, false},
		{"null repository", `{"data":{"repository":null}}`, 200, false},
		{"bad JSON", `bad`, 200, false},
		{"HTTP forbidden", `{}`, 403, false},
		{"transport", ``, 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			gql, err := api.NewGraphQLClient(api.ClientOptions{Host: "github.com", AuthToken: "fixture", LogIgnoreEnv: true, Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				if test.transportError {
					return nil, errors.New("fixture transport failure")
				}
				return labelsResponse(r, test.status, test.body), nil
			})})
			if err != nil {
				t.Fatal(err)
			}
			got, err := (&MergeMethodsClient{graphql: gql}).readCommitMessages(repository.Repository{Owner: "owner", Name: "current"}, desired)
			if err == nil || len(got) != 0 {
				t.Fatalf("partial observations returned: %+v err=%v", got, err)
			}
		})
	}
}
