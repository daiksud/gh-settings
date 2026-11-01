package settings

import "fmt"

const (
	DependabotAlertsField  = "security.dependabot_alerts"
	DependabotUpdatesField = "security.dependabot_security_updates"
	DependabotAlertsPath   = "repository.settings.advanced_security.dependabot.alerts"
	DependabotUpdatesPath  = "repository.settings.advanced_security.dependabot.security_updates"
)

func (d SecurityDeclaration) DependabotEmpty() bool {
	return d.DependabotAlerts == nil && d.DependabotSecurityUpdates == nil
}

type DependabotObservation struct {
	Alerts  DependabotAlertsObservation
	Updates DependabotSecurityUpdatesObservation
}

// DependabotWriteResult distinguishes an accepted 204+404 disable from an
// independently verified write. Acceptance is not proof of retained state.
type DependabotWriteResult struct {
	Wrote           bool
	AcceptedDisable bool
}

func (o DependabotObservation) Values() (alerts, updates bool, err error) {
	alerts, err = o.Alerts.Value()
	if err != nil {
		return false, false, err
	}
	updates, _, _, err = o.Updates.Values()
	if err != nil {
		return false, false, err
	}
	return alerts, updates, nil
}

func (o DependabotObservation) Declaration() (SecurityDeclaration, error) {
	a, u, err := o.Values()
	if err != nil {
		return SecurityDeclaration{}, err
	}
	d := SecurityDeclaration{DependabotAlerts: &a, DependabotSecurityUpdates: &u}
	if err := d.Validate(); err != nil {
		return SecurityDeclaration{}, err
	}
	return d, nil
}

func DependabotTransitions(o DependabotObservation, d SecurityDeclaration) (int, []SecurityChange, error) {
	if d.DependabotEmpty() {
		return 0, nil, nil
	}
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	a, u, err := o.Values()
	if err != nil {
		return 0, nil, err
	}
	if d.DependabotSecurityUpdates != nil && *d.DependabotSecurityUpdates && !a && (d.DependabotAlerts == nil || !*d.DependabotAlerts) {
		return 0, nil, fmt.Errorf("Dependabot security updates require enabled alerts; omitted alerts will not be enabled")
	}
	if d.DependabotAlerts != nil && !*d.DependabotAlerts && u && (d.DependabotSecurityUpdates == nil || *d.DependabotSecurityUpdates) {
		return 0, nil, fmt.Errorf("disabling Dependabot alerts requires explicitly disabling active security updates")
	}
	managed := 0
	var changes []SecurityChange
	for _, field := range []struct {
		name    string
		before  bool
		desired *bool
	}{
		{DependabotAlertsField, a, d.DependabotAlerts}, {DependabotUpdatesField, u, d.DependabotSecurityUpdates},
	} {
		if field.desired == nil {
			continue
		}
		managed++
		if field.before != *field.desired {
			c, err := NewSecurityChange(field.name, field.before, *field.desired)
			if err != nil {
				return 0, nil, err
			}
			changes = append(changes, c)
		}
	}
	if d.DependabotAlerts != nil && !*d.DependabotAlerts && len(changes) == 2 {
		changes[0], changes[1] = changes[1], changes[0]
	}
	return managed, changes, nil
}

// DependabotAlertsObservation retains the explicit 404-as-disabled policy
// separately from positive endpoint evidence. The zero value is unacquired.
type DependabotAlertsObservation struct {
	acquired, enabled, disabledBy404 bool
}

func NewEnabledDependabotAlertsObservation() DependabotAlertsObservation {
	return DependabotAlertsObservation{acquired: true, enabled: true}
}

func NewDependabotAlerts404Observation() DependabotAlertsObservation {
	return DependabotAlertsObservation{acquired: true, disabledBy404: true}
}

func (o DependabotAlertsObservation) Value() (bool, error) {
	if !o.acquired {
		return false, fmt.Errorf("Dependabot alerts observation was not acquired")
	}
	return o.enabled, nil
}

func (o DependabotAlertsObservation) DisabledBy404() bool { return o.disabledBy404 }

// VerifyEnabled does not promote the 404 reading policy into proof that a
// mutation succeeded. Alerts have no authoritative disabled response here.
func (o DependabotAlertsObservation) VerifyEnabled(desired bool) error {
	enabled, err := o.Value()
	if err != nil {
		return err
	}
	if o.disabledBy404 {
		return fmt.Errorf("Dependabot alerts 404 is interpreted as disabled but cannot verify an update; permission may be insufficient")
	}
	if enabled != desired {
		return fmt.Errorf("Dependabot alerts enabled state does not match requested value")
	}
	return nil
}

// DependabotSecurityUpdatesObservation keeps enabled intent distinct from
// operational pause metadata. A 404 acquires false under the product policy,
// but acquires no pause value and cannot establish a successful mutation.
type DependabotSecurityUpdatesObservation struct {
	acquired, enabled, paused, pauseKnown, disabledBy404 bool
}

func NewDependabotSecurityUpdatesObservation(enabled, paused bool) DependabotSecurityUpdatesObservation {
	return DependabotSecurityUpdatesObservation{acquired: true, enabled: enabled, paused: paused, pauseKnown: true}
}

func NewDependabotSecurityUpdates404Observation() DependabotSecurityUpdatesObservation {
	return DependabotSecurityUpdatesObservation{acquired: true, disabledBy404: true}
}

func (o DependabotSecurityUpdatesObservation) Values() (enabled, paused, pauseKnown bool, err error) {
	if !o.acquired {
		return false, false, false, fmt.Errorf("Dependabot security updates observation was not acquired")
	}
	return o.enabled, o.paused, o.pauseKnown, nil
}

func (o DependabotSecurityUpdatesObservation) DisabledBy404() bool { return o.disabledBy404 }

// VerifyEnabled compares only enabled; pause is observed metadata, not a
// request to resume updates. Even disabled intent cannot be verified by 404.
func (o DependabotSecurityUpdatesObservation) VerifyEnabled(desired bool) error {
	enabled, _, _, err := o.Values()
	if err != nil {
		return err
	}
	if o.disabledBy404 {
		return fmt.Errorf("Dependabot security updates 404 is interpreted as disabled but cannot verify an update; permission may be insufficient")
	}
	if enabled != desired {
		return fmt.Errorf("Dependabot security updates enabled state does not match requested value")
	}
	return nil
}
