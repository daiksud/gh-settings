package settings

import (
	"reflect"
	"strings"
	"testing"
)

func TestPagesModeFallbackCannotProveSourceApplicability(t *testing.T) {
	state := PagesState{BuildType: "legacy", Source: &PagesSource{Branch: "main", Path: "/"}, HTTPSEnforced: true, Public: true}
	unknown, err := NewPagesObservationWithModeNormalization(state, ObservationNormalization{Path: PagesModePath, RawValue: "UNMANAGED_MODE_CANARY", Fallback: "deploy_from_a_branch"}, ObservationNormalization{})
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := NewPagesObservation(state)
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := NewPagesObservation(PagesState{BuildType: "workflow", HTTPSEnforced: true, Public: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name     string
		observed PagesObservation
		desired  PagesDeclaration
		valid    bool
	}{
		{"unknown source only", unknown, PagesDeclaration{Source: &PagesSource{Branch: "main", Path: "/docs"}}, false},
		{"explicit legacy resolves", unknown, PagesDeclaration{BuildType: pagesPtr("legacy"), Source: &PagesSource{Branch: "main", Path: "/docs"}}, true},
		{"unrelated HTTPS", unknown, PagesDeclaration{HTTPSEnforced: pagesPtr(false)}, true},
		{"mode only legacy", unknown, PagesDeclaration{BuildType: pagesPtr("legacy")}, true},
		{"mode only workflow", unknown, PagesDeclaration{BuildType: pagesPtr("workflow")}, true},
		{"known legacy source", canonical, PagesDeclaration{Source: &PagesSource{Branch: "main", Path: "/docs"}}, true},
		{"known workflow source only", workflow, PagesDeclaration{Source: &PagesSource{Branch: "main", Path: "/docs"}}, false},
		{"known workflow complete transition", workflow, PagesDeclaration{BuildType: pagesPtr("legacy"), Source: &PagesSource{Branch: "main", Path: "/docs"}}, true},
		{"known workflow missing source", workflow, PagesDeclaration{BuildType: pagesPtr("legacy")}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			after, err := PreparePages(test.observed, test.desired)
			if (err == nil) != test.valid {
				t.Fatal(after, err, test.valid)
			}
			if err != nil {
				if strings.Contains(err.Error(), "UNMANAGED_MODE_CANARY") || !reflect.DeepEqual(after, PagesState{}) {
					t.Fatal("failed prerequisite leaks or forecasts", after, err)
				}
				return
			}
			if test.desired.Source != nil && *after.Source != *test.desired.Source {
				t.Fatal("source intent lost", after)
			}
			if test.desired.HTTPSEnforced != nil && after.HTTPSEnforced {
				t.Fatal("false intent lost", after)
			}
		})
	}
}
