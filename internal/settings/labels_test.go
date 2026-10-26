package settings

import (
	"strings"
	"testing"
)

func labelText(value string) *string { return &value }
func labelID(value int64) *int64     { return &value }
func labelObservation(t *testing.T, values ...LabelState) LabelObservation {
	t.Helper()
	result, err := NewLabelObservation(values)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func labelList(t *testing.T, values ...LabelDeclaration) *LabelDeclarations {
	t.Helper()
	result, err := NewLabelDeclarations(values)
	if err != nil {
		t.Fatal(err)
	}
	return &result
}

func TestCompareLabelsSelectsIDsAndPreservesUnlistedObjects(t *testing.T) {
	current := labelObservation(t, LabelState{ID: 7, Name: "existing", Color: "abcdef", Description: "old"}, LabelState{ID: 8, Name: "unmanaged", Color: "123456"})
	for _, desired := range []*LabelDeclarations{nil, labelList(t)} {
		got, err := CompareLabels(LabelObservation{}, desired)
		if err != nil || got.Managed != 0 || got.Labels.Len() != 0 {
			t.Fatal(got, err)
		}
	}
	desired := labelList(t, LabelDeclaration{ID: labelID(7), Name: "renamed", Color: labelText("fedcba")}, LabelDeclaration{Name: "new", Color: labelText("123456")})
	got, err := CompareLabels(current, desired)
	if err != nil || got.Labels.Len() != 2 {
		t.Fatal(got, err)
	}
	for _, change := range got.Labels.Values() {
		if change.Operation() == "delete" {
			t.Fatal("unlisted object was deleted")
		}
		change.Visit(func(after LabelDeclaration) {
			if after.ID != nil || after.Name != "new" || after.Description == nil || *after.Description != "" {
				t.Fatal(after)
			}
		}, func(before LabelState, after LabelDeclaration) {
			if before.ID != 7 || after.ID == nil || *after.ID != 7 || after.Name != "renamed" || after.Description == nil || *after.Description != "" {
				t.Fatal(before, after)
			}
		}, nil)
	}
}

func TestCompareLabelsRejectsCollisionAndMissingIDWithoutFallback(t *testing.T) {
	current := labelObservation(t, LabelState{ID: 7, Name: "Bug", Color: "abcdef"}, LabelState{ID: 8, Name: "other", Color: "123456"})
	for _, tc := range []struct {
		desired LabelDeclaration
		want    string
	}{
		{LabelDeclaration{Name: "BUG", Color: labelText("abcdef")}, "specify id: 7"},
		{LabelDeclaration{ID: labelID(99), Name: "new", Color: labelText("abcdef")}, "refusing create fallback"},
		{LabelDeclaration{ID: labelID(8), Name: "bug", Color: labelText("abcdef")}, "rename collision"},
	} {
		got, err := CompareLabels(current, labelList(t, tc.desired))
		if err == nil || !strings.Contains(err.Error(), tc.want) || got.Labels.Len() != 0 {
			t.Fatal(got, err)
		}
	}
}

func TestCompareLabelsDefaultsAndCaseIdentity(t *testing.T) {
	current := labelObservation(t, LabelState{ID: 7, Name: "Bug", Color: "abcdef", Description: "clear this"})
	got, err := CompareLabels(current, labelList(t, LabelDeclaration{ID: labelID(7), Name: "BUG", Color: labelText("ABCDEF")}))
	if err != nil || got.Labels.Len() != 1 {
		t.Fatal(got, err)
	}
	got.Labels.Values()[0].Visit(nil, func(before LabelState, after LabelDeclaration) {
		if after.Name != "Bug" || *after.Color != "abcdef" || *after.Description != "" {
			t.Fatal(after)
		}
	}, nil)
	current = labelObservation(t, LabelState{ID: 7, Name: "Bug", Color: "abcdef", Description: ""})
	got, err = CompareLabels(current, labelList(t, LabelDeclaration{ID: labelID(7), Name: "BUG", Color: labelText("ABCDEF")}))
	if err != nil || got.Labels.Len() != 0 {
		t.Fatal(got, err)
	}
}

func TestLabelsValidateWholeObjectAndIdentities(t *testing.T) {
	for _, values := range [][]LabelDeclaration{
		{{Name: "x"}},
		{{ID: labelID(0), Name: "x", Color: labelText("abcdef")}},
		{{ID: labelID(7), Name: "x", Color: labelText("abcdef")}, {ID: labelID(7), Name: "y", Color: labelText("abcdef")}},
		{{Name: "Bug", Color: labelText("abcdef")}, {Name: "BUG", Color: labelText("abcdef")}},
		{{Name: "Σ", Color: labelText("abcdef")}, {Name: "ς", Color: labelText("abcdef")}},
		{{Name: "x", Color: labelText("bad")}},
		{{Name: "x", Color: labelText("abcdef"), Description: labelText(strings.Repeat("界", 101))}},
	} {
		if _, err := NewLabelDeclarations(values); err == nil {
			t.Fatalf("invalid declaration accepted: %+v", values)
		}
	}
	if _, err := NewLabelDeclarations([]LabelDeclaration{{Name: "area/docs 🐛", Color: labelText("ABCDEF")}}); err != nil {
		t.Fatal(err)
	}
}
