package settings

import "testing"

func TestPagesFolderObservationRetainsOwnedRawSource(t *testing.T) {
	for _, raw := range []string{"/future", "", " \t"} {
		source := PagesSource{Branch: "release/site", Path: "/"}
		record := ObservationNormalization{Path: PagesFolderPath, RawValue: raw, Fallback: "root"}
		observed, err := NewPagesObservationWithFolderNormalization(PagesState{BuildType: "legacy", Source: &source, HTTPSEnforced: true, Public: false}, record)
		if err != nil {
			t.Fatal(err)
		}
		state, err := observed.Values()
		if err != nil || state.Source == nil || *state.Source != (PagesSource{Branch: "release/site", Path: "/"}) || !state.HTTPSEnforced || state.Public || observed.FolderNormalization() != record {
			t.Fatal(state, observed.FolderNormalization(), err)
		}
		wire, err := observed.RawSource()
		if err != nil || wire == nil || *wire != (PagesSource{Branch: "release/site", Path: raw}) {
			t.Fatal(wire, err)
		}
		source.Branch = "mutated caller"
		state.Source.Path = "mutated effective"
		wire.Path = "mutated raw"
		record.RawValue = "mutated record"
		owned := observed.FolderNormalization()
		owned.RawValue = "mutated copy"
		wire, err = observed.RawSource()
		if err != nil || wire.Branch != "release/site" || wire.Path != raw || observed.FolderNormalization().RawValue != raw {
			t.Fatal("aliased evidence", wire, err)
		}
	}
	for _, path := range []string{"/", "/docs"} {
		observed, err := NewPagesObservation(PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: path}})
		if err != nil {
			t.Fatal(err)
		}
		wire, err := observed.RawSource()
		if err != nil || wire == nil || wire.Path != path || observed.FolderNormalization() != (ObservationNormalization{}) {
			t.Fatal(wire, err)
		}
	}
	workflow, err := NewPagesObservation(PagesState{BuildType: "workflow", Source: &PagesSource{Branch: "latent", Path: "/future"}})
	if err != nil {
		t.Fatal(err)
	}
	if wire, err := workflow.RawSource(); err != nil || wire != nil {
		t.Fatal("inactive source acquired", wire, err)
	}
	if wire, err := (PagesObservation{}).RawSource(); err == nil || wire != nil {
		t.Fatal("unacquired source invented", wire, err)
	}
}

func TestPagesFolderObservationRejectsForgedOrIncompleteSource(t *testing.T) {
	valid := ObservationNormalization{Path: PagesFolderPath, RawValue: "/future", Fallback: "root"}
	canonical := PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: "/"}}
	for _, test := range []struct {
		state  PagesState
		record ObservationNormalization
	}{
		{canonical, ObservationNormalization{}},
		{canonical, ObservationNormalization{Path: PagesFolderPath, RawValue: "/", Fallback: "root"}},
		{canonical, ObservationNormalization{Path: PagesFolderPath, RawValue: "/docs", Fallback: "root"}},
		{canonical, ObservationNormalization{Path: PagesFolderPath, RawValue: "/future", Fallback: "docs"}},
		{canonical, ObservationNormalization{Path: "repository.settings.pages.build_and_deployment.source", RawValue: "/future", Fallback: "root"}},
		{PagesState{BuildType: "workflow", Source: canonical.Source}, valid},
		{PagesState{BuildType: "FUTURE", Source: canonical.Source}, valid},
		{PagesState{BuildType: "legacy"}, valid},
		{PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "", Path: "/"}}, valid},
		{PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: "/docs"}}, valid},
	} {
		observed, err := NewPagesObservationWithFolderNormalization(test.state, test.record)
		if err == nil {
			t.Fatal("invalid normalization accepted", test)
		}
		if _, err := observed.Values(); err == nil {
			t.Fatal("invalid acquired state escaped", observed)
		}
		if observed.FolderNormalization() != (ObservationNormalization{}) {
			t.Fatal("partial evidence escaped", observed)
		}
	}
	if _, err := NewPagesObservation(PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: "/future"}}); err == nil {
		t.Fatal("canonical constructor relaxed")
	}
}
