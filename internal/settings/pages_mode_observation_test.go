package settings

import "testing"

func TestPagesModeObservationOwnsCompleteTupleAndRawRecords(t *testing.T) {
	for _, rawMode := range []string{"future", "", " \t"} {
		for _, rawFolder := range []string{"/docs", "/future"} {
			state := PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "release/site", Path: "/docs"}, CNAME: pagesPtr("EXAMPLE.COM"), HTTPSEnforced: true, Public: true}
			modeRecord := ObservationNormalization{Path: PagesModePath, RawValue: rawMode, Fallback: "deploy_from_a_branch"}
			folderRecord := ObservationNormalization{}
			if rawFolder == "/future" {
				state.Source.Path = "/"
				folderRecord = ObservationNormalization{Path: PagesFolderPath, RawValue: rawFolder, Fallback: "root"}
			}
			o, err := NewPagesObservationWithModeNormalization(state, modeRecord, folderRecord)
			if err != nil {
				t.Fatal(err)
			}
			effective, err := o.Values()
			mode, modeErr := o.RawBuildType()
			source, sourceErr := o.RawSource()
			if err != nil || modeErr != nil || sourceErr != nil || effective.BuildType != "legacy" || effective.Source.Branch != "release/site" || source.Path != rawFolder || mode != rawMode || o.ModeNormalization() != modeRecord || o.FolderNormalization() != folderRecord || *effective.CNAME != "example.com" {
				t.Fatal(effective, mode, source, err, modeErr, sourceErr)
			}
			state.Source.Branch = "mutated caller"
			*state.CNAME = "mutated.example.com"
			source.Path = "mutated raw"
			effective.Source.Path = "mutated effective"
			modeRecord.RawValue = "mutated record"
			mode, modeErr = o.RawBuildType()
			source, sourceErr = o.RawSource()
			if modeErr != nil || sourceErr != nil || mode != rawMode || source.Path != rawFolder || source.Branch != "release/site" {
				t.Fatal("aliased provenance", mode, source, modeErr, sourceErr)
			}
		}
	}
	for _, mode := range []string{"legacy", "workflow"} {
		state := PagesState{BuildType: mode, Source: &PagesSource{Branch: "main", Path: "/"}}
		o, err := NewPagesObservation(state)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := o.RawBuildType()
		if err != nil || raw != mode || o.ModeNormalization() != (ObservationNormalization{}) {
			t.Fatal(raw, err)
		}
	}
	if raw, err := (PagesObservation{}).RawBuildType(); err == nil || raw != "" {
		t.Fatal("unacquired mode invented", raw, err)
	}
}

func TestPagesModeObservationRejectsForgedOrIncompleteEvidence(t *testing.T) {
	state := PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: "/"}}
	valid := ObservationNormalization{Path: PagesModePath, RawValue: "future", Fallback: "deploy_from_a_branch"}
	for _, record := range []ObservationNormalization{
		{}, {Path: PagesFolderPath, RawValue: "future", Fallback: "deploy_from_a_branch"},
		{Path: PagesModePath, RawValue: "future", Fallback: "github_actions"},
		{Path: PagesModePath, RawValue: "legacy", Fallback: "deploy_from_a_branch"},
		{Path: PagesModePath, RawValue: "workflow", Fallback: "deploy_from_a_branch"},
	} {
		o, err := NewPagesObservationWithModeNormalization(state, record, ObservationNormalization{})
		if err == nil {
			t.Fatal("forged mode", record)
		}
		if _, err = o.Values(); err == nil {
			t.Fatal("failed constructor acquired")
		}
	}
	for _, invalid := range []PagesState{
		{BuildType: "workflow"}, {BuildType: "future", Source: state.Source}, {BuildType: "legacy"},
		{BuildType: "legacy", Source: &PagesSource{Branch: "bad branch", Path: "/"}},
		{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: "/future"}},
		{BuildType: "legacy", Source: state.Source, CNAME: pagesPtr("invalid")},
	} {
		o, err := NewPagesObservationWithModeNormalization(invalid, valid, ObservationNormalization{})
		if err == nil {
			t.Fatal("incomplete tuple accepted", invalid)
		}
		if _, err = o.Values(); err == nil {
			t.Fatal("invalid acquired")
		}
	}
	for _, record := range []ObservationNormalization{
		{Path: PagesModePath, RawValue: "/future", Fallback: "root"},
		{Path: PagesFolderPath, RawValue: "/", Fallback: "root"},
		{Path: PagesFolderPath, RawValue: "/future", Fallback: "docs"},
	} {
		if _, err := NewPagesObservationWithModeNormalization(state, valid, record); err == nil {
			t.Fatal("forged folder", record)
		}
	}
}
