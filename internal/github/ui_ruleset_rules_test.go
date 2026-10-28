package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func uiObservedRuleFlags() uimodel.RulesetRules {
	return uimodel.RulesetRules{RestrictDeletions: rulesetPtr(false), BlockForcePushes: rulesetPtr(false), RequireLinearHistory: rulesetPtr(false), RequireSignedCommits: rulesetPtr(false)}
}

func TestUIRulesetSimpleRuleControls(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, test := range []struct {
			kind     string
			ui       uimodel.RulesetRules
			observed uimodel.RulesetRules
		}{
			{"deletion", uimodel.RulesetRules{RestrictDeletions: &enabled}, func() uimodel.RulesetRules { v := uiObservedRuleFlags(); v.RestrictDeletions = &enabled; return v }()},
			{"non_fast_forward", uimodel.RulesetRules{BlockForcePushes: &enabled}, func() uimodel.RulesetRules { v := uiObservedRuleFlags(); v.BlockForcePushes = &enabled; return v }()},
			{"required_linear_history", uimodel.RulesetRules{RequireLinearHistory: &enabled}, func() uimodel.RulesetRules { v := uiObservedRuleFlags(); v.RequireLinearHistory = &enabled; return v }()},
			{"required_signatures", uimodel.RulesetRules{RequireSignedCommits: &enabled}, func() uimodel.RulesetRules { v := uiObservedRuleFlags(); v.RequireSignedCommits = &enabled; return v }()},
		} {
			ui := uiRulesetBase()
			ui.Rules = test.ui
			got, err := RulesetDeclaration(ui)
			want := []settings.RulesetRule{}
			if enabled {
				want = append(want, settings.RulesetRule{Type: test.kind})
			}
			if err != nil || got.Rules == nil || !reflect.DeepEqual(*got.Rules, want) {
				t.Fatalf("rules=%+v err=%v want %+v", got.Rules, err, want)
			}
			source := apiRulesetBase()
			source.Rules = &want
			observed, err := RulesetObservation(source)
			if err != nil || !reflect.DeepEqual(observed.Rules, test.observed) {
				t.Fatalf("rules=%+v err=%v want %+v", observed.Rules, err, test.observed)
			}
		}
	}
}
