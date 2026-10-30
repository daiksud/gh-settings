package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPagesWarningsKeepIndependentOwnedFieldsAndFreshOrder(t *testing.T) {
	for _, test := range []struct {
		name               string
		mode, folder, sink bool
		observed           settings.PagesObservation
		want               []settings.ObservationNormalization
	}{
		{"mode only", true, false, true, pagesModeVerificationObservation(t, "FUTURE_MODE", "/future"), []settings.ObservationNormalization{{Path: settings.PagesModePath, RawValue: "FUTURE_MODE", Fallback: "deploy_from_a_branch"}}},
		{"both owned", true, true, true, pagesModeVerificationObservation(t, "FUTURE_MODE", "/future"), []settings.ObservationNormalization{{Path: settings.PagesModePath, RawValue: "FUTURE_MODE", Fallback: "deploy_from_a_branch"}, {Path: settings.PagesFolderPath, RawValue: "/future", Fallback: "root"}}},
		{"folder only", false, true, true, pagesModeVerificationObservation(t, "FUTURE_MODE", "/future"), []settings.ObservationNormalization{{Path: settings.PagesFolderPath, RawValue: "/future", Fallback: "root"}}},
		{"fresh mode", true, false, true, pagesModeVerificationObservation(t, "OTHER_MODE", "/"), []settings.ObservationNormalization{{Path: settings.PagesModePath, RawValue: "OTHER_MODE", Fallback: "deploy_from_a_branch"}}},
		{"unrelated", false, false, true, pagesModeVerificationObservation(t, "FUTURE_MODE", "/future"), nil},
		{"canonical mode", true, false, true, pagesFolderVerificationObservation(t, "main", "/future"), nil},
		{"unacquired", true, true, true, settings.PagesObservation{}, nil},
		{"no sink", true, true, false, pagesModeVerificationObservation(t, "FUTURE_MODE", "/future"), nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			var warnings []settings.ObservationNormalization
			c := &PagesClient{}
			if test.sink {
				c.NormalizationWarning = func(n settings.ObservationNormalization) { warnings = append(warnings, n) }
			}
			d := settings.PagesDeclaration{HTTPSEnforced: actionsUIPtr(false)}
			if test.mode {
				d.BuildType = actionsUIPtr("legacy")
			}
			if test.folder {
				d.Source = &settings.PagesSource{Branch: "main", Path: "/"}
			}
			c.warnPagesNormalizations(test.observed, d)
			if !reflect.DeepEqual(warnings, test.want) {
				t.Fatal(warnings, test.want)
			}
		})
	}
}
