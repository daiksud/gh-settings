package settings

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestEnvironmentRefTypeMappingKeepsTwoChoicesAndStrictNativeIdentity(t *testing.T) {
	if EnvironmentRefTypePath != "repository.settings.environments.deployment_branch_or_tag_rules.ref_type" || DefaultEnvironmentRefType != uimodel.EnvironmentRefType("branch") {
		t.Fatal("wrong ref-type path/default")
	}
	for _, test := range []struct {
		choice uimodel.EnvironmentRefType
		value  string
	}{{"branch", "branch"}, {"tag", "tag"}} {
		value, ok := EnvironmentRefTypeValueForChoice(test.choice)
		if !ok || value != test.value {
			t.Fatal("forward choice", value, ok)
		}
		choice, ok := EnvironmentRefTypeChoiceForValue(test.value)
		if !ok || choice != test.choice {
			t.Fatal("reverse choice", choice, ok)
		}
		desired := []EnvironmentPattern{{Type: test.value, Name: "release/*"}}
		if changes, err := ReconcileEnvironmentPatterns(nil, &desired); err != nil || len(changes) != 1 || changes[0].Operation != "create" || changes[0].Pattern != desired[0] {
			t.Fatal("native identity contract changed", changes, err)
		}
	}
	for _, unknown := range []string{"future", "", " \t", "Branch"} {
		if value, ok := EnvironmentRefTypeValueForChoice(uimodel.EnvironmentRefType(unknown)); ok || value != "" {
			t.Fatal("invalid UI choice accepted", value, ok)
		}
		if choice, ok := EnvironmentRefTypeChoiceForValue(unknown); ok || choice != "" {
			t.Fatal("unknown native type gained an exact UI choice", choice, ok)
		}
		empty := []EnvironmentPattern{}
		if _, err := ReconcileEnvironmentPatterns([]EnvironmentPatternState{{ID: 7, Pattern: EnvironmentPattern{Type: unknown, Name: "release/*"}}}, &empty); err == nil {
			t.Fatal("mapping activated native acquisition/reconciliation", unknown)
		}
	}
}
