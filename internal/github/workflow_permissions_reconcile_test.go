package github

import (
	"net/http"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestMatchingWorkflowPermissionsDoNotWrite(t *testing.T) {
	reads := 0
	client := actionsClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" {
			t.Fatalf("matching intent sent %s", req.Method)
		}
		reads++
		return actionsResponse(req, 200, `{"default_workflow_permissions":"read","can_approve_pull_request_reviews":false}`), nil
	})
	if wrote, err := client.ApplyWorkflowPermissions(repository.Repository{Owner: "owner", Name: "current"}, settings.ActionsPolicyDeclaration{
		DefaultWorkflowPermissions: actionsPointer("read"), CanApprovePullRequestReviews: actionsPointer(false),
	}); err != nil || wrote {
		t.Fatal(wrote, err)
	}
	if reads != 1 {
		t.Fatal(reads)
	}
}
