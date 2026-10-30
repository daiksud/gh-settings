package settings

import "testing"

func TestForkApprovalObservationOwnsRawNormalization(t *testing.T) {
	for _, raw := range []string{"FUTURE", "", " \t\n "} {
		record := ObservationNormalization{Path: "repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors", RawValue: raw, Fallback: "require_approval_for_first_time_contributors"}
		observed, err := NewForkApprovalObservationWithNormalization("first_time_contributors", record)
		if err != nil {
			t.Fatal("valid normalization rejected", raw, err)
		}
		if value, err := observed.Value(); err != nil || value != "first_time_contributors" {
			t.Fatal("effective value was not acquired", value, err)
		}
		if value, err := observed.RawValue(); err != nil || value != raw {
			t.Fatal("raw value was not acquired", value, err)
		}
		if observed.Normalization() != record {
			t.Fatal("original evidence lost", observed)
		}
		record.RawValue = "caller mutation"
		returned := observed.Normalization()
		returned.RawValue = "returned mutation"
		if observed.Normalization().RawValue != raw {
			t.Fatal("observation borrowed mutable metadata", observed)
		}
	}
	for _, value := range []string{"first_time_contributors_new_to_github", "first_time_contributors", "all_external_contributors"} {
		observed, err := NewForkApprovalObservation(value)
		if err != nil || observed.Normalization() != (ObservationNormalization{}) {
			t.Fatal("canonical constructor contract changed", observed, err)
		}
		if raw, err := observed.RawValue(); err != nil || raw != value {
			t.Fatal("canonical raw value changed", raw, err)
		}
	}
	if _, err := NewForkApprovalObservation("FUTURE"); err == nil {
		t.Fatal("strict constructor silently normalized")
	}
	var empty ForkApprovalObservation
	if _, err := empty.Value(); err == nil {
		t.Fatal("unacquired effective value invented")
	}
	if _, err := empty.RawValue(); err == nil {
		t.Fatal("unacquired raw value invented")
	}
	if empty.Normalization() != (ObservationNormalization{}) {
		t.Fatal("unacquired default evidence invented")
	}
}

func TestForkApprovalObservationRejectsForgedNormalization(t *testing.T) {
	for _, test := range []struct {
		value  string
		record ObservationNormalization
	}{
		{"first_time_contributors", ObservationNormalization{}},
		{"first_time_contributors", ObservationNormalization{Path: "wrong", RawValue: "FUTURE", Fallback: "require_approval_for_first_time_contributors"}},
		{"first_time_contributors", ObservationNormalization{Path: ForkApprovalPath, RawValue: "FUTURE", Fallback: "require_approval_for_all_external_contributors"}},
		{"first_time_contributors", ObservationNormalization{Path: ForkApprovalPath, RawValue: "first_time_contributors_new_to_github", Fallback: "require_approval_for_first_time_contributors"}},
		{"first_time_contributors", ObservationNormalization{Path: ForkApprovalPath, RawValue: "first_time_contributors", Fallback: "require_approval_for_first_time_contributors"}},
		{"first_time_contributors", ObservationNormalization{Path: ForkApprovalPath, RawValue: "all_external_contributors", Fallback: "require_approval_for_first_time_contributors"}},
		{"all_external_contributors", ObservationNormalization{Path: ForkApprovalPath, RawValue: "FUTURE", Fallback: "require_approval_for_first_time_contributors"}},
		{"FUTURE", ObservationNormalization{Path: ForkApprovalPath, RawValue: "FUTURE", Fallback: "require_approval_for_first_time_contributors"}},
	} {
		observed, err := NewForkApprovalObservationWithNormalization(test.value, test.record)
		if err == nil || observed.Normalization() != (ObservationNormalization{}) {
			t.Fatal("forged evidence acquired policy", test, observed, err)
		}
		if _, err := observed.Value(); err == nil {
			t.Fatal("failed constructor leaked effective acquisition")
		}
		if _, err := observed.RawValue(); err == nil {
			t.Fatal("failed constructor leaked raw acquisition")
		}
	}
}
