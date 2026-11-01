package settings

import "testing"

func TestDependabotTransitionsAndOmission(t *testing.T) {
	ptr := func(v bool) *bool { return &v }
	for _, a := range []bool{false, true} {
		for _, u := range []bool{false, true} {
			ao := NewDependabotAlerts404Observation()
			if a {
				ao = NewEnabledDependabotAlertsObservation()
			}
			o := DependabotObservation{Alerts: ao, Updates: NewDependabotSecurityUpdatesObservation(u, true)}
			for _, da := range []*bool{nil, ptr(false), ptr(true)} {
				for _, du := range []*bool{nil, ptr(false), ptr(true)} {
					d := SecurityDeclaration{DependabotAlerts: da, DependabotSecurityUpdates: du}
					invalid := da != nil && !*da && du != nil && *du || du != nil && *du && !a && da == nil || da != nil && !*da && u && du == nil
					managed, changes, err := DependabotTransitions(o, d)
					if (err != nil) != invalid {
						t.Fatalf("a=%v u=%v da=%v du=%v: %v", a, u, da, du, err)
					}
					if invalid {
						if managed != 0 || len(changes) != 0 {
							t.Fatal("partial preflight escaped")
						}
						continue
					}
					wantManaged, wantChanges := 0, 0
					if da != nil {
						wantManaged++
						if *da != a {
							wantChanges++
						}
					}
					if du != nil {
						wantManaged++
						if *du != u {
							wantChanges++
						}
					}
					if managed != wantManaged || len(changes) != wantChanges {
						t.Fatal(managed, changes, wantManaged, wantChanges)
					}
					if len(changes) == 2 {
						wantFirst := DependabotAlertsField
						if !*da {
							wantFirst = DependabotUpdatesField
						}
						if changes[0].Field() != wantFirst {
							t.Fatal("wrong dependency order", changes)
						}
					}
				}
			}
		}
	}
	if _, _, err := DependabotTransitions(DependabotObservation{}, SecurityDeclaration{}); err != nil {
		t.Fatal("omission acquired state", err)
	}
	if _, _, err := DependabotTransitions(DependabotObservation{}, SecurityDeclaration{DependabotAlerts: ptr(false)}); err == nil {
		t.Fatal("unacquired family accepted")
	}
}

func TestDependabotObservationsRequireAcquisition(t *testing.T) {
	var alerts DependabotAlertsObservation
	if _, err := alerts.Value(); err == nil || alerts.VerifyEnabled(false) == nil {
		t.Fatal("unacquired alerts became disabled or verified")
	}
	var updates DependabotSecurityUpdatesObservation
	if _, _, _, err := updates.Values(); err == nil || updates.VerifyEnabled(false) == nil {
		t.Fatal("unacquired updates became disabled or verified")
	}
}

func TestDependabot404ReadsDoNotVerifyUpdates(t *testing.T) {
	alerts := NewDependabotAlerts404Observation()
	enabled, err := alerts.Value()
	if err != nil || enabled || !alerts.DisabledBy404() {
		t.Fatal(alerts, err)
	}
	updates := NewDependabotSecurityUpdates404Observation()
	enabled, paused, known, err := updates.Values()
	if err != nil || enabled || paused || known || !updates.DisabledBy404() {
		t.Fatal("404 must not invent pause metadata", updates, err)
	}
	for _, desired := range []bool{false, true} {
		if alerts.VerifyEnabled(desired) == nil || updates.VerifyEnabled(desired) == nil {
			t.Fatal("404 reading policy was used to verify an update", desired)
		}
	}
}

func TestDependabotDirectEvidenceAndIndependentPause(t *testing.T) {
	alerts := NewEnabledDependabotAlertsObservation()
	if enabled, err := alerts.Value(); err != nil || !enabled || alerts.DisabledBy404() || alerts.VerifyEnabled(true) != nil || alerts.VerifyEnabled(false) == nil {
		t.Fatal(alerts, err)
	}
	for _, enabled := range []bool{false, true} {
		for _, paused := range []bool{false, true} {
			updates := NewDependabotSecurityUpdatesObservation(enabled, paused)
			gotEnabled, gotPaused, known, err := updates.Values()
			if err != nil || gotEnabled != enabled || gotPaused != paused || !known || updates.DisabledBy404() {
				t.Fatal(updates, err)
			}
			if updates.VerifyEnabled(enabled) != nil || updates.VerifyEnabled(!enabled) == nil {
				t.Fatal("pause was confused with enabled intent", enabled, paused)
			}
		}
	}
}
