package settings

import (
	"reflect"
	"testing"
)

func codeQLTestState() CodeQLConfigurationState {
	return CodeQLConfigurationState{Languages: []string{"python", "go"}, QuerySuite: "default", Runner: CodeQLRunner{Type: "standard_github_runner"}, RawThreatModel: "remote"}
}
func TestCodeQLConfigurationPresenceChoicesAndDefensiveState(t *testing.T) {
	o, err := NewCodeQLConfigurationObservation(codeQLTestState())
	if err != nil {
		t.Fatal(err)
	}
	n, c, err := CompareCodeQLConfiguration(o, CodeQLConfigurationDeclaration{})
	if err != nil || n != 0 || len(c) != 0 {
		t.Fatal(n, c, err)
	}
	languages := []string{"go", "python"}
	suite := "extended"
	d := CodeQLConfigurationDeclaration{Languages: &languages, QuerySuite: &suite}
	n, c, err = CompareCodeQLConfiguration(o, d)
	if err != nil || n != 2 || len(c) != 1 || c[0].Field() != "query_suite" {
		t.Fatal(n, c, err)
	}
	s, _ := o.State()
	s.Languages[0] = "ruby"
	again, _ := o.State()
	if reflect.DeepEqual(s, again) {
		t.Fatal("observation aliased")
	}
	for _, langs := range [][]string{nil, {}, {"go", "go"}, {"actions"}, {"Go"}} {
		if _, err := NewCodeQLConfigurationObservation(CodeQLConfigurationState{Languages: langs, QuerySuite: "default", Runner: codeQLTestState().Runner, RawThreatModel: "remote"}); err == nil {
			t.Fatal(langs)
		}
	}
	for _, r := range []CodeQLRunner{{}, {Type: "standard"}, {Type: "standard_github_runner", Label: "stale"}, {Type: "labeled_runner"}, {Type: "labeled_runner", Label: " \n"}} {
		if err := r.Validate(); err == nil {
			t.Fatal(r)
		}
	}
	disabled := false
	if err := (SecurityDeclaration{CodeQLDefaultSetup: &disabled, CodeQLConfiguration: d}).Validate(); err == nil {
		t.Fatal("disable and configuration accepted")
	}
	if _, err := (CodeQLConfigurationObservation{}).Declaration(); err == nil {
		t.Fatal("unacquired exported")
	}
}

func TestCodeQLPreviewThreatModelIsObservationOnly(t *testing.T) {
	state := codeQLTestState()
	state.RawThreatModel = "remote_and_local"
	observed, err := NewCodeQLConfigurationObservation(state)
	if err != nil {
		t.Fatal(err)
	}
	d, err := observed.Declaration()
	if err != nil {
		t.Fatal(err)
	}
	suite := "extended"
	desired := CodeQLConfigurationDeclaration{QuerySuite: &suite}
	expected := ExpectedCodeQLConfiguration(state, desired)
	if expected.RawThreatModel != "remote_and_local" {
		t.Fatal("Preview value changed", expected)
	}
	if n, _, err := CompareCodeQLConfiguration(observed, d); err != nil || n != 3 {
		t.Fatal(n, err)
	}
	if _, err := NewCodeQLConfigurationChange("threat_model", "remote_sources", "remote_and_local_sources"); err == nil {
		t.Fatal("excluded control accepted as managed change")
	}
}
