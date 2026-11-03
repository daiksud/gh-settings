package github

import (
	"encoding/json"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestRulesetUpdateAcquisitionRequiresBoolean(t *testing.T) {
	for _, parameters := range []string{"", `,"parameters":null`, `,"parameters":{}`, `,"parameters":{"update_allows_fetch_and_merge":null}`, `,"parameters":{"update_allows_fetch_and_merge":"false"}`, `,"parameters":{"update_allows_fetch_and_merge":false,"future":true}`, `,"parameters":{"update_allows_fetch_and_merge":false,"severity":"errors"}`} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(basicRulesetJSON), &raw)
		raw["rules"] = json.RawMessage(`[{"type":"update"` + parameters + `}]`)
		if _, err := decodeRulesetPolicy(raw); err == nil {
			t.Errorf("incomplete/unknown update accepted: %s", parameters)
		}
	}
	for _, enabled := range []bool{false, true} {
		p := &settings.RuleParameters{UpdateAllowsFetchAndMerge: &enabled}
		for _, kind := range []string{"creation", "deletion", "pull_request", "required_status_checks", "code_scanning", "code_quality"} {
			if err := settings.ValidateRuleset(settings.RulesetDeclaration{Name: "policy", Rules: rulesetPtr([]settings.RulesetRule{{Type: kind, Parameters: p}})}, false); err == nil {
				t.Errorf("%s accepted update parameter", kind)
			}
		}
		source := apiRulesetBase()
		source.Rules = rulesetPtr([]settings.RulesetRule{{Type: "update", Parameters: p}})
		observed, err := RulesetObservation(source)
		if err != nil || observed.Rules.RestrictUpdates == nil || *observed.Rules.RestrictUpdates.AllowFetchAndMerge != enabled {
			t.Fatalf("observation=%+v err=%v", observed, err)
		}
		*observed.Rules.RestrictUpdates.AllowFetchAndMerge = !enabled
		if *p.UpdateAllowsFetchAndMerge != enabled {
			t.Fatal("observation aliases source")
		}
		ui := uiRulesetBase()
		ui.Rules.RestrictUpdates = &uimodel.UpdateRule{AllowFetchAndMerge: &enabled}
		desired, err := RulesetDeclaration(ui)
		if err != nil || len(*desired.Rules) != 1 || (*desired.Rules)[0].Type != "update" || *(*desired.Rules)[0].Parameters.UpdateAllowsFetchAndMerge != enabled {
			t.Fatalf("declaration=%+v err=%v", desired, err)
		}
	}
}
