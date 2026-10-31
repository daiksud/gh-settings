package presentation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

const qualityDisplayPolicy = `{"id":7,"name":"policy","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"code_quality","parameters":{"severity":"errors"}}]}`

func TestNormalizedRulesetQualityPresentationPreservesRemovalAndUIChoices(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, test := range []struct{ native, choice string }{{"errors", "errors"}, {"warnings", "warnings_and_higher"}, {"notes", "notes_and_higher"}, {"all", "all"}, {"removed", ""}} {
			t.Run(test.native, func(t *testing.T) {
				beforeJSON := qualityDisplayPolicy
				records := settings.RulesetNormalizations{QualitySeverity: settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "FUTURE_SEVERITY", Fallback: "errors"}}
				prefix := ""
				if both {
					beforeJSON = strings.Replace(beforeJSON, `"enforcement":"active"`, `"enforcement":"disabled"`, 1)
					records.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
					prefix = `Warning: repository.settings.rulesets.enforcement_status: observed "evaluate" is outside UI choices; using "disabled" (projection only). Ruleset 7 "policy".` + "\n" + `repository.settings.rulesets.enforcement_status (Ruleset 7 "policy"): "disabled" (normalized observation) → "disabled" (write required)` + "\n"
				}
				afterJSON := strings.Replace(beforeJSON, `"severity":"errors"`, `"severity":"`+test.native+`"`, 1)
				if test.native == "removed" {
					afterJSON = strings.Replace(beforeJSON, `"rules":[{"type":"code_quality","parameters":{"severity":"errors"}}]`, `"rules":[]`, 1)
				}
				var before, after settings.RulesetDeclaration
				if err := json.Unmarshal([]byte(beforeJSON), &before); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(afterJSON), &after); err != nil {
					t.Fatal(err)
				}
				change, err := settings.NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records)
				if err != nil {
					t.Fatal(err)
				}
				warning := `Warning: repository.settings.rulesets.rules.require_code_quality_results.severity: observed "FUTURE_SEVERITY" is outside UI choices; using "errors" (projection only). Ruleset 7 "policy".`
				decision := settings.RulesetQualitySeverityPath + ` (Ruleset 7 "policy"): "errors" (normalized observation) → `
				if test.native == "removed" {
					decision += `inactive (rule removed; write required)`
				} else {
					decision += `"` + test.choice + `" (write required)`
				}
				whole := `repository.settings.rulesets: update "policy": ` + rulesetPolicy(&before) + ` → ` + rulesetPolicy(&after)
				want := prefix + warning + "\n" + decision + "\nEffective policy: " + whole
				if got := RulesetLine(change); got != want || strings.Count(got, "Effective policy:") != 1 {
					t.Fatal(got, want)
				}
				if test.native != "errors" {
					canonical, err := settings.NewRulesetDisplayChange("update", &before, &after)
					if err != nil || RulesetLine(canonical) != whole {
						t.Fatal("ordinary output changed", RulesetLine(canonical), err)
					}
				}
			})
		}
	}
}
