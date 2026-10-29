package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestLegacyPagesFolderProjectionPreservesBranchAndRawFolder(t *testing.T) {
	for _, raw := range []string{"/", "/docs", "/future", "", " \t"} {
		branch := "release/site"
		prepared, err := prepareLegacyPagesFolderObservation(&branch, &raw)
		if err != nil {
			t.Fatal(err)
		}
		wantPath := raw
		wantRecord := settings.ObservationNormalization{}
		if raw != "/" && raw != "/docs" {
			wantPath = "/"
			wantRecord = settings.ObservationNormalization{Path: "repository.settings.pages.build_and_deployment.select_folder", RawValue: raw, Fallback: "root"}
		}
		if prepared.Source != (settings.PagesSource{Branch: "release/site", Path: wantPath}) || prepared.Normalization != wantRecord {
			t.Fatal(raw, prepared, wantRecord)
		}
		branch = "mutated caller"
		raw = "mutated caller"
		if prepared.Source.Branch != "release/site" || prepared.Source.Path != wantPath || prepared.Normalization != wantRecord {
			t.Fatal("source aliases caller", prepared)
		}
	}
}

func TestLegacyPagesFolderProjectionRejectsIncompleteOrInvalidBranch(t *testing.T) {
	for _, test := range []struct{ branch, path *string }{
		{nil, nil}, {nil, actionsUIPtr("/")}, {actionsUIPtr("main"), nil},
		{actionsUIPtr(""), actionsUIPtr("/future")}, {actionsUIPtr("bad branch"), actionsUIPtr("/future")},
	} {
		prepared, err := prepareLegacyPagesFolderObservation(test.branch, test.path)
		if err == nil || prepared.Source != (settings.PagesSource{}) || prepared.Normalization != (settings.ObservationNormalization{}) {
			t.Fatal("incomplete source invented", test, prepared, err)
		}
	}
}
