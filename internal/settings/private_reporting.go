package settings

import "fmt"

// PrivateReportingObservation is acquired only after a public target and a
// complete enabled boolean have been observed by the adapter.
type PrivateReportingObservation struct{ acquired, enabled bool }

func NewPrivateReportingObservation(enabled bool) PrivateReportingObservation {
	return PrivateReportingObservation{acquired: true, enabled: enabled}
}
func (o PrivateReportingObservation) Value() (bool, error) {
	if !o.acquired {
		return false, fmt.Errorf("private vulnerability reporting observation was not acquired")
	}
	return o.enabled, nil
}
func ComparePrivateReporting(o PrivateReportingObservation, d SecurityDeclaration) (int, []SecurityChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	enabled, err := o.Value()
	if err != nil {
		return 0, nil, err
	}
	changes := []SecurityChange{}
	if d.PrivateVulnerabilityReporting == nil {
		return 0, changes, nil
	}
	if enabled != *d.PrivateVulnerabilityReporting {
		change, _ := NewSecurityChange("security.private_vulnerability_reporting", enabled, *d.PrivateVulnerabilityReporting)
		changes = append(changes, change)
	}
	return 1, changes, nil
}
