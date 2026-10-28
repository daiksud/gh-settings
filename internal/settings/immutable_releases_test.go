package settings

import "testing"

func TestImmutableReleasesObservationAndOwnerEnforcement(t *testing.T) {
	if _, _, _, err := (ImmutableReleasesObservation{}).Values(); err == nil {
		t.Fatal("unacquired observation accepted")
	}
	enabled, enforced, known, err := NewDisabledImmutableReleasesObservation().Values()
	if err != nil || enabled || enforced || known {
		t.Fatal("404 invented owner metadata", enabled, enforced, known, err)
	}
	for _, current := range []bool{false, true} {
		for _, desired := range []bool{false, true} {
			d := SecurityDeclaration{ImmutableReleases: &desired}
			managed, changes, err := CompareImmutableReleases(NewImmutableReleasesObservation(current, false), d)
			if err != nil || managed != 1 || (len(changes) == 1) != (current != desired) {
				t.Fatal(current, desired, managed, changes, err)
			}
		}
	}
	no, yes := false, true
	if _, _, err := CompareImmutableReleases(NewImmutableReleasesObservation(true, true), SecurityDeclaration{ImmutableReleases: &no}); err == nil {
		t.Fatal("owner enforcement ignored")
	}
	if _, changes, err := CompareImmutableReleases(NewImmutableReleasesObservation(true, true), SecurityDeclaration{ImmutableReleases: &yes}); err != nil || len(changes) != 0 {
		t.Fatal(changes, err)
	}
	if _, _, err := CompareImmutableReleases(ImmutableReleasesObservation{}, SecurityDeclaration{}); err != nil {
		t.Fatal("omission required an observation", err)
	}
}
