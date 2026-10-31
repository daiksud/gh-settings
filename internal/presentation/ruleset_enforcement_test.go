package presentation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

const enforcementDisplayPolicy = `{"id":7,"name":"policy","target":"branch","enforcement":"disabled","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[],"rules":[]}`

func TestNormalizedRulesetEnforcementPresentationRetainsWholeObjectContext(t *testing.T) {
	for _, desired := range []string{"disabled", "active"} {
		var before, after settings.RulesetDeclaration
		if err := json.Unmarshal([]byte(enforcementDisplayPolicy), &before); err != nil {
			t.Fatal(err)
		}
		afterJSON := strings.Replace(enforcementDisplayPolicy, `"enforcement":"disabled"`, `"enforcement":"`+desired+`"`, 1)
		if err := json.Unmarshal([]byte(afterJSON), &after); err != nil {
			t.Fatal(err)
		}
		record := settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
		change, err := settings.NewRulesetDisplayChangeWithNormalizations("update", &before, &after, settings.RulesetNormalizations{Enforcement: record})
		if err != nil {
			t.Fatal(err)
		}
		warning := `Warning: repository.settings.rulesets.enforcement_status: observed "evaluate" is outside UI choices; using "disabled" (projection only). Ruleset 7 "policy".`
		decision := `repository.settings.rulesets.enforcement_status (Ruleset 7 "policy"): "disabled" (normalized observation) → "` + desired + `" (write required)`
		whole := `repository.settings.rulesets: update "policy": ` + rulesetPolicy(&before) + ` → ` + rulesetPolicy(&after)
		want := warning + "\n" + decision + "\nEffective policy: " + whole
		if got := RulesetLine(change); got != want {
			t.Fatal(got, want)
		}
		if desired == "active" {
			canonical, err := settings.NewRulesetDisplayChange("update", &before, &after)
			if err != nil || RulesetLine(canonical) != whole {
				t.Fatal("canonical output changed", RulesetLine(canonical), err)
			}
		}
	}
}

func TestRulesetEnforcementWarningEscapesRawAndName(t *testing.T) {
	record := settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "FUTURE\"\n<token>", Fallback: "disabled"}
	want := `Warning: repository.settings.rulesets.enforcement_status: observed "FUTURE\"\n<token>" is outside UI choices; using "disabled" (projection only). Ruleset 7 "policy \"quoted\" <name>".`
	var before, after settings.RulesetDeclaration
	if err := json.Unmarshal([]byte(enforcementDisplayPolicy), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(enforcementDisplayPolicy), &after); err != nil {
		t.Fatal(err)
	}
	before.Name, after.Name = `policy "quoted" <name>`, `policy "quoted" <name>`
	change, err := settings.NewRulesetDisplayChangeWithNormalizations("update", &before, &after, settings.RulesetNormalizations{Enforcement: record})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(RulesetLine(change), "\n")
	if got := lines[0]; len(lines) != 3 || got != want {
		t.Fatal(got, want)
	}
}
