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

func TestUIActionsRetentionValues(t *testing.T) {
	for _, days := range []int{1, 400} {
		got, err := ActionsDeclaration(uimodel.ActionsGeneral{CheckWorkflowRunStatusArtifactAndLogRetention: &days})
		if err != nil || !reflect.DeepEqual(got, settings.ActionsPolicyDeclaration{RetentionDays: &days}) {
			t.Fatal(got, err)
		}
	}
	for _, days := range []int{0, 401} {
		if _, err := ActionsDeclaration(uimodel.ActionsGeneral{CheckWorkflowRunStatusArtifactAndLogRetention: &days}); err == nil {
			t.Fatal("invalid retention accepted")
		}
	}
	if _, err := WithActionsRetentionObservation(uimodel.ActionsGeneral{}, settings.RetentionObservation{}); err == nil {
		t.Fatal("missing observation became zero")
	}
}

func TestUIActionsRetentionRawConfiguredAndCap(t *testing.T) {
	client := &RetentionClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
		return labelsResponse(req, 200, `{"days":90,"maximum_allowed_days":30}`), nil
	}).rest}
	o, err := client.Read(repository.Repository{Owner: "owner", Name: "current"})
	if err != nil {
		t.Fatal(err)
	}
	view, err := WithActionsRetentionObservation(uimodel.ActionsGeneral{AllowGitHubActionsToCreateAndApprovePullRequests: actionsUIPtr(false)}, o)
	if err != nil || view.MaximumAllowedDays != 30 || *view.General.CheckWorkflowRunStatusArtifactAndLogRetention != 90 || view.General.AllowGitHubActionsToCreateAndApprovePullRequests == nil {
		t.Fatal(view, err)
	}
	if err := settings.PrepareRetention(o, *view.General.CheckWorkflowRunStatusArtifactAndLogRetention); err == nil {
		t.Fatal("observed configuration above cap became valid desired value")
	}
}

func TestUIActionsRetentionExactWrite(t *testing.T) {
	calls := 0
	client := &RetentionClient{rest: actionsClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Path != "/repos/owner/current/actions/permissions/artifact-and-log-retention" {
			t.Fatal(req.URL)
		}
		if req.Method == "PUT" {
			var got map[string]any
			if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, map[string]any{"days": float64(1)}) {
				t.Fatal(got)
			}
			return labelsResponse(req, 204, ""), nil
		}
		if calls == 1 {
			return labelsResponse(req, 200, `{"days":90,"maximum_allowed_days":90}`), nil
		}
		return labelsResponse(req, 200, `{"days":1,"maximum_allowed_days":90}`), nil
	}).rest}
	d, err := ActionsDeclaration(uimodel.ActionsGeneral{CheckWorkflowRunStatusArtifactAndLogRetention: actionsUIPtr(1)})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := client.Update(repository.Repository{Owner: "owner", Name: "current"}, *d.RetentionDays)
	if err != nil || !changed || calls != 3 {
		t.Fatal(changed, calls, err)
	}
}
