package settings

import "testing"

func securityBool(v bool) *bool { return &v }
func TestSecretScanningObservationAndTransitions(t *testing.T) {
	if _, _, err := (SecretScanningObservation{}).Values(); err == nil {
		t.Fatal("unacquired state became false")
	}
	for _, current := range []struct{ s, p bool }{{false, false}, {true, false}, {true, true}, {false, true}} {
		o := NewSecretScanningObservation(current.s, current.p)
		for _, d := range []SecurityDeclaration{{}, {SecretScanning: securityBool(true)}, {SecretScanning: securityBool(false)}, {PushProtection: securityBool(true)}, {PushProtection: securityBool(false)}, {SecretScanning: securityBool(true), PushProtection: securityBool(true)}, {SecretScanning: securityBool(false), PushProtection: securityBool(false)}, {SecretScanning: securityBool(false), PushProtection: securityBool(true)}} {
			wantErr := d.Validate() != nil || d.PushProtection != nil && *d.PushProtection && d.SecretScanning == nil && !current.s || d.SecretScanning != nil && !*d.SecretScanning && current.p && d.PushProtection == nil
			changes, err := SecretScanningTransitions(o, d)
			if (err != nil) != wantErr {
				t.Fatalf("current=%+v desired=%+v changes=%v err=%v", current, d, changes, err)
			}
			if err != nil {
				continue
			}
			state := o
			for _, change := range changes {
				state, err = AfterSecretScanningChange(state, change)
				if err != nil {
					t.Fatal(current, d, change, err)
				}
			}
			s, p, _ := state.Values()
			if d.SecretScanning != nil && s != *d.SecretScanning || d.PushProtection != nil && p != *d.PushProtection {
				t.Fatal("desired values not reached", s, p, d)
			}
			if d.SecretScanning == nil && s != current.s || d.PushProtection == nil && p != current.p {
				t.Fatal("omitted member changed")
			}
			_, again, err := CompareSecretScanning(state, d)
			if err != nil || len(again) != 0 {
				t.Fatal("not idempotent", again, err)
			}
		}
	}
	original := NewSecretScanningObservation(true, false)
	d, err := original.Declaration()
	if err != nil {
		t.Fatal(err)
	}
	*d.SecretScanning = false
	s, _, _ := original.Values()
	if !s {
		t.Fatal("observation alias")
	}
}
func TestSecurityComparisonsRejectForgedValues(t *testing.T) {
	for _, field := range []string{"security.secret_scanning", "security.secret_scanning_push_protection"} {
		for _, wrong := range []any{nil, "true", 0, []bool{true}} {
			if _, err := NewSecurityChange(field, false, wrong); err == nil {
				t.Fatal(field, wrong)
			}
		}
	}
	if _, err := NewSecurityChange("security.code_security", false, true); err == nil {
		t.Fatal("candidate accepted")
	}
	c, _ := NewSecurityChange("security.secret_scanning", true, false)
	p, _ := NewSecurityChange("security.secret_scanning_push_protection", false, true)
	if ValidateSecurityChanges([]SecurityChange{c, c}) == nil || ValidateSecurityChanges([]SecurityChange{c, p}) == nil {
		t.Fatal("forged or contradictory report accepted")
	}
}
