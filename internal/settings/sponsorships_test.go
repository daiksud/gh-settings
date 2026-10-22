package settings

import (
	"reflect"
	"testing"
)

func TestSponsorshipsOmissionAndAcquiredFalse(t *testing.T) {
	no := false
	desired := RepositoryDeclaration{HasSponsorships: &no}
	if desired.Empty() || desired.RepositoryAccessEmpty() || !desired.BaseEmpty() || !desired.PRAccessEmpty() {
		t.Fatal("incorrect resource ownership", desired)
	}
	missing := NewRepositoryObservation(nil, nil, nil)
	observed := missing.WithSponsorships(SponsorshipsState{Enabled: true})
	if len(missing.Fields()) != 0 || len(observed.Fields()) != 1 {
		t.Fatal("observation mutated or companions invented")
	}
	got, err := CompareRepositoryObservation(observed, desired)
	want := Comparison{Managed: 1, Changes: []Change{{field: FieldID{path: "repository.has_sponsorships"}, values: boolPair{before: true, after: false}}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal(got, err)
	}
	if _, err := CompareRepositoryObservation(missing, desired); err == nil {
		t.Fatal("missing observation became false")
	}
	got, err = CompareRepositoryObservation(missing.WithSponsorships(SponsorshipsState{Enabled: false}), desired)
	if err != nil || got.Managed != 1 || len(got.Changes) != 0 {
		t.Fatal("observed false not preserved", got, err)
	}
	if _, err := observed.Declaration(); err == nil {
		t.Fatal("partial observation exported")
	}
	got, err = CompareRepositoryObservation(observed, RepositoryDeclaration{})
	if err != nil || got.Managed != 0 || len(got.Changes) != 0 {
		t.Fatal("omission became managed", got, err)
	}
}
