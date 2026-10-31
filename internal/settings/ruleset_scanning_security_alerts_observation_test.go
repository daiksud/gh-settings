package settings

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func normalizedScanningSecurityAlertsState(companions bool) RulesetState {
	state := normalizedScanningAlertsState(companions)
	if !companions {
		state.ScanningAlertsNormalizations = nil
	}
	for i := range *rulesetCodeScanningTools(state.Policy) {
		tool := &(*rulesetCodeScanningTools(state.Policy))[i]
		if tool.Tool != "canonical" {
			tool.SecurityAlertsThreshold = "high_or_higher"
		}
	}
	state.ScanningSecurityAlertsNormalizations = []RulesetScanningSecurityAlertsNormalization{
		{Tool: "z tool", Path: RulesetScanningSecurityAlertsPath, RawValue: "future security", Fallback: "high_or_higher"},
		{Tool: "CodeQL", Path: RulesetScanningSecurityAlertsPath, RawValue: "", Fallback: "high_or_higher"},
	}
	return state
}
func scanningSecurityAlertsRecords(state RulesetState) RulesetNormalizations {
	records := scanningAlertsRecords(state)
	records.ScanningSecurityAlerts = state.ScanningSecurityAlertsNormalizations
	return records
}

func TestScanningSecurityAlertsMappingAndOwnedState(t *testing.T) {
	if RulesetScanningSecurityAlertsPath != "repository.settings.rulesets.rules.require_code_scanning_results.code_scanning_tools.security_alerts" || DefaultRulesetScanningSecurityAlertsChoice != "high_or_higher" {
		t.Fatal("path/default")
	}
	for _, value := range []string{"none", "critical", "high_or_higher", "medium_or_higher", "all", "", " ", "errors", "HIGH_OR_HIGHER", "unknown"} {
		want := value == "none" || value == "critical" || value == "high_or_higher" || value == "medium_or_higher" || value == "all"
		choice, ok := RulesetScanningSecurityAlertsChoiceForValue(value)
		if ok != want {
			t.Fatal(value)
		}
		got, valid := RulesetScanningSecurityAlertsValueForChoice(uimodel.CodeScanningSecurityAlerts(value))
		if valid != want || (valid && (got != value || string(choice) != value)) {
			t.Fatal(value, got)
		}
	}
	for _, companions := range []bool{false, true} {
		state := normalizedScanningSecurityAlertsState(companions)
		observed, err := NewRulesetObservation([]RulesetState{state}, 0)
		if err != nil {
			t.Fatal(err)
		}
		state.ScanningSecurityAlertsNormalizations[0].RawValue = "CORRUPT"
		(*rulesetCodeScanningTools(state.Policy))[0].Tool = "CORRUPT"
		values, err := observed.Values()
		if err != nil {
			t.Fatal(err)
		}
		raw, err := values[0].RawCodeScanningTools()
		firstAlerts, lastAlerts := "errors", "errors"
		if companions {
			firstAlerts, lastAlerts = "", "future"
		}
		want := []CodeScanningTool{{Tool: "CodeQL", AlertsThreshold: firstAlerts, SecurityAlertsThreshold: ""}, {Tool: "canonical", AlertsThreshold: "none", SecurityAlertsThreshold: "none"}, {Tool: "z tool", AlertsThreshold: lastAlerts, SecurityAlertsThreshold: "future security"}}
		if err != nil || !reflect.DeepEqual(raw, want) {
			t.Fatal(raw, err)
		}
		raw[0].Tool = "changed"
		values[0].ScanningSecurityAlertsNormalizations[0].RawValue = "changed"
		again, _ := observed.Values()
		raw, _ = again[0].RawCodeScanningTools()
		if !reflect.DeepEqual(raw, want) {
			t.Fatal("aliased tools")
		}
		ordered := OrderedRulesetScanningSecurityAlertsNormalizations(again[0].ScanningSecurityAlertsNormalizations)
		if ordered[0].Tool != "CodeQL" || again[0].ScanningSecurityAlertsNormalizations[0].Tool != "z tool" {
			t.Fatal("ordering mutated records")
		}
	}
	state := normalizedScanningSecurityAlertsState(false)
	state.ScanningSecurityAlertsNormalizations = nil
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

func TestScanningSecurityAlertsRejectInvalidIdentityAndEvidenceIndependently(t *testing.T) {
	for _, mutate := range []func(*RulesetState){
		func(s *RulesetState) {
			s.ScanningSecurityAlertsNormalizations[0] = RulesetScanningSecurityAlertsNormalization{}
		},
		func(s *RulesetState) { s.ScanningSecurityAlertsNormalizations[0].Tool = "Z TOOL" },
		func(s *RulesetState) { s.ScanningSecurityAlertsNormalizations[0].Path = "forged" },
		func(s *RulesetState) { s.ScanningSecurityAlertsNormalizations[0].Fallback = "none" },
		func(s *RulesetState) { s.ScanningSecurityAlertsNormalizations[0].RawValue = "all" },
		func(s *RulesetState) {
			s.ScanningSecurityAlertsNormalizations = append(s.ScanningSecurityAlertsNormalizations, s.ScanningSecurityAlertsNormalizations[0])
		},
		func(s *RulesetState) { (*rulesetCodeScanningTools(s.Policy))[0].SecurityAlertsThreshold = "all" },
		func(s *RulesetState) { (*rulesetCodeScanningTools(s.Policy))[0].AlertsThreshold = "future" },
		func(s *RulesetState) { (*rulesetCodeScanningTools(s.Policy))[0].Tool = "\n" },
		func(s *RulesetState) { (*rulesetCodeScanningTools(s.Policy))[0].Tool = "CodeQL" },
		func(s *RulesetState) { s.Policy.ID = rsID(99) },
	} {
		state := normalizedScanningSecurityAlertsState(true)
		mutate(&state)
		if got, err := NewRulesetObservation([]RulesetState{state}, 0); err == nil || !reflect.DeepEqual(got, RulesetObservation{}) {
			t.Fatal(got, err)
		}
		if raw, err := state.RawCodeScanningTools(); err == nil || raw != nil {
			t.Fatal(raw, err)
		}
	}
}

func TestScanningSecurityAlertsReconcileOwnsSelectedWholePolicyAndRemoval(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, mode := range []string{"equal", "different", "tool removed", "replacement", "rule removed", "empty tools"} {
			state := normalizedScanningSecurityAlertsState(companions)
			after := cloneRuleset(state.Policy)
			tools := rulesetCodeScanningTools(after)
			switch mode {
			case "different":
				(*tools)[2].SecurityAlertsThreshold = "all"
			case "tool removed":
				*tools = (*tools)[:2]
			case "replacement":
				(*tools)[2].Tool = "other tool"
				(*tools)[2].SecurityAlertsThreshold = "none"
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
			private := normalizedScanningSecurityAlertsState(false)
			private.ID = 9
			private.Policy.ID = rsID(9)
			private.Policy.Name = "PRIVATE_SCANNING"
			observed := rsObserved(t, private, state)
			changes, err := ReconcileRulesets(observed, rsDesired(t, after))
			if err != nil || len(changes) != 1 || changes[0].ID() != 7 || !reflect.DeepEqual(changes[0].Normalizations(), scanningSecurityAlertsRecords(state)) {
				t.Fatal(mode, changes, err)
			}
			records := changes[0].Normalizations()
			records.ScanningSecurityAlerts[0].RawValue = "changed"
			if !reflect.DeepEqual(changes[0].Display().Normalizations(), scanningSecurityAlertsRecords(state)) {
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
	canonical := normalizedScanningSecurityAlertsState(false)
	canonical.ScanningSecurityAlertsNormalizations = nil
	if changes, err := ReconcileRulesets(rsObserved(t, canonical), rsDesired(t, canonical.Policy)); err != nil || len(changes) != 0 {
		t.Fatal("canonical", changes, err)
	}
	desired := cloneRuleset(canonical.Policy)
	(*rulesetCodeScanningTools(desired))[0].SecurityAlertsThreshold = ""
	if _, err := NewRulesetDeclarations([]RulesetDeclaration{desired}); err == nil {
		t.Fatal("invented desired default")
	}
}
