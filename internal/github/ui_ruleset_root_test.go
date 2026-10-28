package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func rulesetPtr[T any](v T) *T { return &v }

func TestUIRulesetRootDeclaration(t *testing.T) {
	for _, test := range []struct {
		name             string
		ui               uimodel.Ruleset
		target           string
		include, exclude []string
	}{
		{"branch", uimodel.Ruleset{RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.EnforcementStatus("active")), TargetBranches: &uimodel.BranchTargets{IncludeDefaultBranch: rulesetPtr(true), IncludeAllBranches: rulesetPtr(false), IncludeByPattern: rulesetPtr([]string{"release/*"}), ExcludeByPattern: rulesetPtr([]string{"release/test"})}}, "branch", []string{"~DEFAULT_BRANCH", "refs/heads/release/*"}, []string{"refs/heads/release/test"}},
		{"tag", uimodel.Ruleset{RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.EnforcementStatus("disabled")), TargetTags: &uimodel.TagTargets{IncludeAllTags: rulesetPtr(true), IncludeByPattern: rulesetPtr([]string{"v*"})}}, "tag", []string{"~ALL", "refs/tags/v*"}, []string{}},
		{"explicit empty", uimodel.Ruleset{RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.EnforcementStatus("active")), TargetBranches: &uimodel.BranchTargets{}}, "branch", []string{}, []string{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := RulesetDeclaration(test.ui)
			want := settings.RulesetDeclaration{Name: "Policy", Target: &test.target, Enforcement: rulesetPtr(string(*test.ui.EnforcementStatus)), Conditions: &settings.RulesetConditions{RefName: &settings.RefNameCondition{Include: &test.include, Exclude: &test.exclude}}}
			want.Rules = rulesetPtr([]settings.RulesetRule{})
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("declaration=%+v err=%v want %+v", got, err, want)
			}
		})
	}
}

func TestUIRulesetRootObservation(t *testing.T) {
	source := settings.RulesetDeclaration{ID: rulesetPtr(int64(7)), Name: "Policy", Target: rulesetPtr("branch"), Enforcement: rulesetPtr("active"), Conditions: &settings.RulesetConditions{RefName: &settings.RefNameCondition{Include: rulesetPtr([]string{"~DEFAULT_BRANCH", "refs/heads/release/*"}), Exclude: rulesetPtr([]string{})}}, BypassActors: rulesetPtr([]settings.BypassActor{}), Rules: rulesetPtr([]settings.RulesetRule{})}
	got, err := RulesetObservation(source)
	want := uimodel.Ruleset{ID: rulesetPtr(int64(7)), RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.EnforcementStatus("active")), TargetBranches: &uimodel.BranchTargets{IncludeDefaultBranch: rulesetPtr(true), IncludeAllBranches: rulesetPtr(false), IncludeByPattern: rulesetPtr([]string{"release/*"}), ExcludeByPattern: rulesetPtr([]string{})}, BypassList: rulesetPtr([]uimodel.BypassEntry{})}
	want.Rules = uiObservedRuleFlags()
	if err != nil || !reflect.DeepEqual(got.Ruleset, want) {
		t.Fatalf("observation=%+v err=%v want %+v", got, err, want)
	}
	for _, enforcement := range []string{"evaluate", "UNKNOWN"} {
		*source.Enforcement = enforcement
		if _, err := RulesetObservation(source); err == nil {
			t.Fatal("out-of-UI enforcement accepted")
		}
	}
	*source.Enforcement = "active"
	*source.Conditions.RefName.Include = []string{"refs/tags/v*"}
	if _, err := RulesetObservation(source); err == nil {
		t.Fatal("wrong target prefix accepted")
	}
}

func TestUIRulesetRootRequiredAndOwned(t *testing.T) {
	for _, ui := range []uimodel.Ruleset{
		{RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.EnforcementStatus("active"))},
		{RulesetName: "Policy", TargetBranches: &uimodel.BranchTargets{}},
		{RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.EnforcementStatus("evaluate")), TargetBranches: &uimodel.BranchTargets{}},
		{RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.EnforcementStatus("active")), TargetBranches: &uimodel.BranchTargets{}, TargetTags: &uimodel.TagTargets{}},
	} {
		if _, err := RulesetDeclaration(ui); err == nil {
			t.Fatal("invalid root accepted")
		}
	}
	ui := uimodel.Ruleset{ID: rulesetPtr(int64(7)), RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.EnforcementStatus("active")), TargetBranches: &uimodel.BranchTargets{IncludeByPattern: rulesetPtr([]string{"main"})}}
	got, err := RulesetDeclaration(ui)
	if err != nil {
		t.Fatal(err)
	}
	*ui.ID = 99
	(*ui.TargetBranches.IncludeByPattern)[0] = "changed"
	if *got.ID != 7 || (*got.Conditions.RefName.Include)[0] != "refs/heads/main" {
		t.Fatal("converted intent aliases caller")
	}
}

func TestUIRulesetBlankPatterns(t *testing.T) {
	for _, pattern := range []string{"", "   "} {
		for _, tag := range []bool{false, true} {
			for _, exclude := range []bool{false, true} {
				ui := uimodel.Ruleset{RulesetName: "Policy", EnforcementStatus: rulesetPtr(uimodel.EnforcementStatus("active"))}
				patterns := []string{pattern}
				if tag {
					ui.TargetTags = &uimodel.TagTargets{}
					if exclude {
						ui.TargetTags.ExcludeByPattern = &patterns
					} else {
						ui.TargetTags.IncludeByPattern = &patterns
					}
				} else {
					ui.TargetBranches = &uimodel.BranchTargets{}
					if exclude {
						ui.TargetBranches.ExcludeByPattern = &patterns
					} else {
						ui.TargetBranches.IncludeByPattern = &patterns
					}
				}
				if _, err := RulesetDeclaration(ui); err == nil {
					t.Fatalf("blank desired pattern accepted: tag=%v exclude=%v pattern=%q", tag, exclude, pattern)
				}
				prefix, target := "refs/heads/", "branch"
				if tag {
					prefix, target = "refs/tags/", "tag"
				}
				include, excluded := []string{}, []string{}
				if exclude {
					excluded = []string{prefix + pattern}
				} else {
					include = []string{prefix + pattern}
				}
				source := settings.RulesetDeclaration{Name: "Policy", Target: &target, Enforcement: rulesetPtr("active"), Conditions: &settings.RulesetConditions{RefName: &settings.RefNameCondition{Include: &include, Exclude: &excluded}}, BypassActors: rulesetPtr([]settings.BypassActor{}), Rules: rulesetPtr([]settings.RulesetRule{})}
				if _, err := RulesetObservation(source); err == nil {
					t.Fatalf("blank observed suffix accepted: tag=%v exclude=%v", tag, exclude)
				}
			}
		}
	}
}
