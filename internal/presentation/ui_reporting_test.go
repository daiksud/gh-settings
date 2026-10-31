package presentation

import (
	"encoding/json"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
	"testing"
)

func TestPagesChoicesOnBothSides(t *testing.T) {
	for _, tc := range []struct{ field, before, after, want string }{
		{"pages.build_type", "legacy", "workflow", `repository.settings.pages.build_and_deployment.source: "deploy_from_a_branch" → "github_actions" (branch source becomes inactive)`},
		{"pages.source.path", "/", "/docs", `repository.settings.pages.build_and_deployment.select_folder: "root" → "docs"`},
	} {
		c, err := settings.NewPagesChange(tc.field, tc.before, tc.after, false)
		if err != nil {
			t.Fatal(err)
		}
		if got := PagesLine(c); got != tc.want {
			t.Fatalf("%s; want %s", got, tc.want)
		}
	}
}
func TestWorkflowChoicesDoNotLeakTransportNames(t *testing.T) {
	c, err := settings.NewActionsPolicyChange("actions.default_workflow_permissions", "read", "write")
	if err != nil {
		t.Fatal(err)
	}
	got := ActionsPolicyLine(c)
	if got != `repository.settings.actions.general.workflow_permissions: "read_repository_contents_and_packages_permissions" → "read_and_write_permissions"` {
		t.Fatal(got)
	}
	if c.Field() != "actions.default_workflow_permissions" || strings.Contains(got, `"write"`) {
		t.Fatal("transport identity or display changed incorrectly")
	}
}

func TestRulesetPresentationUsesUIFieldsOnBothSides(t *testing.T) {
	var before, after settings.RulesetDeclaration
	if err := json.Unmarshal([]byte(enforcementDisplayPolicy), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(enforcementDisplayPolicy), &after); err != nil {
		t.Fatal(err)
	}
	active := "active"
	after.Enforcement = &active
	change, err := settings.NewRulesetDisplayChange("update", &before, &after)
	if err != nil {
		t.Fatal(err)
	}
	got := RulesetLine(change)
	wantBefore := `{"bypass_list":[],"enforcement_status":"disabled","id":7,"rules":{"block_force_pushes":false,"require_linear_history":false,"require_signed_commits":false,"restrict_deletions":false},"ruleset_name":"policy","target_branches":{"exclude_by_pattern":[],"include_all_branches":false,"include_by_pattern":[],"include_default_branch":false}}`
	wantAfter := strings.Replace(wantBefore, `"enforcement_status":"disabled"`, `"enforcement_status":"active"`, 1)
	want := `repository.settings.rulesets: update "policy": ` + wantBefore + ` → ` + wantAfter
	if got != want {
		t.Fatalf("got %s; want %s", got, want)
	}
}

func TestEnvironmentPresentationKeepsOmittedRequestValuesUnknown(t *testing.T) {
	var before settings.EnvironmentDeclaration
	if err := json.Unmarshal([]byte(deploymentDisplayPolicy), &before); err != nil {
		t.Fatal(err)
	}
	timer := 5
	after := settings.EnvironmentDeclaration{Name: "production", WaitTimer: &timer}
	change, err := settings.NewEnvironmentChange("update", &before, &after)
	if err != nil {
		t.Fatal(err)
	}
	got := EnvironmentLine(change)
	want := `repository.settings.environments: update "production": effective before policy {"deployment_protection_rules":{"wait_timer":0},"name":"production"} → request {"deployment_protection_rules":{"wait_timer":5},"name":"production"} (omitted fields are API-controlled; resulting values are unknown until apply)`
	if got != want {
		t.Fatalf("got %s; want %s", got, want)
	}
}

func TestEnvironmentPresentationDoesNotDiscloseOmittedValues(t *testing.T) {
	var before settings.EnvironmentDeclaration
	if err := json.Unmarshal([]byte(deploymentDisplayPolicy), &before); err != nil {
		t.Fatal(err)
	}
	reviewers := []settings.EnvironmentReviewer{{Type: "Team", ID: 987654321}}
	patterns := []settings.EnvironmentPattern{{Type: "branch", Name: "OMITTED_PATTERN_CANARY"}}
	before.Reviewers = &reviewers
	before.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{ProtectedBranches: false, CustomBranchPolicies: true}
	before.DeploymentBranchPatterns = &patterns
	timer := 5
	after := settings.EnvironmentDeclaration{Name: "production", WaitTimer: &timer}
	change, err := settings.NewEnvironmentChange("update", &before, &after)
	if err != nil {
		t.Fatal(err)
	}
	got := EnvironmentLine(change)
	if strings.Contains(got, "987654321") || strings.Contains(got, "OMITTED_PATTERN_CANARY") || strings.Contains(got, "deployment_branches_and_tags") || !strings.Contains(got, `"wait_timer":0`) || !strings.Contains(got, `"wait_timer":5`) {
		t.Fatal(got)
	}
}
