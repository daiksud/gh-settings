package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUICreationPolicies(t *testing.T) {
	for _, test := range []struct{ ui, api string }{{"all_users", "ALL"}, {"collaborators_only", "COLLABORATORS_ONLY"}} {
		for _, issue := range []bool{false, true} {
			choice := uimodel.CreationAllowedBy(test.ui)
			ui := uimodel.General{}
			var want settings.RepositoryDeclaration
			if issue {
				ui.Features.IssuePermissions.CreationAllowedBy = &choice
				want.IssueCreationPolicy = &test.api
			} else {
				ui.Features.PullRequestPermissions.CreationAllowedBy = &choice
				want.PullRequestCreationPolicy = &test.api
			}
			got, err := GeneralDeclaration(ui)
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("declaration=%+v err=%v want %+v", got, err, want)
			}
			observed, err := GeneralObservation(want)
			if err != nil || !reflect.DeepEqual(observed.General, ui) {
				t.Fatalf("observation=%+v err=%v want %+v", observed, err, ui)
			}
		}
	}
	if _, err := GeneralDeclaration(uimodel.General{Features: uimodel.Features{IssuePermissions: uimodel.CreationPermissions{CreationAllowedBy: pointer(uimodel.CreationAllowedBy("ALL"))}}}); err == nil {
		t.Fatal("API token accepted as UI choice")
	}
	for _, source := range []settings.RepositoryDeclaration{{IssueCreationPolicy: pointer("UNKNOWN")}, {PullRequestCreationPolicy: pointer("UNKNOWN")}} {
		if _, err := GeneralObservation(source); err == nil {
			t.Fatal("unknown acquired policy accepted")
		}
	}
}

func TestUICreationPolicyGraphQLPayload(t *testing.T) {
	calls := 0
	client := prAccessTestClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		var body struct {
			Query     string
			Variables map[string]any
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if calls == 1 {
			if !reflect.DeepEqual(body.Variables, map[string]any{"owner": "owner", "name": "current"}) {
				t.Fatal(body.Variables)
			}
			return labelsResponse(req, 200, `{"data":{"repository":{"id":"repo-id","nameWithOwner":"owner/current"}}}`), nil
		}
		want := map[string]any{"input": map[string]any{"repositoryId": "repo-id", "hasPullRequestsEnabled": true, "hasSponsorshipsEnabled": false, "issueCreationPolicy": "COLLABORATORS_ONLY", "pullRequestCreationPolicy": "ALL"}}
		if !reflect.DeepEqual(body.Variables, want) {
			t.Fatalf("variables=%v want %v", body.Variables, want)
		}
		return labelsResponse(req, 200, `{"data":{"updateRepository":{"repository":{"id":"repo-id","nameWithOwner":"owner/current","hasPullRequestsEnabled":true,"hasSponsorshipsEnabled":false,"issueCreationPolicy":"COLLABORATORS_ONLY","pullRequestCreationPolicy":"ALL"}}}}`), nil
	}))
	desired, err := GeneralDeclaration(uimodel.General{Features: uimodel.Features{PullRequests: pointer(true), Sponsorships: pointer(false), IssuePermissions: uimodel.CreationPermissions{CreationAllowedBy: pointer(uimodel.CreationAllowedBy("collaborators_only"))}, PullRequestPermissions: uimodel.CreationPermissions{CreationAllowedBy: pointer(uimodel.CreationAllowedBy("all_users"))}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ApplyRepositoryAccess(repository.Repository{Owner: "owner", Name: "current"}, desired); err != nil || calls != 2 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
