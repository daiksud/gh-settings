package settings

import (
	"reflect"
	"strings"
	"testing"
)

func TestLabelObservationOwnsCompleteValues(t *testing.T) {
	input := []LabelState{{Name: "Bug", Color: "ABCDEF", Description: ""}}
	observed, err := NewLabelObservation(input)
	if err != nil {
		t.Fatal(err)
	}
	if input[0].Color != "ABCDEF" {
		t.Fatal("construction mutated input")
	}
	input[0] = LabelState{}
	values, err := observed.Values()
	want := []LabelState{{Name: "Bug", Color: "abcdef", Description: ""}}
	if err != nil || !reflect.DeepEqual(values, want) {
		t.Fatalf("values=%+v error=%v", values, err)
	}
	values[0] = LabelState{}
	values, err = observed.Values()
	if err != nil || !reflect.DeepEqual(values, want) {
		t.Fatalf("retrieved values mutated observation: %+v error=%v", values, err)
	}
}

func TestLabelObservationRejectsInvalidValues(t *testing.T) {
	for _, input := range [][]LabelState{
		{{Name: "", Color: "abcdef"}}, {{Name: " ", Color: "abcdef"}}, {{Name: "a\nb", Color: "abcdef"}}, {{Name: "\xff", Color: "abcdef"}},
		{{Name: "bug", Color: "#abcdef"}}, {{Name: "bug", Color: "gggggg"}}, {{Name: "bug", Color: "fff"}},
		{{Name: "bug", Color: "abcdef", Description: strings.Repeat("界", 101)}}, {{Name: "bug", Color: "abcdef", Description: "\xff"}},
		{{Name: "bug", Color: "abcdef"}, {Name: "BUG", Color: "123456"}}, {{Name: "Σ", Color: "abcdef"}, {Name: "ς", Color: "123456"}},
	} {
		observed, err := NewLabelObservation(input)
		values, readErr := observed.Values()
		if err == nil || readErr == nil || values != nil {
			t.Fatalf("input=%+v observation=%+v error=%v read error=%v", input, observed, err, readErr)
		}
	}
}

func TestLabelObservationDistinguishesMissingAndCompleteEmpty(t *testing.T) {
	var missing LabelObservation
	if values, err := missing.Values(); err == nil || values != nil {
		t.Fatalf("missing observation became values=%+v error=%v", values, err)
	}
	if comparison, err := CompareLabels(missing, labelList(t)); err == nil || !reflect.DeepEqual(comparison, Comparison{}) {
		t.Fatalf("missing observation became comparison=%+v error=%v", comparison, err)
	}
	for _, input := range [][]LabelState{nil, {}} {
		observed, err := NewLabelObservation(input)
		if err != nil {
			t.Fatal(err)
		}
		values, err := observed.Values()
		if err != nil || values == nil || len(values) != 0 {
			t.Fatalf("complete empty observation=%+v error=%v", values, err)
		}
		comparison, err := CompareLabels(observed, labelList(t))
		if err != nil || comparison.Managed != 1 || len(comparison.Labels) != 0 {
			t.Fatalf("comparison=%+v error=%v", comparison, err)
		}
	}
}
