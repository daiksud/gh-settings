package settings

import "fmt"

// AIScanRepositoriesObservation is acquired only after a public target and a
// complete enabled boolean have been observed by the adapter.
type AIScanRepositoriesObservation struct{ acquired, enabled bool }

func NewAIScanRepositoriesObservation(enabled bool) AIScanRepositoriesObservation {
	return AIScanRepositoriesObservation{acquired: true, enabled: enabled}
}
func (o AIScanRepositoriesObservation) Value() (bool, error) {
	if !o.acquired {
		return false, fmt.Errorf("AI Scan for repositories observation was not acquired")
	}
	return o.enabled, nil
}
func CompareAIScanRepositories(o AIScanRepositoriesObservation, d SecurityDeclaration) (int, []SecurityChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	enabled, err := o.Value()
	if err != nil {
		return 0, nil, err
	}
	changes := []SecurityChange{}
	if d.AIScanRepositories == nil {
		return 0, changes, nil
	}
	if enabled != *d.AIScanRepositories {
		change, _ := NewSecurityChange("security.ai_scan_repositories", enabled, *d.AIScanRepositories)
		changes = append(changes, change)
	}
	return 1, changes, nil
}
