package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestLabelPolicyUI(t *testing.T) {
	policy := settings.LabelDeclaration{ID: rulesetPtr(int64(17)), Name: "bug", Color: rulesetPtr("abcdef"), Description: rulesetPtr("")}
	want := uimodel.Label{ID: rulesetPtr(int64(17)), Name: "bug", Color: rulesetPtr("#abcdef"), Description: rulesetPtr("")}
	got, err := LabelPolicyUI(policy)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("label projection %#v; want %#v; error=%v", got, want, err)
	}
	*got.ID, *got.Description, *got.Color = 99, "changed", "#123456"
	if *policy.ID != 17 || *policy.Description != "" || *policy.Color != "abcdef" {
		t.Fatal("label projection aliases policy")
	}
}

func TestLabelPolicyUIRejectsIncompletePolicies(t *testing.T) {
	for _, mutate := range []func(*settings.LabelDeclaration){
		func(p *settings.LabelDeclaration) { p.ID = nil },
		func(p *settings.LabelDeclaration) { p.ID = rulesetPtr(int64(0)) },
		func(p *settings.LabelDeclaration) { p.Color = nil },
		func(p *settings.LabelDeclaration) { p.Description = nil },
		func(p *settings.LabelDeclaration) { p.Color = rulesetPtr("#abcdef") },
	} {
		policy := settings.LabelDeclaration{ID: rulesetPtr(int64(17)), Name: "bug", Color: rulesetPtr("abcdef"), Description: rulesetPtr("")}
		mutate(&policy)
		got, err := LabelPolicyUI(policy)
		if err == nil || !reflect.DeepEqual(got, uimodel.Label{}) {
			t.Fatalf("incomplete label projected: %#v, %v", got, err)
		}
	}
}
