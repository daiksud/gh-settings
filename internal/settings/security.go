package settings

import "fmt"

const SecurityFieldCount = 2

// SecurityDeclaration owns only the two adopted Secret scanning controls.
type SecurityDeclaration struct {
	SecretScanning *bool `yaml:"secret_scanning,omitempty"`
	PushProtection *bool `yaml:"secret_scanning_push_protection,omitempty"`
}

func (d SecurityDeclaration) Empty() bool { return d.SecretScanning == nil && d.PushProtection == nil }
func (d SecurityDeclaration) Validate() error {
	if d.SecretScanning != nil && !*d.SecretScanning && d.PushProtection != nil && *d.PushProtection {
		return fmt.Errorf("security.secret_scanning_push_protection requires secret_scanning")
	}
	return nil
}

// SecretScanningObservation cannot turn an unacquired response into disabled.
type SecretScanningObservation struct {
	acquired bool
	scanning bool
	push     bool
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
}

func (c SecurityChange) Field() string        { return c.field }
func (c SecurityChange) Values() (bool, bool) { return c.before, c.after }
func NewSecurityChange(field string, before, after any) (SecurityChange, error) {
	b, bok := before.(bool)
	a, aok := after.(bool)
	if (field != "security.secret_scanning" && field != "security.secret_scanning_push_protection") || !bok || !aok || b == a {
		return SecurityChange{}, fmt.Errorf("invalid security comparison")
	}
	return SecurityChange{field: field, before: b, after: a}, nil
}
func ValidateSecurityChanges(changes []SecurityChange) error {
	seen := map[string]bool{}
	var d SecurityDeclaration
	for _, c := range changes {
		if _, err := NewSecurityChange(c.field, c.before, c.after); err != nil || seen[c.field] {
			return fmt.Errorf("invalid or duplicate security comparison")
		}
		seen[c.field] = true
		v := c.after
		if c.field == "security.secret_scanning" {
			d.SecretScanning = &v
		} else {
			d.PushProtection = &v
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
		key     string
		before  bool
		desired *bool
	}{{"security.secret_scanning", s, d.SecretScanning}, {"security.secret_scanning_push_protection", p, d.PushProtection}} {
		if f.desired == nil {
			continue
		}
		managed++
		if f.before != *f.desired {
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
	s, p, _ := o.Values()
	if d.PushProtection != nil && *d.PushProtection && d.SecretScanning == nil && !s {
		return nil, fmt.Errorf("push protection requires already enabled secret_scanning or an explicit true declaration")
	}
	if d.SecretScanning != nil && !*d.SecretScanning && p && (d.PushProtection == nil || *d.PushProtection) {
		return nil, fmt.Errorf("disabling secret_scanning requires explicitly disabling active push protection")
	}
	if d.SecretScanning != nil && !*d.SecretScanning && len(changes) == 2 {
		changes[0], changes[1] = changes[1], changes[0]
	}
	return changes, nil
}

// AfterSecretScanningChange checks fresh prerequisites for a single narrow write.
func AfterSecretScanningChange(o SecretScanningObservation, c SecurityChange) (SecretScanningObservation, error) {
	if _, err := NewSecurityChange(c.field, c.before, c.after); err != nil {
		return SecretScanningObservation{}, err
	}
	s, p, err := o.Values()
	if err != nil {
		return SecretScanningObservation{}, err
	}
	if c.field == "security.secret_scanning" {
		if !c.after && p {
			return SecretScanningObservation{}, fmt.Errorf("active push protection prevents disabling secret_scanning")
		}
		s = c.after
	} else {
		if c.after && !s {
			return SecretScanningObservation{}, fmt.Errorf("secret_scanning must be enabled before push protection")
		}
		p = c.after
	}
	return NewSecretScanningObservation(s, p), nil
}
