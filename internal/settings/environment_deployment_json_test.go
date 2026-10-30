package settings

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestEnvironmentDeploymentMetadataRequiresCompleteTypedBooleanPair(t *testing.T) {
	for _, raw := range []bool{false, true} {
		text := `{"path":"repository.settings.environments.deployment_branches_and_tags","raw_policy":{"protected_branches":` + strconv.FormatBool(raw) + `,"custom_branch_policies":` + strconv.FormatBool(raw) + `},"fallback":"no_restriction"}`
		want := EnvironmentDeploymentNormalization{Path: EnvironmentDeploymentModePath, RawPolicy: EnvironmentBranchPolicy{ProtectedBranches: raw, CustomBranchPolicies: raw}, Fallback: uimodel.NoRestriction}
		var got EnvironmentDeploymentNormalization
		if err := json.Unmarshal([]byte(text), &got); err != nil || got != want {
			t.Fatal("explicit false/true lost", got, err)
		}
		encoded, err := json.Marshal(got)
		if err != nil || string(encoded) != text {
			t.Fatal("boolean pair changed shape", string(encoded), err)
		}
	}
	for _, text := range []string{
		`null`, `{}`, `{"raw_policy":{"protected_branches":false,"custom_branch_policies":false},"fallback":"no_restriction"}`,
		`{"path":null,"raw_policy":{"protected_branches":false,"custom_branch_policies":false},"fallback":"no_restriction"}`,
		`{"path":"x","fallback":"no_restriction"}`, `{"path":"x","raw_policy":null,"fallback":"no_restriction"}`, `{"path":"x","raw_policy":{},"fallback":"no_restriction"}`,
		`{"path":"x","raw_policy":{"protected_branches":false},"fallback":"no_restriction"}`, `{"path":"x","raw_policy":{"custom_branch_policies":false},"fallback":"no_restriction"}`,
		`{"path":"x","raw_policy":{"protected_branches":null,"custom_branch_policies":false},"fallback":"no_restriction"}`, `{"path":"x","raw_policy":{"protected_branches":false,"custom_branch_policies":null},"fallback":"no_restriction"}`,
		`{"path":"x","raw_policy":{"protected_branches":"false","custom_branch_policies":false},"fallback":"no_restriction"}`, `{"path":"x","raw_policy":{"protected_branches":false,"custom_branch_policies":0},"fallback":"no_restriction"}`,
		`{"path":"x","raw_policy":{"protected_branches":false,"custom_branch_policies":false}}`, `{"path":"x","raw_policy":{"protected_branches":false,"custom_branch_policies":false},"fallback":null}`, `{"path":"x","raw_policy":{"protected_branches":false,"custom_branch_policies":false},"fallback":false}`,
		`{"path":"x","raw_policy":{"protected_branches":false,"custom_branch_policies":false,"extra":true},"fallback":"no_restriction"}`, `{"path":"x","raw_policy":{"protected_branches":false,"custom_branch_policies":false},"fallback":"no_restriction","extra":true}`,
	} {
		before := EnvironmentDeploymentNormalization{Path: "sentinel", RawPolicy: EnvironmentBranchPolicy{ProtectedBranches: true}, Fallback: "sentinel"}
		got := before
		if err := json.Unmarshal([]byte(text), &got); err == nil || got != before {
			t.Fatal("incomplete metadata accepted or partially assigned", text, got, err)
		}
	}
}
