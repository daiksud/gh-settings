package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func pagesModeVerificationObservation(t *testing.T, rawMode, rawFolder string) settings.PagesObservation {
	t.Helper()
	folder := settings.ObservationNormalization{}
	state := settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: rawFolder}, CNAME: actionsUIPtr("EXAMPLE.COM"), HTTPSEnforced: true, Public: true}
	if rawFolder != "/" && rawFolder != "/docs" {
		state.Source.Path = "/"
		folder = settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: rawFolder, Fallback: "root"}
	}
	o, err := settings.NewPagesObservationWithModeNormalization(state, settings.ObservationNormalization{Path: settings.PagesModePath, RawValue: rawMode, Fallback: "deploy_from_a_branch"}, folder)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestPagesModeVerificationPreservesOmittedRawModeAndSource(t *testing.T) {
	for _, rawMode := range []string{"FUTURE_MODE", "", " \t"} {
		o := pagesModeVerificationObservation(t, rawMode, "/future")
		for _, test := range []struct {
			name string
			d    settings.PagesDeclaration
			want settings.PagesState
		}{
			{"mode omitted", settings.PagesDeclaration{HTTPSEnforced: actionsUIPtr(false)}, settings.PagesState{BuildType: rawMode, Source: &settings.PagesSource{Branch: "main", Path: "/future"}, CNAME: actionsUIPtr("example.com"), Public: true}},
			{"explicit legacy", settings.PagesDeclaration{BuildType: actionsUIPtr("legacy")}, settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/future"}, CNAME: actionsUIPtr("example.com"), HTTPSEnforced: true, Public: true}},
			{"explicit legacy source", settings.PagesDeclaration{BuildType: actionsUIPtr("legacy"), Source: &settings.PagesSource{Branch: "other", Path: "/docs"}}, settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "other", Path: "/docs"}, CNAME: actionsUIPtr("example.com"), HTTPSEnforced: true, Public: true}},
			{"explicit workflow", settings.PagesDeclaration{BuildType: actionsUIPtr("workflow")}, settings.PagesState{BuildType: "workflow", CNAME: actionsUIPtr("example.com"), HTTPSEnforced: true, Public: true}},
		} {
			t.Run(rawMode+"/"+test.name, func(t *testing.T) {
				got, err := preparePagesVerificationState(o, test.d)
				if err != nil || !reflect.DeepEqual(got, test.want) {
					t.Fatal(got, test.want, err)
				}
			})
		}
		if got, err := preparePagesVerificationState(o, settings.PagesDeclaration{Source: &settings.PagesSource{Branch: "main", Path: "/"}}); err == nil || !reflect.DeepEqual(got, settings.PagesState{}) {
			t.Fatal("unknown mode proves source applicability", got, err)
		}
	}
}

func TestPagesModeVerificationRejectsProjectedEqualityAndRawCompanionDrift(t *testing.T) {
	for _, test := range []struct {
		name, expected, actual string
		want                   bool
	}{
		{"unchanged omitted unknown", "FUTURE_MODE", "FUTURE_MODE", true},
		{"changed omitted unknown", "FUTURE_MODE", "OTHER_MODE", false},
		{"canonicalized omitted unknown", "FUTURE_MODE", "legacy", false},
		{"declared legacy retained unknown", "legacy", "FUTURE_MODE", false},
		{"declared legacy confirmed", "legacy", "legacy", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			expected := settings.PagesState{BuildType: test.expected, Source: &settings.PagesSource{Branch: "main", Path: "/"}, CNAME: actionsUIPtr("example.com"), HTTPSEnforced: true, Public: true}
			var actual settings.PagesObservation
			if test.actual == "legacy" {
				var err error
				actual, err = settings.NewPagesObservation(settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/"}, CNAME: actionsUIPtr("EXAMPLE.COM"), HTTPSEnforced: true, Public: true})
				if err != nil {
					t.Fatal(err)
				}
			} else {
				actual = pagesModeVerificationObservation(t, test.actual, "/")
			}
			got, err := samePagesVerificationState(expected, actual)
			if err != nil || got != test.want {
				t.Fatal(got, test.want, err)
			}
		})
	}
}
