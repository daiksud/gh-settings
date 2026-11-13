package settings

import "testing"

func effortString(value string) *string { return &value }

func TestCopilotReviewEffortObservation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configured *string
		effective  string
		available  []string
		fail       bool
	}{
		{"default", nil, "balanced", []string{"lite", "balanced", "max"}, false},
		{"lite", effortString("lite"), "lite", []string{"lite", "balanced"}, false},
		{"explicit balanced", effortString("balanced"), "balanced", []string{"balanced"}, false},
		{"max configured", effortString("max"), "max", []string{"lite", "balanced", "max"}, false},
		{"max effective", nil, "max", []string{"lite", "balanced", "max"}, false},
		{"unknown effective", nil, "future", []string{"future"}, true},
		{"unavailable max configured", effortString("max"), "balanced", []string{"balanced"}, true},
		{"unknown configured", effortString("future"), "balanced", []string{"balanced", "future"}, true},
		{"inherit is API write only", effortString("inherit"), "balanced", []string{"balanced"}, true},
		{"default is UI only", effortString("default"), "balanced", []string{"balanced"}, true},
		{"missing effective", nil, "", []string{"balanced"}, true},
		{"missing availability", nil, "balanced", nil, true},
		{"unavailable configured", effortString("lite"), "balanced", []string{"balanced"}, true},
		{"unavailable effective", nil, "lite", []string{"balanced"}, true},
		{"empty available entry", nil, "balanced", []string{"balanced", ""}, true},
		{"duplicate available entry", nil, "balanced", []string{"balanced", "balanced"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o, err := NewCopilotReviewEffortObservation(tc.configured, tc.effective, tc.available)
			if (err != nil) != tc.fail {
				t.Fatalf("observation=%+v err=%v", o, err)
			}
			if !tc.fail {
				value, err := o.Value()
				want := "default"
				if tc.configured != nil {
					want = *tc.configured
				}
				if err != nil || value != want {
					t.Fatal(value, err)
				}
			}
		})
	}
}

func TestCopilotReviewEffortComparison(t *testing.T) {
	for _, tc := range []struct {
		name             string
		configured       *string
		desired          *string
		available        []string
		managed, changes int
		fail             bool
	}{
		{"omitted unread", nil, nil, nil, 0, 0, false},
		{"unread default", nil, effortString("default"), nil, 0, 0, true},
		{"observable default", nil, effortString("default"), []string{"balanced"}, 1, 0, false},
		{"default differs from explicit balanced", nil, effortString("balanced"), []string{"balanced"}, 1, 1, false},
		{"explicit balanced differs from default", effortString("balanced"), effortString("default"), []string{"balanced"}, 1, 1, false},
		{"matching explicit balanced", effortString("balanced"), effortString("balanced"), []string{"balanced"}, 1, 0, false},
		{"unavailable lite", nil, effortString("lite"), []string{"balanced"}, 0, 0, true},
		{"max desired", nil, effortString("max"), []string{"balanced", "max"}, 1, 1, false},
		{"matching max", effortString("max"), effortString("max"), []string{"balanced", "max"}, 1, 0, false},
		{"unavailable max", nil, effortString("max"), []string{"balanced"}, 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var o CopilotReviewEffortObservation
			if tc.available != nil {
				var err error
				o, err = NewCopilotReviewEffortObservation(tc.configured, "balanced", tc.available)
				if err != nil {
					t.Fatal(err)
				}
			}
			managed, changes, err := CompareCopilotReviewEffort(o, tc.desired)
			if (err != nil) != tc.fail || managed != tc.managed || len(changes) != tc.changes {
				t.Fatal(managed, changes, err)
			}
			if len(changes) > 0 {
				before, after := changes[0].Values()
				want, _ := o.Value()
				if before != want || after != *tc.desired || changes[0].Field() != CopilotReviewEffortPath || !(Comparison{CopilotReviewEffort: changes}).HasChanges() {
					t.Fatal(changes)
				}
			}
		})
	}
}

func TestCopilotReviewEffortOwnedCopiesAndAcquisition(t *testing.T) {
	configured := "lite"
	available := []string{"lite", "balanced", "max"}
	o, err := NewCopilotReviewEffortObservation(&configured, "lite", available)
	if err != nil {
		t.Fatal(err)
	}
	configured = "max"
	available[0] = "max"
	got, _ := o.Configured()
	*got = "max"
	choices, _ := o.Available()
	choices[0] = "max"
	got, _ = o.Configured()
	choices, _ = o.Available()
	effective, _ := o.Effective()
	if *got != "lite" || choices[0] != "lite" || effective != "lite" {
		t.Fatal(got, choices, effective)
	}
	var unread CopilotReviewEffortObservation
	if _, err := unread.Configured(); err == nil {
		t.Fatal("unread configured")
	}
	if _, err := unread.Effective(); err == nil {
		t.Fatal("unread effective")
	}
	if _, err := unread.Available(); err == nil {
		t.Fatal("unread available")
	}
	if (Declaration{CopilotReviewEffort: effortString("default")}).Empty() {
		t.Fatal("default declaration is managed")
	}
	for _, pair := range [][2]string{{"future", "lite"}, {"lite", "inherit"}, {"default", "default"}} {
		if _, err := NewCopilotReviewEffortChange(pair[0], pair[1]); err == nil {
			t.Fatal(pair)
		}
	}
}
