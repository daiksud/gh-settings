package settings

import "fmt"

// AIScanPullRequestsObservation is acquired only after a public target and a
// complete enabled boolean have been observed by the adapter.
type AIScanPullRequestsObservation struct{ acquired, enabled bool }

func NewAIScanPullRequestsObservation(enabled bool) AIScanPullRequestsObservation {
	return AIScanPullRequestsObservation{acquired: true, enabled: enabled}
}
func (o AIScanPullRequestsObservation) Value() (bool, error) {
	if !o.acquired {
		return false, fmt.Errorf("AI Scan for pull requests observation was not acquired")
	}
	return o.enabled, nil
}
func CompareAIScanPullRequests(o AIScanPullRequestsObservation, d SecurityDeclaration) (int, []SecurityChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	enabled, err := o.Value()
	if err != nil {
		return 0, nil, err
	}
	changes := []SecurityChange{}
	if d.AIScanPullRequests == nil {
		return 0, changes, nil
	}
	if enabled != *d.AIScanPullRequests {
		change, _ := NewSecurityChange("security.ai_scan_pull_requests", enabled, *d.AIScanPullRequests)
		changes = append(changes, change)
	}
	return 1, changes, nil
}
