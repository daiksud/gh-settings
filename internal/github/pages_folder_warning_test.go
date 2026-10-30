package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPagesFolderWarningRequiresOwnedAcquiredSource(t *testing.T) {
	for _, test := range []struct {
		name                 string
		observed             settings.PagesObservation
		declared, sink, want bool
	}{
		{"owned unknown", pagesFolderVerificationObservation(t, "main", "/future"), true, true, true},
		{"owned fresh different", pagesFolderVerificationObservation(t, "main", "/other"), true, true, true},
		{"unowned", pagesFolderVerificationObservation(t, "main", "/future"), false, true, false},
		{"canonical", pagesFolderVerificationObservation(t, "main", "/"), true, true, false},
		{"unacquired", settings.PagesObservation{}, true, true, false},
		{"nil sink", pagesFolderVerificationObservation(t, "main", "/future"), true, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &PagesClient{}
			var warnings []settings.ObservationNormalization
			if test.sink {
				client.NormalizationWarning = func(n settings.ObservationNormalization) { warnings = append(warnings, n) }
			}
			desired := settings.PagesDeclaration{HTTPSEnforced: actionsUIPtr(false)}
			if test.declared {
				desired.Source = &settings.PagesSource{Branch: "main", Path: "/docs"}
			}
			client.warnPagesNormalizations(test.observed, desired)
			var want []settings.ObservationNormalization
			if test.want {
				want = []settings.ObservationNormalization{test.observed.FolderNormalization()}
			}
			if !reflect.DeepEqual(warnings, want) {
				t.Fatal(warnings, want)
			}
		})
	}
}
