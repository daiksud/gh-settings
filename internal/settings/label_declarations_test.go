package settings

import (
	"reflect"
	"testing"
)

func TestLabelDeclarationsOwnValues(t *testing.T) {
	color, description := "ABCDEF", ""
	input := []LabelDeclaration{{Name: "bug", Color: &color, Description: &description}, {Name: "help wanted"}}
	declarations, err := NewLabelDeclarations(input)
	if err != nil {
		t.Fatal(err)
	}
	want := []LabelDeclaration{{Name: "bug", Color: labelText("abcdef"), Description: labelText("")}, {Name: "help wanted"}}
	if color != "ABCDEF" {
		t.Fatal("construction mutated input color")
	}
	color, description, input[0].Name = "bad", "changed", ""
	input[1] = LabelDeclaration{Name: "BUG"}
	values := declarations.Values()
	if declarations.Len() != 2 || !reflect.DeepEqual(values, want) {
		t.Fatalf("values=%+v, want %+v", values, want)
	}
	*values[0].Color, *values[0].Description, values[0].Name = "bad", "changed", ""
	values[1] = LabelDeclaration{}
	comparison, err := CompareLabels(labelObservation(t), &declarations)
	if err != nil || len(comparison.Labels) != 2 {
		t.Fatalf("comparison=%+v error=%v", comparison, err)
	}
	*comparison.Labels[0].After.Color = "bad"
	*comparison.Labels[0].After.Description = "changed"
	comparison.Labels[0].After.Name = ""
	if !reflect.DeepEqual(declarations.Values(), want) {
		t.Fatal("retrieved values or comparison mutated declarations")
	}
}

func TestLabelDeclarationsEmpty(t *testing.T) {
	for _, input := range [][]LabelDeclaration{nil, {}} {
		declarations, err := NewLabelDeclarations(input)
		if err != nil || declarations.Len() != 0 || declarations.Values() == nil {
			t.Fatalf("declarations=%+v error=%v", declarations, err)
		}
		comparison, err := CompareLabels(labelObservation(t), &declarations)
		if err != nil || comparison.Managed != 1 || len(comparison.Labels) != 0 {
			t.Fatalf("comparison=%+v error=%v", comparison, err)
		}
	}
}
