package presentation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentRefTypePresentationShowsRawWarningsAndOneOwnedCollectionDecision(t *testing.T) {
	for _, collision := range []bool{false, true} {
		for _, clear := range []bool{false, true} {
			var before settings.EnvironmentDeclaration
			if err := json.Unmarshal([]byte(deploymentDisplayPolicy), &before); err != nil {
				t.Fatal(err)
			}
			before.Name = "prod\"<>&"
			before.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
			patterns := []settings.EnvironmentPatternState{
				{ID: 32, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "release\"/*"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "future\n\"", Fallback: "branch"}},
				{ID: 17, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "", Fallback: "branch"}},
				{ID: 33, Pattern: settings.EnvironmentPattern{Type: "tag", Name: "CANONICAL_COMPANION"}},
			}
			if collision {
				patterns = append(patterns, settings.EnvironmentPatternState{ID: 34, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}})
			} else {
				effective, err := settings.EffectiveEnvironmentPatterns(patterns)
				if err != nil {
					t.Fatal(err)
				}
				before.DeploymentBranchPatterns = &effective
			}
			wanted := []settings.EnvironmentPattern{{Type: "tag", Name: "main"}}
			requestJSON := `[{"ref_type":"tag","name_pattern":"main"}]`
			if clear {
				wanted = []settings.EnvironmentPattern{}
				requestJSON = `[]`
			}
			after := settings.EnvironmentDeclaration{Name: before.Name, DeploymentBranchPatterns: &wanted}
			change, err := settings.NewEnvironmentChangeWithPatternNormalizations("update", &before, &after, patterns)
			if err != nil {
				t.Fatal(err)
			}
			got := EnvironmentLine(change)
			warning17 := `Warning: repository.settings.environments.deployment_branch_or_tag_rules.ref_type: observed "" is outside UI choices; using "branch" (projection only). Environment "prod\"<>&", pattern 17 name_pattern "main".`
			warning32 := `Warning: repository.settings.environments.deployment_branch_or_tag_rules.ref_type: observed "future\n\"" is outside UI choices; using "branch" (projection only). Environment "prod\"<>&", pattern 32 name_pattern "release\"/*".`
			decision := `repository.settings.environments.deployment_branch_or_tag_rules.ref_type (Environment "prod\"<>&"): normalized pattern observation → request ` + requestJSON + ` (write required; raw identities differ)`
			if !strings.HasPrefix(got, warning17+"\n"+warning32+"\n"+decision+"\n") || strings.Count(got, "Warning:") != 2 || strings.Count(got, "write required") != 1 || !strings.Contains(got, "effective before policy") || !strings.Contains(got, "omitted fields are API-controlled") || strings.Contains(got, "pattern 33") || strings.Contains(got, "pattern 34") {
				t.Fatal("warning ordering/scope/escaping or collection decision", got)
			}
			if strings.Contains(got, "effective deployment pattern list unavailable (normalized ref types collide; acquired raw identities retained)") != collision {
				t.Fatal("collision displayed as absence", got)
			}
		}
	}
}
