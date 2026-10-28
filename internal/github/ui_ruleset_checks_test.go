package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIRulesetCheckSources(t *testing.T) {
	for _, test := range []struct {
		source uimodel.CheckSource
		id     *int64
	}{
		{uimodel.CheckSource{}, nil},
		{uimodel.CheckSource{AnySource: rulesetPtr(true)}, nil},
		{uimodel.CheckSource{App: rulesetPtr(int64(17))}, rulesetPtr(int64(17))},
	} {
		checks := []uimodel.StatusCheck{{Check: "build", Source: test.source}}
		ui := uiRulesetBase()
		ui.Rules.RequireStatusChecksToPass = &uimodel.StatusChecksRule{RequiredStatusChecks: &checks}
		got, err := RulesetDeclaration(ui)
		want := []settings.RequiredStatusCheck{{Context: "build", IntegrationID: test.id}}
		if err != nil || !reflect.DeepEqual((*got.Rules)[0].Parameters.RequiredStatusChecks, &want) {
			t.Fatalf("rules=%+v err=%v want %+v", got.Rules, err, want)
		}
		if checks[0].Source.App != nil {
			*checks[0].Source.App = 99
			if *(*(*got.Rules)[0].Parameters.RequiredStatusChecks)[0].IntegrationID != 17 {
				t.Fatal("source identity aliases caller")
			}
		}
	}
}

func TestUIRulesetCheckSingletonFlags(t *testing.T) {
	for _, value := range []bool{false, true} {
		for _, test := range []struct {
			ui  uimodel.StatusChecksRule
			api settings.RuleParameters
		}{
			{uimodel.StatusChecksRule{RequireBranchesToBeUpToDateBeforeMerging: &value}, settings.RuleParameters{StrictRequiredStatusChecksPolicy: &value}},
			{uimodel.StatusChecksRule{DoNotRequireStatusChecksOnCreation: &value}, settings.RuleParameters{DoNotEnforceOnCreate: &value}},
			{uimodel.StatusChecksRule{RequiredStatusChecks: rulesetPtr([]uimodel.StatusCheck{})}, settings.RuleParameters{RequiredStatusChecks: rulesetPtr([]settings.RequiredStatusCheck{})}},
		} {
			ui := uiRulesetBase()
			ui.Rules.RequireStatusChecksToPass = &test.ui
			got, err := RulesetDeclaration(ui)
			if err != nil || !reflect.DeepEqual((*got.Rules)[0].Parameters, &test.api) {
				t.Fatalf("rules=%+v err=%v want %+v", got.Rules, err, test.api)
			}
		}
	}
}

func TestUIRulesetCheckRawNullSource(t *testing.T) {
	const raw = `{"name":"Policy","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"required_status_checks","parameters":{"required_status_checks":[{"context":"ci","integration_id":null}],"strict_required_status_checks_policy":false,"do_not_enforce_on_create":true}}]}`
	policy, err := decodeRulesetPolicyFromTest([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	got, err := RulesetObservation(policy)
	want := &uimodel.StatusChecksRule{RequiredStatusChecks: rulesetPtr([]uimodel.StatusCheck{{Check: "ci", Source: uimodel.CheckSource{AnySource: rulesetPtr(true)}}}), RequireBranchesToBeUpToDateBeforeMerging: rulesetPtr(false), DoNotRequireStatusChecksOnCreation: rulesetPtr(true)}
	if err != nil || !reflect.DeepEqual(got.Rules.RequireStatusChecksToPass, want) {
		t.Fatalf("checks=%+v err=%v", got.Rules, err)
	}
}

func TestUIRulesetInvalidCheckSources(t *testing.T) {
	for _, source := range []uimodel.CheckSource{{AnySource: rulesetPtr(false)}, {App: rulesetPtr(int64(0))}, {App: rulesetPtr(int64(-1))}, {App: rulesetPtr(int64(17)), AnySource: rulesetPtr(true)}} {
		ui := uiRulesetBase()
		ui.Rules.RequireStatusChecksToPass = &uimodel.StatusChecksRule{RequiredStatusChecks: rulesetPtr([]uimodel.StatusCheck{{Check: "ci", Source: source}})}
		if _, err := RulesetDeclaration(ui); err == nil {
			t.Fatal("invalid check source accepted")
		}
	}
}
