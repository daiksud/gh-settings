package settings

import (
	"reflect"
	"strings"
	"testing"
)

func labelText(value string) *string { return &value }
func labelList(t *testing.T, values ...LabelDeclaration) *LabelDeclarations {
	t.Helper()
	result, err := NewLabelDeclarations(values)
	if err != nil {
		t.Fatal(err)
	}
	return &result
}

func TestCompareLabelsAuthoritativeMembership(t *testing.T) {
	current := []LabelState{{Name: "old", Color: "FFFFFF", Description: "assigned"}, {Name: "Bug", Color: "abcdef", Description: "keep"}}
	desired := labelList(t, LabelDeclaration{Name: "new", Description: labelText("")}, LabelDeclaration{Name: "bug", Color: labelText("ABCDEF")})
	got, err := CompareLabels(labelObservation(t, current...), desired)
	want := Comparison{Managed: 1, Labels: []LabelChange{
		{Operation: "create", Name: "new", After: &LabelDeclaration{Name: "new", Description: labelText("")}},
		{Operation: "delete", Name: "old", Before: &LabelState{Name: "old", Color: "ffffff", Description: "assigned"}},
	}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("comparison=%+v error=%v want=%+v", got, err, want)
	}
	if err := ValidateLabelChanges(got.Labels); err != nil {
		t.Fatal(err)
	}
	if current[0].Color != "FFFFFF" || *desired.Values()[1].Color != "abcdef" {
		t.Fatal("comparison mutated input")
	}
}

func TestCompareLabelsPresenceAndPartialAttributes(t *testing.T) {
	current := []LabelState{{Name: "bug", Color: "ffffff", Description: "keep"}}
	for _, test := range []struct {
		name      string
		desired   *LabelDeclarations
		managed   int
		operation string
	}{
		{"omitted", nil, 0, ""},
		{"empty deletes all", labelList(t), 1, "delete"},
		{"name only preserves attributes", labelList(t, LabelDeclaration{Name: "BUG"}), 1, ""},
		{"color only", labelList(t, LabelDeclaration{Name: "bug", Color: labelText("000000")}), 1, "update"},
		{"empty description clears", labelList(t, LabelDeclaration{Name: "bug", Description: labelText("")}), 1, "update"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := CompareLabels(labelObservation(t, current...), test.desired)
			if err != nil {
				t.Fatal(err)
			}
			if got.Managed != test.managed {
				t.Fatalf("comparison=%+v", got)
			}
			if test.operation == "" {
				if len(got.Labels) != 0 {
					t.Fatalf("unexpected changes=%+v", got)
				}
			} else if len(got.Labels) != 1 || got.Labels[0].Operation != test.operation {
				t.Fatalf("changes=%+v", got.Labels)
			}
			if (Declaration{Labels: test.desired}).Empty() != (test.desired == nil) {
				t.Fatal("omission lost")
			}
			if err := ValidateLabelChanges(got.Labels); err != nil {
				t.Fatal(err)
			}
		})
	}
	empty, err := CompareLabels(labelObservation(t), labelList(t))
	if err != nil || empty.Managed != 1 || len(empty.Labels) != 0 {
		t.Fatalf("empty=%+v error=%v", empty, err)
	}
}

func TestCompareLabelsDeterministicOrderAndNoRenameInference(t *testing.T) {
	current := []LabelState{{Name: "z", Color: "ffffff"}, {Name: "old", Color: "000000"}}
	desired := labelList(t, LabelDeclaration{Name: "new"}, LabelDeclaration{Name: "a"})
	got, err := CompareLabels(labelObservation(t, current...), desired)
	if err != nil {
		t.Fatal(err)
	}
	var operations []string
	for _, change := range got.Labels {
		operations = append(operations, change.Operation+":"+change.Name)
	}
	if !reflect.DeepEqual(operations, []string{"create:a", "create:new", "delete:old", "delete:z"}) {
		t.Fatalf("operations=%v", operations)
	}
	reordered, err := CompareLabels(labelObservation(t, current[1], current[0]), labelList(t, desired.Values()[1], desired.Values()[0]))
	if err != nil || !reflect.DeepEqual(reordered, got) {
		t.Fatalf("unstable comparison=%+v err=%v", reordered, err)
	}
}

func TestLabelsValidationAndAmbiguity(t *testing.T) {
	for _, labels := range [][]LabelDeclaration{
		{{Name: ""}}, {{Name: "\xff"}}, {{Name: " "}}, {{Name: "a\nb"}}, {{Name: "bug"}, {Name: "BUG"}}, {{Name: "Σ"}, {Name: "ς"}},
		{{Name: "bug", Color: labelText("#ffffff")}}, {{Name: "bug", Color: labelText("fff")}}, {{Name: "bug", Color: labelText("gggggg")}},
		{{Name: "bug", Description: labelText(strings.Repeat("界", 101))}}, {{Name: "bug", Description: labelText("\xff")}},
	} {
		if _, err := NewLabelDeclarations(labels); err == nil {
			t.Fatalf("accepted %+v", labels)
		}
	}
	if _, err := NewLabelDeclarations([]LabelDeclaration{{Name: "area/docs 🐛", Color: labelText("ABC123"), Description: labelText(strings.Repeat("界", 100))}}); err != nil {
		t.Fatal(err)
	}
}

func TestLabelOperationValidation(t *testing.T) {
	before := &LabelState{Name: "bug", Color: "ffffff", Description: "old"}
	for _, change := range []LabelChange{
		{Operation: "rename", Name: "bug", Before: before},
		{Operation: "delete", Name: "bug"},
		{Operation: "create", Name: "bug", Before: before, After: &LabelDeclaration{Name: "bug"}},
		{Operation: "update", Name: "bug", Before: before, After: &LabelDeclaration{Name: "bug"}},
		{Operation: "update", Name: "bug", Before: before, After: &LabelDeclaration{Name: "bug", Color: labelText("ffffff")}},
		{Operation: "create", Name: "bug", After: &LabelDeclaration{Name: "other"}},
	} {
		if err := ValidateLabelChanges([]LabelChange{change}); err == nil {
			t.Fatalf("accepted %+v", change)
		}
	}
}

func labelObservation(t *testing.T, values ...LabelState) LabelObservation {
	t.Helper()
	result, err := NewLabelObservation(values)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
