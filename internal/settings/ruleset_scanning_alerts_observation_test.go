package settings

import (
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"testing"
)

func normalizedScanningAlertsState(companions bool) RulesetState {
	state := normalizedTeamApprovalsState(companions)
	if !companions {
		state.TeamApprovalsNormalizations = nil
	}
	state.Policy.Name = "scanning"
	*state.Policy.Rules = append(*state.Policy.Rules, RulesetRule{Type: "code_scanning", Parameters: &RuleParameters{CodeScanningTools: &[]CodeScanningTool{
		{Tool: "z tool", AlertsThreshold: "errors", SecurityAlertsThreshold: "all"},
		{Tool: "CodeQL", AlertsThreshold: "errors", SecurityAlertsThreshold: "high_or_higher"},
		{Tool: "canonical", AlertsThreshold: "none", SecurityAlertsThreshold: "none"},
	}}})
	state.ScanningAlertsNormalizations = []RulesetScanningAlertsNormalization{
		{Tool: "z tool", Path: RulesetScanningAlertsPath, RawValue: "future", Fallback: "errors"},
		{Tool: "CodeQL", Path: RulesetScanningAlertsPath, RawValue: "", Fallback: "errors"},
	}
	return state
}
func scanningAlertsRecords(state RulesetState) RulesetNormalizations {
	records := teamApprovalsRecords(state)
	records.ScanningAlerts = state.ScanningAlertsNormalizations
	return records
}

func TestScanningAlertsMappingAndOwnedState(t *testing.T) {
	if RulesetScanningAlertsPath != "repository.settings.rulesets.rules.require_code_scanning_results.code_scanning_tools.alerts" || DefaultRulesetScanningAlertsChoice != "errors" {
		t.Fatal("path/default")
	}
	for _, value := range []string{"none", "errors", "errors_and_warnings", "all", "", " ", "ERRORS", "unknown"} {
		want := value == "none" || value == "errors" || value == "errors_and_warnings" || value == "all"
		choice, ok := RulesetScanningAlertsChoiceForValue(value)
		if ok != want {
			t.Fatal(value)
		}
		got, valid := RulesetScanningAlertsValueForChoice(uimodel.CodeScanningAlerts(value))
		if valid != want || (valid && (got != value || string(choice) != value)) {
			t.Fatal(value, got)
		}
	}
	for _, companions := range []bool{false, true} {
		state := normalizedScanningAlertsState(companions)
		observed, err := NewRulesetObservation([]RulesetState{state}, 0)
		if err != nil {
			t.Fatal(err)
		}
		state.ScanningAlertsNormalizations[0].RawValue = "CORRUPT"
		(*rulesetCodeScanningTools(state.Policy))[0].Tool = "CORRUPT"
		values, err := observed.Values()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := values[0].RawCodeScanningTools()
		want := []CodeScanningTool{{Tool: "CodeQL", AlertsThreshold: "", SecurityAlertsThreshold: "high_or_higher"}, {Tool: "canonical", AlertsThreshold: "none", SecurityAlertsThreshold: "none"}, {Tool: "z tool", AlertsThreshold: "future", SecurityAlertsThreshold: "all"}}
		if err != nil || !reflect.DeepEqual(raw, want) {
			t.Fatal(raw, err)
		}
		raw[0].Tool = "changed"
		values[0].ScanningAlertsNormalizations[0].RawValue = "changed"
		again, _ := observed.Values()
		raw, _ = again[0].RawCodeScanningTools()
		if !reflect.DeepEqual(raw, want) {
			t.Fatal("aliased tools")
		}
		ordered := OrderedRulesetScanningAlertsNormalizations(again[0].ScanningAlertsNormalizations)
		if ordered[0].Tool != "CodeQL" || again[0].ScanningAlertsNormalizations[0].Tool != "z tool" {
			t.Fatal("ordering mutated records")
		}
	}
	state := normalizedScanningAlertsState(false)
	state.ScanningAlertsNormalizations = nil
	*rulesetCodeScanningTools(state.Policy) = []CodeScanningTool{}
	raw, err := state.RawCodeScanningTools()
	if err != nil || raw == nil || len(raw) != 0 {
		t.Fatal("empty", raw, err)
	}
	rules := []RulesetRule{}
	state.Policy.Rules = &rules
	raw, err = state.RawCodeScanningTools()
	if err != nil || raw != nil {
		t.Fatal("absent", raw, err)
	}
}

