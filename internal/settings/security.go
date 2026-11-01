package settings

import "fmt"

const SecurityFieldCount = 6

// SecurityDeclaration owns only the adopted repository security controls.
type SecurityDeclaration struct {
	DependabotAlerts              *bool `yaml:"dependabot_alerts,omitempty"`
	DependabotSecurityUpdates     *bool `yaml:"dependabot_security_updates,omitempty"`
	ImmutableReleases             *bool `yaml:"immutable_releases,omitempty"`
	SecretScanning                *bool `yaml:"secret_scanning,omitempty"`
	PushProtection                *bool `yaml:"secret_scanning_push_protection,omitempty"`
	PrivateVulnerabilityReporting *bool `yaml:"private_vulnerability_reporting,omitempty"`
}

func (d SecurityDeclaration) SecretScanningEmpty() bool {
	return d.SecretScanning == nil && d.PushProtection == nil
}
func (d SecurityDeclaration) Empty() bool {
	return d.DependabotEmpty() && d.SecretScanningEmpty() && d.PrivateVulnerabilityReporting == nil && d.ImmutableReleases == nil
}
func (d SecurityDeclaration) Validate() error {
	if d.DependabotAlerts != nil && !*d.DependabotAlerts && d.DependabotSecurityUpdates != nil && *d.DependabotSecurityUpdates {
		return fmt.Errorf("repository.settings.advanced_security.dependabot.security_updates requires alerts")
	}
	if d.SecretScanning != nil && !*d.SecretScanning && d.PushProtection != nil && *d.PushProtection {
		return fmt.Errorf("%s requires %s", PushProtectionControl.Path(), SecretProtectionControl.Path())
	}
	return nil
}

// SecretScanningObservation cannot turn an unacquired response into disabled.
type SecretScanningObservation struct {
	acquired       bool
	scanning       bool
	push           bool
	normalizations [2]SecurityStatusNormalization
}

func NewSecretScanningObservation(scanning, push bool) SecretScanningObservation {
	return SecretScanningObservation{acquired: true, scanning: scanning, push: push}
}
func (o SecretScanningObservation) Values() (bool, bool, error) {
	if !o.acquired {
		return false, false, fmt.Errorf("Secret scanning observation was not acquired")
	}
	return o.scanning, o.push, nil
}
func (o SecretScanningObservation) Declaration() (SecurityDeclaration, error) {
	s, p, err := o.Values()
	if err != nil {
		return SecurityDeclaration{}, err
	}
	d := SecurityDeclaration{SecretScanning: &s, PushProtection: &p}
	if err := d.Validate(); err != nil {
		return SecurityDeclaration{}, fmt.Errorf("observed security controls cannot be exported as a consistent declaration: %w", err)
	}
	return d, nil
}

// SecurityChange retains a closed field identity and two native, known booleans.
type SecurityChange struct {
	field         string
	before, after bool
	normalization SecurityStatusNormalization
}

