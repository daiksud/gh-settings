package settings

import (
	"testing"
)

func TestPruneSelectionValidation(t *testing.T) {
	p, err := ParsePruneScopes("labels,rulesets")
	if err != nil || !p.Has("labels") || !p.Has("rulesets") || p.Has("environments") {
		t.Fatal(p, err)
	}
	for _, value := range []string{"", "labels,", ",labels", "labels,,rulesets", "all,labels", "labels,all", "all", "labels,labels", "Labels", " labels", "unknown", "actions-policies", "environments"} {
		if _, err := ParsePruneScopes(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
}
func TestLabelPrunePresenceIdentityAndOrder(t *testing.T) {
	current, _ := NewLabelObservation([]LabelState{{ID: 7, Name: "keep", Color: "abcdef"}, {ID: 8, Name: "dependencies", Color: "123456"}})
	empty, _ := NewLabelDeclarations(nil)
	for _, tc := range []struct {
		d    *LabelDeclarations
		p    bool
		want int
	}{{nil, true, 0}, {&empty, false, 0}, {&empty, true, 2}} {
		got, err := CompareLabelsWithPrune(current, tc.d, tc.p)
		if err != nil || got.Labels.Len() != tc.want {
			t.Fatal(got, err)
		}
	}
	if _, err := CompareLabelsWithPrune(LabelObservation{}, &empty, true); err == nil {
		t.Fatal("missing read accepted")
	}
	id := int64(7)
	color := "abcdef"
	desired, _ := NewLabelDeclarations([]LabelDeclaration{{ID: &id, Name: "renamed", Color: &color}, {Name: "new", Color: &color}})
	got, err := CompareLabelsWithPrune(current, &desired, true)
	if err != nil {
		t.Fatal(err)
	}
	changes := got.Labels.Values()
	if len(changes) != 3 || changes[0].Operation() != "create" || changes[1].Operation() != "update" || changes[2].Operation() != "delete" || changes[2].Name() != "dependencies" {
		t.Fatal(changes)
	}
	collision, _ := NewLabelDeclarations([]LabelDeclaration{{Name: "keep", Color: &color}})
	if _, err := CompareLabelsWithPrune(current, &collision, true); err == nil {
		t.Fatal("create-only collision erased existing label")
	}
}
