package settings

import (
	"reflect"
	"testing"
)

func TestLabelOperationConstructionRejectsInvalidValues(t *testing.T) {
	before := LabelState{Name: "bug", Color: "abcdef", Description: "old"}
	for _, after := range []LabelDeclaration{
		{Name: ""}, {Name: "bug", Color: labelText("ABCDEF")}, {Name: "bug", Color: labelText("bad")},
	} {
		if _, err := NewLabelCreate(after); err == nil {
			t.Fatalf("created invalid label %+v", after)
		}
	}
	for _, after := range []LabelDeclaration{
		{Name: "other", Description: labelText("")}, {Name: "bug"},
		{Name: "bug", Color: labelText("abcdef")}, {Name: "bug", Description: labelText("old")},
		{Name: "bug", Color: labelText("ABCDEF")}, {Name: "bug", Color: labelText("000000"), Description: labelText("old")},
	} {
		if _, err := NewLabelUpdate(before, after); err == nil {
			t.Fatalf("created invalid update %+v", after)
		}
	}
	for _, invalid := range []LabelState{{}, {Name: "bug", Color: "ABCDEF"}, {Name: "bug", Color: "bad"}} {
		if _, err := NewLabelDelete(invalid); err == nil {
			t.Fatalf("created invalid delete %+v", invalid)
		}
		if _, err := NewLabelUpdate(invalid, LabelDeclaration{Name: "bug", Description: labelText("")}); err == nil {
			t.Fatalf("created update with invalid before %+v", invalid)
		}
	}
	if _, err := NewLabelChanges([]LabelChange{{}}); err == nil {
		t.Fatal("accepted missing operation")
	}
	create, err := NewLabelCreate(LabelDeclaration{Name: "Σ"})
	if err != nil {
		t.Fatal(err)
	}
	remove, err := NewLabelDelete(LabelState{Name: "ς", Color: "abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewLabelChanges([]LabelChange{create, remove}); err == nil {
		t.Fatal("accepted duplicate identities")
	}
}

func TestLabelChangesEmptySequence(t *testing.T) {
	var empty LabelChanges
	if empty.Len() != 0 || empty.Values() == nil {
		t.Fatal("invalid empty sequence")
	}
	for _, input := range [][]LabelChange{nil, {}} {
		sequence, err := NewLabelChanges(input)
		if err != nil || sequence.Len() != 0 || sequence.Values() == nil {
			t.Fatalf("empty sequence=%+v error=%v", sequence, err)
		}
	}
}

func TestLabelOperationsOwnAttributesAndSequence(t *testing.T) {
	color, empty := "123456", ""
	input := LabelDeclaration{Name: "bug", Color: &color, Description: &empty}
	create, err := NewLabelCreate(input)
	if err != nil {
		t.Fatal(err)
	}
	update, err := NewLabelUpdate(LabelState{Name: "existing", Color: "abcdef", Description: "old"}, LabelDeclaration{Name: "existing", Description: &empty})
	if err != nil {
		t.Fatal(err)
	}
	remove, err := NewLabelDelete(LabelState{Name: "old", Color: "abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	input.Name, color, empty = "", "bad", "changed"
	operations := []LabelChange{create, update, remove}
	sequence, err := NewLabelChanges(operations)
	if err != nil {
		t.Fatal(err)
	}
	operations[0] = LabelChange{}
	values := sequence.Values()
	var names []string
	for _, change := range values {
		names = append(names, change.Operation()+":"+change.Name())
		change.Visit(func(after LabelDeclaration) {
			if after.Name != "bug" || *after.Color != "123456" || *after.Description != "" {
				t.Fatalf("create=%+v", after)
			}
			*after.Color, *after.Description = "bad", "changed"
		}, func(before LabelState, after LabelDeclaration) {
			if before.Name != "existing" || after.Color != nil || *after.Description != "" {
				t.Fatalf("update before=%+v after=%+v", before, after)
			}
			*after.Description = "changed"
		}, func(before LabelState) {
			if before.Name != "old" {
				t.Fatalf("delete=%+v", before)
			}
		})
	}
	values[0] = LabelChange{}
	if sequence.Len() != 3 || !reflect.DeepEqual(names, []string{"create:bug", "update:existing", "delete:old"}) {
		t.Fatalf("names=%v", names)
	}
	sequence.Values()[0].Visit(func(after LabelDeclaration) {
		if *after.Color != "123456" || *after.Description != "" {
			t.Fatal("visitor mutated create")
		}
	}, nil, nil)
	sequence.Values()[1].Visit(nil, func(_ LabelState, after LabelDeclaration) {
		if *after.Description != "" {
			t.Fatal("visitor mutated update")
		}
	}, nil)
}
