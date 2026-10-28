package settings

import "fmt"

// ImmutableReleasesObservation keeps owner enforcement separate from the toggle.
// A disabled-state 404 acquires enabled=false, but no owner-enforcement value.
type ImmutableReleasesObservation struct {
	acquired, enabled, enforcedByOwner, ownerKnown bool
}

func NewImmutableReleasesObservation(enabled, enforcedByOwner bool) ImmutableReleasesObservation {
	return ImmutableReleasesObservation{acquired: true, enabled: enabled, enforcedByOwner: enforcedByOwner, ownerKnown: true}
}

func NewDisabledImmutableReleasesObservation() ImmutableReleasesObservation {
	return ImmutableReleasesObservation{acquired: true}
}

func (o ImmutableReleasesObservation) Values() (enabled, enforcedByOwner, ownerKnown bool, err error) {
	if !o.acquired {
		return false, false, false, fmt.Errorf("immutable releases observation was not acquired")
	}
	if o.enforcedByOwner && !o.enabled {
		return false, false, false, fmt.Errorf("owner-enforced immutable releases cannot be observed as disabled")
	}
	return o.enabled, o.enforcedByOwner, o.ownerKnown, nil
}

func CompareImmutableReleases(o ImmutableReleasesObservation, d SecurityDeclaration) (int, []SecurityChange, error) {
	if d.ImmutableReleases == nil {
		return 0, nil, nil
	}
	enabled, enforced, _, err := o.Values()
	if err != nil {
		return 0, nil, err
	}
	if !*d.ImmutableReleases && enforced {
		return 0, nil, fmt.Errorf("immutable releases are enforced by the repository owner; disabling is not allowed")
	}
	if enabled == *d.ImmutableReleases {
		return 1, nil, nil
	}
	change, err := NewSecurityChange("security.immutable_releases", enabled, *d.ImmutableReleases)
	return 1, []SecurityChange{change}, err
}
