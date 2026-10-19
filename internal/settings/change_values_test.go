package settings

import (
	"slices"
	"testing"
)

func TestTypedScalarChanges(t *testing.T) {
	boolean, err := NewBoolChange(FieldID{path: "repository.has_issues"}, true, false)
	if err != nil {
		t.Fatal(err)
	}
	text, err := NewStringChange(FieldID{path: "repository.description"}, "before", "")
	if err != nil {
		t.Fatal(err)
	}
	var boolCalls, stringCalls int
	boolean.Visit(func(before, after bool) {
		boolCalls++
		if !before || after {
			t.Fatalf("boolean pair=%t, %t", before, after)
		}
	}, nil, nil)
	text.Visit(nil, func(before, after string) {
		stringCalls++
		if before != "before" || after != "" {
			t.Fatalf("string pair=%q, %q", before, after)
		}
	}, nil)
	if boolCalls != 1 || stringCalls != 1 {
		t.Fatalf("bool callbacks=%d string callbacks=%d", boolCalls, stringCalls)
	}
}

func TestChangeValuesMustMatchField(t *testing.T) {
	for _, path := range []string{"", "repository.unknown", "repository.description", "topics.names"} {
		if change, err := NewBoolChange(FieldID{path: path}, true, false); err == nil || change != (Change{}) {
			t.Errorf("accepted boolean pair for %q: %+v, %v", path, change, err)
		}
	}
	for _, path := range []string{"repository.has_issues", "topics.names"} {
		if change, err := NewStringChange(FieldID{path: path}, "before", "after"); err == nil || change != (Change{}) {
			t.Errorf("accepted string pair for %q: %+v, %v", path, change, err)
		}
	}
	field := FieldID{path: "repository.merge_commit_title"}
	for _, pair := range [][2]string{{"UNKNOWN", "PR_TITLE"}, {"PR_TITLE", "UNKNOWN"}} {
		if change, err := NewStringChange(field, pair[0], pair[1]); err == nil || change != (Change{}) {
			t.Errorf("accepted invalid enum pair: %+v, %v", change, err)
		}
	}
	if _, err := NewStringChange(field, "MERGE_MESSAGE", "PR_TITLE"); err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2][]string{{{"Upper"}, {"valid"}}, {{"valid"}, {"Upper"}}} {
		if change, err := NewTopicsChange(pair[0], pair[1]); err == nil || change != (Change{}) {
			t.Errorf("accepted invalid topics pair: %+v, %v", change, err)
		}
	}
}

func TestTopicsComparisonValuesAreIndependent(t *testing.T) {
	for _, test := range []struct {
		name      string
		construct func([]string, []string) Change
	}{
		{"constructor", func(before, after []string) Change {
			change, err := NewTopicsChange(before, after)
			if err != nil {
				t.Fatal(err)
			}
			return change
		}},
		{"comparison", func(before, after []string) Change {
			return CompareTopics(before, TopicsDeclaration{Names: &after}).Changes[0]
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			before, after := []string{"z", "a"}, []string{"y", "b"}
			change := test.construct(before, after)
			before[0], after[0] = "changed", "changed"
			for range 2 {
				change.Visit(nil, nil, func(before, after []string) {
					if !slices.Equal(before, []string{"a", "z"}) || !slices.Equal(after, []string{"b", "y"}) {
						t.Fatalf("stored topics changed: before=%v after=%v", before, after)
					}
					before[0], after[0] = "changed", "changed"
				})
			}
		})
	}
}

func TestEmptyTopicsComparisonValues(t *testing.T) {
	change, err := NewTopicsChange(nil, []string{})
	if err != nil {
		t.Fatal(err)
	}
	change.Visit(nil, nil, func(before, after []string) {
		if before == nil || after == nil || len(before) != 0 || len(after) != 0 {
			t.Fatalf("empty values lost array identity: before=%#v after=%#v", before, after)
		}
	})
}
