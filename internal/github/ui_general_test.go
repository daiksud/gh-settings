package github

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIGeneralScalars(t *testing.T) {
	for _, value := range []bool{false, true} {
		for _, test := range []struct {
			name string
			ui   uimodel.General
			want settings.RepositoryDeclaration
		}{
			{"omitted", uimodel.General{}, settings.RepositoryDeclaration{}},
			{"template", uimodel.General{TemplateRepository: &value}, settings.RepositoryDeclaration{IsTemplate: &value}},
			{"default branch", uimodel.General{DefaultBranch: pointer("trunk")}, settings.RepositoryDeclaration{DefaultBranch: pointer("trunk")}},
			{"wikis", uimodel.General{Features: uimodel.Features{Wikis: &value}}, settings.RepositoryDeclaration{HasWiki: &value}},
			{"issues", uimodel.General{Features: uimodel.Features{Issues: &value}}, settings.RepositoryDeclaration{HasIssues: &value}},
			{"projects", uimodel.General{Features: uimodel.Features{Projects: &value}}, settings.RepositoryDeclaration{HasProjects: &value}},
			{"discussions", uimodel.General{Features: uimodel.Features{Discussions: &value}}, settings.RepositoryDeclaration{HasDiscussions: &value}},
			{"sponsorships", uimodel.General{Features: uimodel.Features{Sponsorships: &value}}, settings.RepositoryDeclaration{HasSponsorships: &value}},
			{"pull requests", uimodel.General{Features: uimodel.Features{PullRequests: &value}}, settings.RepositoryDeclaration{HasPullRequests: &value}},
			{"merge", uimodel.General{PullRequests: uimodel.PullRequests{AllowMergeCommits: &value}}, settings.RepositoryDeclaration{AllowMergeCommit: &value}},
			{"squash", uimodel.General{PullRequests: uimodel.PullRequests{AllowSquashMerging: &value}}, settings.RepositoryDeclaration{AllowSquashMerge: &value}},
			{"rebase", uimodel.General{PullRequests: uimodel.PullRequests{AllowRebaseMerging: &value}}, settings.RepositoryDeclaration{AllowRebaseMerge: &value}},
			{"update", uimodel.General{PullRequests: uimodel.PullRequests{AlwaysSuggestUpdatingPullRequestBranches: &value}}, settings.RepositoryDeclaration{AllowUpdateBranch: &value}},
			{"auto merge", uimodel.General{PullRequests: uimodel.PullRequests{AllowAutoMerge: &value}}, settings.RepositoryDeclaration{AllowAutoMerge: &value}},
			{"delete branches", uimodel.General{PullRequests: uimodel.PullRequests{AutomaticallyDeleteHeadBranches: &value}}, settings.RepositoryDeclaration{DeleteBranchOnMerge: &value}},
			{"sign off", uimodel.General{Commits: uimodel.Commits{RequireContributorsToSignOffOnWebBasedCommits: &value}}, settings.RepositoryDeclaration{WebCommitSignoffRequired: &value}},
			{"archive", uimodel.General{DangerZone: uimodel.DangerZone{ArchiveThisRepository: &value}}, settings.RepositoryDeclaration{Archived: &value}},
		} {
			t.Run(test.name, func(t *testing.T) {
				if got, err := GeneralObservation(test.want); err != nil || !reflect.DeepEqual(got.General, test.ui) {
					t.Fatalf("observation=%+v error=%v want %+v", got, err, test.ui)
				}
				got, err := GeneralDeclaration(test.ui)
				if err != nil || !reflect.DeepEqual(got, test.want) {
					t.Fatalf("declaration=%+v error=%v want %+v", got, err, test.want)
				}
			})
		}
	}
}

func TestUIGeneralInvalidBranch(t *testing.T) {
	if _, err := GeneralDeclaration(uimodel.General{DefaultBranch: pointer("bad..branch")}); err == nil {
		t.Fatal("invalid branch accepted")
	}
}

func TestUIGeneralRESTPayload(t *testing.T) {
	want := map[string]any{"is_template": true, "default_branch": "trunk", "has_issues": false, "has_projects": true, "has_wiki": false, "has_discussions": true, "web_commit_signoff_required": true, "allow_merge_commit": false, "allow_squash_merge": true, "allow_rebase_merge": false, "allow_update_branch": true, "allow_auto_merge": false, "delete_branch_on_merge": true}
	client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "PATCH" || req.URL.Path != "/repos/owner/current" {
			t.Fatalf("request=%s %s", req.Method, req.URL)
		}
		var got map[string]any
		if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("payload=%v want %v", got, want)
		}
		return &http.Response{Request: req, StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"default_branch":"trunk","has_discussions":true}`))}, nil
	})
	desired, err := GeneralDeclaration(uimodel.General{
		TemplateRepository: pointer(true), DefaultBranch: pointer("trunk"),
		Features:     uimodel.Features{Issues: pointer(false), Projects: pointer(true), Wikis: pointer(false), Discussions: pointer(true)},
		PullRequests: uimodel.PullRequests{AllowMergeCommits: pointer(false), AllowSquashMerging: pointer(true), AllowRebaseMerging: pointer(false), AlwaysSuggestUpdatingPullRequestBranches: pointer(true), AllowAutoMerge: pointer(false), AutomaticallyDeleteHeadBranches: pointer(true)},
		Commits:      uimodel.Commits{RequireContributorsToSignOffOnWebBasedCommits: pointer(true)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, desired); err != nil {
		t.Fatal(err)
	}
}

func TestUIGeneralRawMergeObservation(t *testing.T) {
	access := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		return labelsResponse(req, 200, `{"data":{"repository":{"mergeCommitAllowed":false,"squashMergeAllowed":true,"rebaseMergeAllowed":false,"deleteBranchOnMerge":true,"autoMergeAllowed":false,"allowUpdateBranch":true,"squashMergeCommitTitle":"PR_TITLE","squashMergeCommitMessage":"PR_BODY","mergeCommitTitle":"MERGE_MESSAGE","mergeCommitMessage":"PR_TITLE"}}}`), nil
	}))
	state, err := (&MergeMethodsClient{graphql: access.graphql}).ReadMergeMethods(repository.Repository{Owner: "owner", Name: "current"})
	if err != nil {
		t.Fatal(err)
	}
	declaration := settings.RepositoryDeclaration{}.WithMergeMethods(state)
	got, err := GeneralObservation(declaration)
	want := uimodel.General{PullRequests: uimodel.PullRequests{AllowMergeCommits: pointer(false), AllowSquashMerging: pointer(true), AllowRebaseMerging: pointer(false), AutomaticallyDeleteHeadBranches: pointer(true), AllowAutoMerge: pointer(false), AlwaysSuggestUpdatingPullRequestBranches: pointer(true), SquashMerge: uimodel.CommitMessage{DefaultCommitMessage: pointer(uimodel.CommitMessageChoice("pull_request_title_and_description"))}}}
	if err != nil || !reflect.DeepEqual(got.General, want) || got.NativeConditionalValues.MergeCommitTitle == nil || *got.NativeConditionalValues.MergeCommitTitle != "MERGE_MESSAGE" || got.NativeConditionalValues.MergeCommitMessage == nil || *got.NativeConditionalValues.MergeCommitMessage != "PR_TITLE" {
		t.Fatalf("observation=%+v err=%v want %+v", got, err, want)
	}
}
