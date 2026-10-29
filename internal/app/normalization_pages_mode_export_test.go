package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPagesModeExportWarningsRequireSelectedCompleteObservation(t *testing.T) {
	for _, test := range []struct {
		name                        string
		include, mode, folder, sink bool
		failure                     string
	}{
		{"mode", true, true, false, true, ""}, {"coupled mode folder", true, true, true, true, ""},
		{"canonical", true, false, false, true, ""}, {"unrequested", false, true, true, true, ""},
		{"failed read with value", true, true, true, true, "read"}, {"unacquired", true, false, false, true, "unacquired"},
		{"no sink", true, true, true, false, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "release/site", Path: "/"}, HTTPSEnforced: true, Public: false}
			modeRecord := settings.ObservationNormalization{Path: settings.PagesModePath, RawValue: "FUTURE_MODE", Fallback: "deploy_from_a_branch"}
			folderRecord := settings.ObservationNormalization{}
			if test.folder {
				folderRecord = settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: "/future", Fallback: "root"}
			}
			var observed settings.PagesObservation
			var err error
			if test.mode {
				observed, err = settings.NewPagesObservationWithModeNormalization(state, modeRecord, folderRecord)
			} else {
				observed, err = settings.NewPagesObservation(state)
			}
			if err != nil {
				t.Fatal(err)
			}
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			cause := errors.New("Pages unavailable")
			reads := 0
			reader.Pages = func(repository.Repository) (settings.PagesObservation, error) {
				reads++
				switch test.failure {
				case "read":
					return observed, cause
				case "unacquired":
					return settings.PagesObservation{}, nil
				}
				return observed, nil
			}
			var warnings []settings.ObservationNormalization
			saved := false
			checkout := ExportCheckout{Repository: checkoutRepository, IncludePages: test.include, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					if _, err := github.UIFromExportDeclaration(d); err != nil {
						return err
					}
					if test.include && (d.Pages.BuildType == nil || *d.Pages.BuildType != "legacy" || d.Pages.Source == nil || *d.Pages.Source != (settings.PagesSource{Branch: "release/site", Path: "/"})) {
						t.Fatal("effective tuple lost", d.Pages)
					}
					saved = true
					return nil
				}, nil
			}}
			if test.sink {
				checkout.NormalizationWarning = func(n settings.ObservationNormalization) { warnings = append(warnings, n) }
			}
			err = Export(checkout, reader)
			wantReads := 0
			var want []settings.ObservationNormalization
			if test.include {
				wantReads = 1
				if test.failure == "" && test.sink && test.mode {
					want = append(want, modeRecord)
					if test.folder {
						want = append(want, folderRecord)
					}
				}
			}
			if (err != nil) != (test.failure != "") || saved != (test.failure == "") || reads != wantReads || !reflect.DeepEqual(warnings, want) {
				t.Fatal(saved, reads, warnings, want, err)
			}
			if test.failure == "read" && !errors.Is(err, cause) {
				t.Fatal("cause lost", err)
			}
		})
	}
}
