package github

import (
	"encoding/json"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"net/http"
	"reflect"
	"testing"
)

func TestUIPagesExactExistingSiteUpdates(t *testing.T) {
	for _, test := range []struct {
		ui                               uimodel.Pages
		before, after, payload, beforeUI string
		public                           bool
	}{
		{uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: pagesUIPtr(uimodel.DeployFromABranch), Branch: pagesUIPtr("pages/source"), SelectFolder: pagesUIPtr(uimodel.PagesDocs)}, CustomDomain: &uimodel.PagesCustomDomain{}, EnforceHTTPS: pagesUIPtr(false)}, `{"build_type":"workflow","cname":"old.example.com","https_enforced":true,"public":false}`, `{"build_type":"legacy","source":{"branch":"pages/source","path":"/docs"},"cname":null,"https_enforced":false,"public":false}`, `{"build_type":"legacy","source":{"branch":"pages/source","path":"/docs"},"cname":null,"https_enforced":false}`, "github_actions", false},
		{uimodel.Pages{BuildAndDeployment: &uimodel.PagesBuildAndDeployment{Source: pagesUIPtr(uimodel.GitHubActions)}}, `{"build_type":"legacy","source":{"branch":"main","path":"/"},"cname":null,"https_enforced":true,"public":true}`, `{"build_type":"workflow","source":{"branch":"inactive","path":"nonsense"},"cname":null,"https_enforced":true,"public":true}`, `{"build_type":"workflow"}`, "deploy_from_a_branch", true},
		{uimodel.Pages{CustomDomain: &uimodel.PagesCustomDomain{Value: pagesUIPtr("Docs.Example.com")}}, `{"build_type":"legacy","source":{"branch":"main","path":"/"},"cname":null,"https_enforced":false,"public":true}`, `{"build_type":"legacy","source":{"branch":"main","path":"/"},"cname":"docs.example.com","https_enforced":false,"public":true}`, `{"cname":"docs.example.com"}`, "deploy_from_a_branch", true},
	} {
		calls := 0
		written := false
		client := &PagesClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if req.URL.Path == "/repos/owner/current" && req.Method == "GET" {
				return labelsResponse(req, 200, `{"id":7,"full_name":"owner/current","archived":false}`), nil
			}
			if req.URL.Path != "/repos/owner/current/pages" {
				t.Fatal(req.URL)
			}
			if req.Method == "PUT" {
				var got, want any
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(test.payload), &want); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatal(got, want)
				}
				written = true
				return labelsResponse(req, 204, ""), nil
			}
			if req.Method != "GET" {
				t.Fatal("lifecycle write", req.Method)
			}
			body := test.before
			if written {
				body = test.after
			}
			return labelsResponse(req, 200, body), nil
		}).rest}
		repo := repository.Repository{Owner: "owner", Name: "current"}
		raw, err := client.Read(repo)
		if err != nil {
			t.Fatal(err)
		}
		view, err := PagesObservation(raw)
		if err != nil || string(*view.Pages.BuildAndDeployment.Source) != test.beforeUI || view.Public != test.public {
			t.Fatal(view, err)
		}
		d, err := PagesDeclaration(test.ui)
		if err != nil {
			t.Fatal(err)
		}
		changed, err := client.Update(repo, d)
		if err != nil || !changed || calls != 7 {
			t.Fatal(changed, calls, err)
		}
	}
}
func TestUIPagesUnavailableIsNotAbsent(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
	}{
		{404, `{"message":"Not Found"}`}, {403, `{"message":"Forbidden"}`},
		{200, `{"build_type":"future","cname":null,"https_enforced":false,"public":true}`},
		{200, `{"build_type":"legacy","source":{"branch":"main","path":"future"},"cname":null,"https_enforced":false,"public":true}`},
		{200, `{"build_type":"workflow","https_enforced":false,"public":true}`},
	} {
		calls := 0
		client := &PagesClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != "GET" {
				t.Fatal("failed acquisition wrote")
			}
			return labelsResponse(req, test.status, test.body), nil
		}).rest}
		raw, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
		if err == nil || calls != 1 {
			t.Fatal(calls, err)
		}
		if _, err := PagesObservation(raw); err == nil {
			t.Fatal("failed read became absent/defaulted site")
		}
	}
}
