package settings

import (
	"reflect"
	"testing"
)

func codeQLTestState() CodeQLConfigurationState {
	return CodeQLConfigurationState{Languages: []string{"python", "go"}, QuerySuite: "default", Runner: CodeQLRunner{Type: "standard_github_runner"}, ThreatModel: "remote_sources"}
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
		if _, err := NewCodeQLConfigurationObservation(CodeQLConfigurationState{Languages: langs, QuerySuite: "default", Runner: codeQLTestState().Runner, ThreatModel: "remote_sources"}); err == nil {
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

func TestCodeQLThreatModelSelectionAndPreservation(t *testing.T) {
	for _, value := range []string{"remote_sources", "remote_and_local_sources"} {
		state := codeQLTestState()
		state.ThreatModel = value
		observed, err := NewCodeQLConfigurationObservation(state)
		if err != nil {
			t.Fatal(err)
		}
		all, err := observed.Declaration()
		if err != nil || all.ThreatModel == nil || *all.ThreatModel != value {
			t.Fatal(all, err)
		}
		if n, changes, err := CompareCodeQLConfiguration(observed, all); err != nil || n != 4 || len(changes) != 0 {
			t.Fatal(n, changes, err)
		}
		suite := "extended"
		expected := ExpectedCodeQLConfiguration(state, CodeQLConfigurationDeclaration{QuerySuite: &suite})
		if expected.ThreatModel != value {
			t.Fatal("omitted threat model changed", expected)
		}
		for _, desired := range []string{"remote_sources", "remote_and_local_sources"} {
			d := CodeQLConfigurationDeclaration{ThreatModel: &desired}
			n, changes, err := CompareCodeQLConfiguration(observed, d)
			if err != nil || n != 1 || (len(changes) == 1) != (value != desired) {
				t.Fatal(n, changes, err)
			}
			expected = ExpectedCodeQLConfiguration(state, d)
			want := copyCodeQLState(state)
			want.ThreatModel = desired
			if !reflect.DeepEqual(expected, want) {
				t.Fatal("omitted companions changed", expected, want)
			}
		}
	}
	for _, value := range []string{"", "remote", "remote_and_local", "future", "REMOTE_SOURCES"} {
		d := CodeQLConfigurationDeclaration{ThreatModel: &value}
		state := codeQLTestState()
		state.ThreatModel = value
		if d.Validate() == nil {
			t.Fatal("accepted invalid desired threat model", value)
		}
		if _, err := NewCodeQLConfigurationObservation(state); err == nil {
			t.Fatal("accepted invalid observation", value)
		}
		if _, err := NewCodeQLConfigurationChange("threat_model", "remote_sources", value); err == nil {
			t.Fatal("accepted invalid change", value)
		}
	}
}
