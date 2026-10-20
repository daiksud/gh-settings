package settings

import (
	"math"
	"testing"
)

func retInt(v int) *int { return &v }
func TestRetentionAcquisitionBoundsAndReadOnlyMaximum(t *testing.T) {
	if _, _, err := (RetentionObservation{}).Values(); err == nil {
		t.Fatal("zero observation became defaults")
	}
	for _, days := range []int{0, -1, 401} {
		if _, err := NewRetentionObservation(days, 400); err == nil {
			t.Fatal(days)
		}
	}
	if _, err := NewRetentionObservation(90, 0); err == nil {
		t.Fatal("invalid maximum accepted")
	}
	for _, pair := range [][2]int{{1, 90}, {90, 90}, {400, 400}, {90, 14}} {
		o, err := NewRetentionObservation(pair[0], pair[1])
		if err != nil {
			t.Fatal(pair, err)
		}
		d, err := o.Declaration()
		if (err == nil) != (pair[0] <= pair[1]) {
			t.Fatal(pair, d, err)
		}
		lower := 1
		managed, changes, err := CompareRetention(o, ActionsPolicyDeclaration{RetentionDays: &lower})
		wantChanges := 1
		if pair[0] == 1 {
			wantChanges = 0
		}
		if err != nil || managed != 1 || len(changes) != wantChanges {
			t.Fatal(pair, managed, changes, err)
		}
	}
	d := ActionsPolicyDeclaration{RetentionDays: retInt(14)}
	if d.Empty() || !d.ExecutionEmpty() || !d.WorkflowPermissionsEmpty() {
		t.Fatal("retention coupled to unrelated policy", d)
	}
	o, _ := NewRetentionObservation(90, 90)
	if _, _, err := CompareRetention(o, ActionsPolicyDeclaration{RetentionDays: retInt(100)}); err == nil {
		t.Fatal("upper cap ignored")
	}
	if _, _, err := CompareRetention(o, ActionsPolicyDeclaration{RetentionDays: retInt(90)}); err != nil {
		t.Fatal(err)
	}
}
func TestRetentionNativeReportNumbersAreClosed(t *testing.T) {
	for _, value := range []any{14, float64(14)} {
		c, err := NewActionsPolicyChange("actions.artifact_and_log_retention_days", 90, value)
		if err != nil {
			t.Fatal(value, err)
		}
		before, after := c.Values()
		if before != 90 || after != 14 {
			t.Fatal(before, after)
		}
	}
	for _, value := range []any{nil, "14", true, 0, -1, 401, 1.5, math.Inf(1), math.NaN()} {
		if _, err := NewActionsPolicyChange("actions.artifact_and_log_retention_days", 90, value); err == nil {
			t.Fatal("invalid integer accepted", value)
		}
	}
	if _, err := NewActionsPolicyChange("actions.artifact_and_log_retention_days", nil, 14); err == nil {
		t.Fatal("unobservable retention accepted")
	}
	if _, err := NewActionsPolicyChange("actions.maximum_allowed_days", 90, 14); err == nil {
		t.Fatal("read-only field accepted")
	}
	if _, err := NewActionsPolicyChange("actions.artifact_and_log_retention_days", float64(90), 90); err == nil {
		t.Fatal("normalized no-op accepted")
	}
}
