package presentation

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const deploymentDisplayPolicy = `{"name":"production","wait_timer":0,"prevent_self_review":false,"can_admins_bypass":false,"reviewers":[],"deployment_branch_policy":null}`

func TestEnvironmentDeploymentPresentationMarksEffectiveBeforeAndScopesModeEvidence(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, choice := range []uimodel.EnvironmentDeploymentChoice{uimodel.NoRestriction, uimodel.ProtectedBranchesOnly, uimodel.SelectedBranchesAndTags} {
			for _, partial := range []bool{false, true} {
				var before, after settings.EnvironmentDeclaration
				if err := json.Unmarshal([]byte(deploymentDisplayPolicy), &before); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(deploymentDisplayPolicy), &after); err != nil {
					t.Fatal(err)
				}
				if partial {
					after = settings.EnvironmentDeclaration{Name: "production", DeploymentBranchPolicyDeclared: true}
				}
				policy, _ := settings.EnvironmentDeploymentPolicyForChoice(choice)
				after.DeploymentBranchPolicy = policy
				record := settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: settings.EnvironmentBranchPolicy{ProtectedBranches: raw, CustomBranchPolicies: raw}, Fallback: uimodel.NoRestriction}
				change, err := settings.NewEnvironmentChangeWithDeploymentNormalization("update", &before, &after, record)
				if err != nil {
					t.Fatal(err)
				}
				beforeJSON, _ := json.Marshal(before)
				afterJSON, _ := json.Marshal(after)
				whole := `settings.environments: update "production": effective before policy ` + string(beforeJSON) + ` → request ` + string(afterJSON)
				if partial {
					whole += " (omitted fields are API-controlled; resulting values are unknown until apply)"
				}
				warning := `Warning: repository.settings.environments.deployment_branches_and_tags: observed {"protected_branches":` + strconv.FormatBool(raw) + `,"custom_branch_policies":` + strconv.FormatBool(raw) + `} is outside UI choices; using "no_restriction" (projection only). Environment "production".`
				decision := `repository.settings.environments.deployment_branches_and_tags (Environment "production"): "no_restriction" (normalized observation) → "` + string(choice) + `" (write required)`
				want := warning + "\n" + decision + "\n" + whole
				if got := EnvironmentLine(change); got != want || strings.Count(got, "effective before policy") != 1 {
					t.Fatal(got, want)
				}
				if partial || choice != uimodel.NoRestriction {
					ordinary, err := settings.NewEnvironmentChange("update", &before, &after)
					if err != nil || EnvironmentLine(ordinary) != whole {
						t.Fatal("ordinary effective policy label or request changed", EnvironmentLine(ordinary), err)
					}
				}
			}
		}
	}
	var before settings.EnvironmentDeclaration
	if err := json.Unmarshal([]byte(deploymentDisplayPolicy), &before); err != nil {
		t.Fatal(err)
	}
	timer := 5
	after := settings.EnvironmentDeclaration{Name: "production", WaitTimer: &timer}
	ordinary, err := settings.NewEnvironmentChange("update", &before, &after)
	if err != nil {
		t.Fatal(err)
	}
	got := EnvironmentLine(ordinary)
	if !strings.Contains(got, "effective before policy") || !strings.Contains(got, "omitted fields are API-controlled") || strings.Contains(got, "Warning:") || strings.Contains(got, "normalized observation") || strings.Contains(got, "protected_branches") || strings.Contains(got, "custom_branch_policies") {
		t.Fatal("omitted mode disclosed or implied raw proof", got)
	}
}
