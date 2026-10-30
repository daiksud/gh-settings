package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func pagesFolderVerificationObservation(t *testing.T, branch, folder string) settings.PagesObservation {
	t.Helper()
	s := settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: branch, Path: folder}, CNAME: actionsUIPtr("EXAMPLE.COM"), HTTPSEnforced: true, Public: true}
	var o settings.PagesObservation
	var err error
	if folder == "/" || folder == "/docs" {
		o, err = settings.NewPagesObservation(s)
	} else {
		s.Source.Path = "/"
		o, err = settings.NewPagesObservationWithFolderNormalization(s, settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: folder, Fallback: "root"})
	}
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestPreparePagesVerificationPreservesOnlyOmittedRawSource(t *testing.T) {
	o := pagesFolderVerificationObservation(t, "main", "/future")
	for _, test := range []struct {
		name string
		d    settings.PagesDeclaration
		want settings.PagesState
	}{
		{"root", settings.PagesDeclaration{Source: &settings.PagesSource{Branch: "main", Path: "/"}}, settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/"}, CNAME: actionsUIPtr("example.com"), HTTPSEnforced: true, Public: true}},
		{"docs and branch", settings.PagesDeclaration{Source: &settings.PagesSource{Branch: "release/site", Path: "/docs"}}, settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "release/site", Path: "/docs"}, CNAME: actionsUIPtr("example.com"), HTTPSEnforced: true, Public: true}},
		{"omitted source", settings.PagesDeclaration{HTTPSEnforced: actionsUIPtr(false)}, settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/future"}, CNAME: actionsUIPtr("example.com"), Public: true}},
		{"remove domain", settings.PagesDeclaration{CNAME: &settings.PagesDomain{}}, settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/future"}, HTTPSEnforced: true, Public: true}},
		{"canonical domain", settings.PagesDeclaration{CNAME: &settings.PagesDomain{Value: actionsUIPtr("NEW.EXAMPLE.COM")}}, settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/future"}, CNAME: actionsUIPtr("new.example.com"), HTTPSEnforced: true, Public: true}},
		{"workflow drops source", settings.PagesDeclaration{BuildType: actionsUIPtr("workflow")}, settings.PagesState{BuildType: "workflow", CNAME: actionsUIPtr("example.com"), HTTPSEnforced: true, Public: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := preparePagesVerificationState(o, test.d)
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatal(got, test.want, err)
			}
			if got.Source != nil {
				got.Source.Path = "caller mutation"
			}
			raw, err := o.RawSource()
			if err != nil || raw.Path != "/future" {
				t.Fatal(raw, err)
			}
		})
	}
	got, err := preparePagesVerificationState(settings.PagesObservation{}, settings.PagesDeclaration{})
	if err == nil || !reflect.DeepEqual(got, settings.PagesState{}) {
		t.Fatal(got, err)
	}
}

func TestPagesVerificationRequiresExactRawSourceAndConfiguredCompanions(t *testing.T) {
	expected := settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/future"}, CNAME: actionsUIPtr("example.com"), HTTPSEnforced: true, Public: true}
	for _, test := range []struct {
		name, branch, folder string
		want                 bool
	}{
		{"preserved unknown", "main", "/future", true}, {"other unknown", "main", "/other", false},
		{"silently normalized", "main", "/", false}, {"branch drift", "other", "/future", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := samePagesVerificationState(expected, pagesFolderVerificationObservation(t, test.branch, test.folder))
			if err != nil || got != test.want {
				t.Fatal(got, test.want, err)
			}
		})
	}
	expected.Source.Path = "/"
	for _, folder := range []string{"/", "/future"} {
		got, err := samePagesVerificationState(expected, pagesFolderVerificationObservation(t, "main", folder))
		if err != nil || got != (folder == "/") {
			t.Fatal(folder, got, err)
		}
	}
	state := settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/"}, CNAME: actionsUIPtr("EXAMPLE.COM"), HTTPSEnforced: true, Public: true}
	for _, field := range []string{"domain", "https", "visibility"} {
		changed := state
		switch field {
		case "domain":
			changed.CNAME = actionsUIPtr("other.example.com")
		case "https":
			changed.HTTPSEnforced = false
		case "visibility":
			changed.Public = false
		}
		actual, err := settings.NewPagesObservation(changed)
		if err != nil {
			t.Fatal(err)
		}
		got, err := samePagesVerificationState(expected, actual)
		if err != nil || got {
			t.Fatal(field, got, err)
		}
	}
	workflow, err := settings.NewPagesObservation(settings.PagesState{BuildType: "workflow", Source: &settings.PagesSource{Branch: "latent", Path: "/future"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := samePagesVerificationState(settings.PagesState{BuildType: "workflow"}, workflow); err != nil || !got {
		t.Fatal(got, err)
	}
	if got, err := samePagesVerificationState(expected, settings.PagesObservation{}); err == nil || got {
		t.Fatal(got, err)
	}
}
