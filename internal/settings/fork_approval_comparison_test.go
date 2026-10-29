package settings

import "testing"

func TestForkApprovalNormalizedComparisonRetainsRequiredWrites(t *testing.T) {
	for _, raw := range []string{"FUTURE", ""} {
		record := ObservationNormalization{Path: ForkApprovalPath, RawValue: raw, Fallback: "require_approval_for_first_time_contributors"}
		observed, err := NewForkApprovalObservationWithNormalization("first_time_contributors", record)
		if err != nil {
			t.Fatal(err)
		}
		for _, desired := range []string{"first_time_contributors_new_to_github", "first_time_contributors", "all_external_contributors"} {
			managed, changes, err := CompareForkApproval(observed, ActionsPolicyDeclaration{ForkApprovalPolicy: &desired})
			if err != nil || managed != 1 || len(changes) != 1 || changes[0].Normalization() != record {
				t.Fatal("fallback equality lost actionable provenance", raw, desired, managed, changes, err)
			}
			before, after := changes[0].Values()
			if before != "first_time_contributors" || after != desired {
				t.Fatal("native values changed", before, after)
			}
		}
		if managed, changes, err := CompareForkApproval(observed, ActionsPolicyDeclaration{}); err != nil || managed != 0 || len(changes) != 0 {
			t.Fatal("omitted policy became default intent", managed, changes, err)
		}
	}
	for _, pair := range [][2]string{{"first_time_contributors", "first_time_contributors"}, {"first_time_contributors", "all_external_contributors"}} {
		observed, err := NewForkApprovalObservation(pair[0])
		if err != nil {
			t.Fatal(err)
		}
		managed, changes, err := CompareForkApproval(observed, ActionsPolicyDeclaration{ForkApprovalPolicy: &pair[1]})
		want := 0
		if pair[0] != pair[1] {
			want = 1
		}
		if err != nil || managed != 1 || len(changes) != want {
			t.Fatal("canonical comparison changed", managed, changes, err)
		}
		if want > 0 && changes[0].Normalization() != (ObservationNormalization{}) {
			t.Fatal("canonical comparison gained raw provenance")
		}
	}
}

func TestForkApprovalComparisonRejectsUnacquiredOrInconsistentEvidence(t *testing.T) {
	desired := "first_time_contributors"
	for _, observed := range []ForkApprovalObservation{
		{},
		{acquired: true, policy: "first_time_contributors", normalization: ObservationNormalization{Path: "wrong", RawValue: "FUTURE", Fallback: "require_approval_for_first_time_contributors"}},
		{acquired: true, policy: "all_external_contributors", normalization: ObservationNormalization{Path: ForkApprovalPath, RawValue: "FUTURE", Fallback: "require_approval_for_first_time_contributors"}},
	} {
		if managed, changes, err := CompareForkApproval(observed, ActionsPolicyDeclaration{ForkApprovalPolicy: &desired}); err == nil || managed != 0 || len(changes) != 0 {
			t.Fatal("invalid state returned comparison", managed, changes, err)
		}
	}
}
