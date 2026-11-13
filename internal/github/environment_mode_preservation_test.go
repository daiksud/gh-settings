package github

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentModeGuardRechecksLatestPatterns(t *testing.T) {
	hasPattern, writes := false, 0
	client := &EnvironmentsClient{rest: actionsClient(t, func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" {
			writes++
			return actionsResponse(r, 400, `{}`), nil
		}
		base := "/repos/owner/current/environments/production"
		switch r.URL.Path {
		case "/repos/owner/current":
			return actionsResponse(r, 200, `{"id":71,"full_name":"owner/current","archived":false}`), nil
		case "/repos/owner/current/environments":
			return actionsResponse(r, 200, `{"total_count":1,"environments":[{"id":1,"name":"production"}]}`), nil
		case base:
			return actionsResponse(r, 200, `{"id":1,"name":"production","protection_rules":[],"can_admins_bypass":false,"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}`), nil
		case base + "/deployment_protection_rules":
			return actionsResponse(r, 200, customEnvironmentEmpty), nil
		case base + "/deployment-branch-policies":
			if hasPattern {
				return actionsResponse(r, 200, `{"total_count":1,"branch_policies":[{"id":7,"name":"late","type":"tag"}]}`), nil
			}
			return actionsResponse(r, 200, `{"total_count":0,"branch_policies":[]}`), nil
		default:
			t.Fatal(r.URL)
			return nil, nil
		}
	}).rest}
	repo := repository.Repository{Owner: "owner", Name: "current"}
	observed, err := client.Read(repo, []string{"production"})
	if err != nil {
		t.Fatal(err)
	}
	request := settings.EnvironmentDeclaration{Name: "production", DeploymentBranchPolicyDeclared: true}
	desired, _ := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{request})
	if changes, err := settings.ReconcileEnvironments(observed, &desired); err != nil || len(changes) != 1 {
		t.Fatal(changes, err)
	}
	hasPattern = true
	changed, err := client.Update(repo, request)
	var pre interface{ ReadFailedBeforeWrite() }
	if changed || writes != 0 || !errors.As(err, &pre) || !strings.Contains(err.Error(), "omitted deployment patterns") {
		t.Fatal("earlier empty plan bypassed fresh guard", changed, writes, err)
	}
}