func TestScanningAlertsRejectInvalidIdentityAndEvidenceIndependently(t *testing.T) {
	for _, mutate := range []func(*RulesetState){
		func(s *RulesetState) { s.ScanningAlertsNormalizations[0] = RulesetScanningAlertsNormalization{} },
		func(s *RulesetState) { s.ScanningAlertsNormalizations[0].Tool = "Z TOOL" },
		func(s *RulesetState) { s.ScanningAlertsNormalizations[0].Path = "forged" },
		func(s *RulesetState) { s.ScanningAlertsNormalizations[0].Fallback = "none" },
		func(s *RulesetState) { s.ScanningAlertsNormalizations[0].RawValue = "all" },
		func(s *RulesetState) {
			s.ScanningAlertsNormalizations = append(s.ScanningAlertsNormalizations, s.ScanningAlertsNormalizations[0])
		},
		func(s *RulesetState) { (*rulesetCodeScanningTools(s.Policy))[0].AlertsThreshold = "all" },
		func(s *RulesetState) { (*rulesetCodeScanningTools(s.Policy))[0].SecurityAlertsThreshold = "future" },
		func(s *RulesetState) { (*rulesetCodeScanningTools(s.Policy))[0].Tool = "\n" },
		func(s *RulesetState) { (*rulesetCodeScanningTools(s.Policy))[0].Tool = "CodeQL" },
		func(s *RulesetState) { s.Policy.ID = rsID(99) },
	} {
		state := normalizedScanningAlertsState(true)
		mutate(&state)
		if got, err := NewRulesetObservation([]RulesetState{state}, 0); err == nil || !reflect.DeepEqual(got, RulesetObservation{}) {
			t.Fatal(got, err)
		}
		if raw, err := state.RawCodeScanningTools(); err == nil || raw != nil {
			t.Fatal(raw, err)
		}
	}
}

func TestScanningAlertsReconcileOwnsSelectedWholePolicyAndRemoval(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, mode := range []string{"equal", "different", "tool removed", "replacement", "rule removed", "empty tools"} {
			state := normalizedScanningAlertsState(companions)
			after := cloneRuleset(state.Policy)
			tools := rulesetCodeScanningTools(after)
			switch mode {
			case "different":
				(*tools)[2].AlertsThreshold = "all"
			case "tool removed":
				*tools = (*tools)[:2]
			case "replacement":
				(*tools)[2].Tool = "other tool"
				(*tools)[2].AlertsThreshold = "none"
			case "empty tools":
				*tools = []CodeScanningTool{}
			case "rule removed":
				var kept []RulesetRule
				for _, r := range *after.Rules {
					if r.Type != "code_scanning" {
						kept = append(kept, r)
					}
				}
				after.Rules = &kept
			}
			private := normalizedScanningAlertsState(false)
			private.ID = 9
			private.Policy.ID = rsID(9)
			private.Policy.Name = "PRIVATE_SCANNING"
			observed := rsObserved(t, private, state)
			changes, err := ReconcileRulesets(observed, rsDesired(t, after))
			if err != nil || len(changes) != 1 || changes[0].ID() != 7 || !reflect.DeepEqual(changes[0].Normalizations(), scanningAlertsRecords(state)) {
				t.Fatal(mode, changes, err)
			}
			records := changes[0].Normalizations()
			records.ScanningAlerts[0].RawValue = "changed"
			if !reflect.DeepEqual(changes[0].Display().Normalizations(), scanningAlertsRecords(state)) {
				t.Fatal("evidence alias")
			}
			if _, err := NewRulesetDisplayChangeWithNormalizations("update", changes[0].Before(), changes[0].After(), changes[0].Normalizations()); err != nil {
				t.Fatal(err)
			}
			for _, desired := range []*RulesetDeclarations{nil, rsDesired(t)} {
				if got, err := ReconcileRulesets(observed, desired); err != nil || len(got) != 0 {
					t.Fatal("unmanaged", got, err)
				}
			}
		}
	}
	canonical := normalizedScanningAlertsState(false)
	canonical.ScanningAlertsNormalizations = nil
	if changes, err := ReconcileRulesets(rsObserved(t, canonical), rsDesired(t, canonical.Policy)); err != nil || len(changes) != 0 {
		t.Fatal("canonical", changes, err)
	}
	desired := cloneRuleset(canonical.Policy)
	(*rulesetCodeScanningTools(desired))[0].AlertsThreshold = ""
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{desired}); err == nil {
		t.Fatal("invented desired default")
	}
}
