package settings

import "fmt"

// CodeQLDefaultSetupObservation is acquired only after a public target and a
// complete enabled boolean have been observed by the adapter.
type CodeQLDefaultSetupObservation struct{ acquired, enabled bool }

func NewCodeQLDefaultSetupObservation(enabled bool) CodeQLDefaultSetupObservation {
	return CodeQLDefaultSetupObservation{acquired: true, enabled: enabled}
}
func (o CodeQLDefaultSetupObservation) Value() (bool, error) {
	if !o.acquired {
		return false, fmt.Errorf("CodeQL default setup observation was not acquired")
	}
	return o.enabled, nil
}
func CompareCodeQLDefaultSetup(o CodeQLDefaultSetupObservation, d SecurityDeclaration) (int, []SecurityChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	enabled, err := o.Value()
	if err != nil {
		return 0, nil, err
	}
	changes := []SecurityChange{}
	if d.CodeQLDefaultSetup == nil {
		return 0, changes, nil
	}
	if enabled != *d.CodeQLDefaultSetup {
		change, _ := NewSecurityChange("security.codeql_default_setup", enabled, *d.CodeQLDefaultSetup)
		changes = append(changes, change)
	}
	return 1, changes, nil
}
