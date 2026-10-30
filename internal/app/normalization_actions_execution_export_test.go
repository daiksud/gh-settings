package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestActionsExecutionExportWarningsRequireVisibleCompleteObservation(t *testing.T) {
	for _, test := range []struct {
		name                               string
		include, enabled, normalized, sink bool
		failure                            string
	}{
		{"visible", true, true, true, true, ""},
		{"hidden", true, false, true, true, ""},
		{"canonical", true, true, false, true, ""},
		{"unrequested", false, true, true, true, ""},
		{"failed read with value", true, true, true, true, "read"},
		{"unacquired selected", true, true, false, true, "selected"},
		{"no sink", true, true, true, false, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			observed := settings.ActionsPolicyState{Enabled: test.enabled, AllowedActions: "all", SHAPinningRequired: true}
			record := settings.ObservationNormalization{Path: settings.ActionsPermissionsPath, RawValue: "FUTURE_POLICY", Fallback: string(settings.DefaultActionsPermissionsChoice)}
			if test.normalized {
				observed.AllowedActionsNormalization = record
			}
			if test.failure == "selected" {
				observed.AllowedActions = "selected"
			}
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			cause := errors.New("Actions acquisition failed")
			reads := 0
			reader.Actions = func(_ repository.Repository, selected bool) (settings.ActionsPolicyState, error) {
				reads++
				if !selected {
					t.Fatal("export must acquire active selected settings")
				}
				if test.failure == "read" {
					return observed, cause
				}
				return observed, nil
			}
			saved := false
			checkout := ExportCheckout{Repository: checkoutRepository, IncludeActions: test.include, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					ui, err := github.UIFromExportDeclaration(d)
					if err != nil {
						return err
					}
					if test.include {
						want := settings.DefaultActionsPermissionsChoice
						if !test.enabled {
							want = uimodel.DisableActions
						}
						if ui.Settings.Actions.General.ActionsPermissions == nil || *ui.Settings.Actions.General.ActionsPermissions != want {
							t.Fatal("incorrect exported radio", ui.Settings.Actions)
						}
						if d.Actions.SHAPinningRequired == nil || !*d.Actions.SHAPinningRequired {
							t.Fatal("SHA value lost")
						}
					}
					saved = true
					return nil
				}, nil
			}}
			var warnings []settings.ObservationNormalization
			if test.sink {
				checkout.NormalizationWarning = func(n settings.ObservationNormalization) { warnings = append(warnings, n) }
			}
			err := Export(checkout, reader)
			var want []settings.ObservationNormalization
			if test.include && test.enabled && test.normalized && test.sink && test.failure == "" {
				want = []settings.ObservationNormalization{record}
			}
			wantReads := 0
			if test.include {
				wantReads = 1
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