func (c SecurityChange) Field() string                              { return c.field }
func (c SecurityChange) Values() (bool, bool)                       { return c.before, c.after }
func (c SecurityChange) Normalization() SecurityStatusNormalization { return c.normalization }
func NewSecurityChange(field string, before, after any) (SecurityChange, error) {
	b, bok := before.(bool)
	a, aok := after.(bool)
	if (field != DependabotAlertsField && field != DependabotUpdatesField && field != "security.secret_scanning" && field != "security.secret_scanning_push_protection" && field != "security.private_vulnerability_reporting" && field != "security.immutable_releases") || !bok || !aok || b == a {
		return SecurityChange{}, fmt.Errorf("invalid security comparison")
	}
	return SecurityChange{field: field, before: b, after: a}, nil
}
func ValidateSecurityChanges(changes []SecurityChange) error {
	seen := map[string]bool{}
	var d SecurityDeclaration
	for _, c := range changes {
		var err error
		if c.normalization != (SecurityStatusNormalization{}) {
			_, err = NewSecurityChangeWithNormalization(c.field, c.before, c.after, c.normalization)
		} else {
			_, err = NewSecurityChange(c.field, c.before, c.after)
		}
		if err != nil || seen[c.field] {
			return fmt.Errorf("invalid or duplicate security comparison")
		}
		seen[c.field] = true
		v := c.after
		switch c.field {
		case DependabotAlertsField:
			d.DependabotAlerts = &v
		case DependabotUpdatesField:
			d.DependabotSecurityUpdates = &v
		case "security.secret_scanning":
			d.SecretScanning = &v
		case "security.secret_scanning_push_protection":
			d.PushProtection = &v
		case "security.immutable_releases":
			d.ImmutableReleases = &v
		case "security.private_vulnerability_reporting":
			d.PrivateVulnerabilityReporting = &v
		default:
			return fmt.Errorf("unknown security comparison")
		}
	}
	return d.Validate()
}
func CompareSecretScanning(o SecretScanningObservation, d SecurityDeclaration) (int, []SecurityChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	s, p, err := o.Values()
	if err != nil {
		return 0, nil, err
	}
	managed := 0
	changes := []SecurityChange{}
	for _, f := range []struct {
		key           string
		before        bool
		desired       *bool
		normalization SecurityStatusNormalization
	}{{"security.secret_scanning", s, d.SecretScanning, o.normalizations[0]}, {"security.secret_scanning_push_protection", p, d.PushProtection, o.normalizations[1]}} {
		if f.desired == nil {
			continue
		}
		managed++
		if f.normalization != (SecurityStatusNormalization{}) {
			change, err := NewSecurityChangeWithNormalization(f.key, f.before, *f.desired, f.normalization)
			if err != nil {
				return 0, nil, err
			}
			changes = append(changes, change)
		} else if f.before != *f.desired {
			c, _ := NewSecurityChange(f.key, f.before, *f.desired)
			changes = append(changes, c)
		}
	}
	return managed, changes, nil
}

// SecretScanningTransitions preflights the complete family before its first write.
func SecretScanningTransitions(o SecretScanningObservation, d SecurityDeclaration) ([]SecurityChange, error) {
	_, changes, err := CompareSecretScanning(o, d)
	if err != nil {
		return nil, err
	}
	s, p, _ := o.RawStatuses()
	if d.PushProtection != nil && *d.PushProtection && d.SecretScanning == nil && s != "enabled" {
		return nil, fmt.Errorf("%s requires %s to be enabled or an explicit true declaration", PushProtectionControl.Path(), SecretProtectionControl.Path())
	}
	if d.SecretScanning != nil && !*d.SecretScanning && p != "disabled" && (d.PushProtection == nil || *d.PushProtection) {
		if _, supported := SecurityValueForStatus(p); !supported {
			return nil, fmt.Errorf("disabling secret_protection requires explicitly disabling unverified push protection")
		}
		return nil, fmt.Errorf("disabling secret_protection requires explicitly disabling active push protection")
	}
	if d.SecretScanning != nil && !*d.SecretScanning && len(changes) == 2 {
		changes[0], changes[1] = changes[1], changes[0]
	}
	return changes, nil
}

// AfterSecretScanningChange checks fresh prerequisites for a single narrow write.
func AfterSecretScanningChange(o SecretScanningObservation, c SecurityChange) (SecretScanningObservation, error) {
	if err := ValidateSecurityChanges([]SecurityChange{c}); err != nil {
		return SecretScanningObservation{}, err
	}
	s, p, err := o.RawStatuses()
	if err != nil {
		return SecretScanningObservation{}, err
	}
	if c.field == "security.secret_scanning" {
		if !c.after && p != "disabled" {
			if _, supported := SecurityValueForStatus(p); !supported {
				return SecretScanningObservation{}, fmt.Errorf("unverified push protection prevents disabling secret_protection")
			}
			return SecretScanningObservation{}, fmt.Errorf("active push protection prevents disabling secret_protection")
		}
		o.scanning = c.after
		o.normalizations[0] = SecurityStatusNormalization{}
	} else if c.field == "security.secret_scanning_push_protection" {
		if c.after && s != "enabled" {
			return SecretScanningObservation{}, fmt.Errorf("secret_protection must be enabled before push protection")
		}
		o.push = c.after
		o.normalizations[1] = SecurityStatusNormalization{}
	} else {
		return SecretScanningObservation{}, fmt.Errorf("change does not belong to Secret scanning")
	}
	return o, nil
}
