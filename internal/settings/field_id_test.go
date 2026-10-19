package settings

import "testing"

func TestTopicsFieldIdentity(t *testing.T) {
	field, err := ParseFieldID("topics.names")
	if err != nil || field != TopicsField() || field.Path() != "topics.names" {
		t.Fatalf("topics identity=%+v: %v", field, err)
	}
}

func TestFieldIdentityRejectsUnknownPaths(t *testing.T) {
	for _, path := range []string{"", "description", "repository.unknown", "settings.repository.description", "repository.Description", "topics", "topics.unknown", "labels"} {
		if field, err := ParseFieldID(path); err == nil || field != (FieldID{}) {
			t.Errorf("accepted %q as %+v: %v", path, field, err)
		}
	}
}

func TestChangeRequiresSupportedFieldIdentity(t *testing.T) {
	field, err := ParseFieldID("repository.description")
	if err != nil {
		t.Fatal(err)
	}
	change, err := NewChange(field, "before", "")
	if err != nil || change.Field() != field || change.Before != "before" || change.After != "" {
		t.Fatalf("change=%+v: %v", change, err)
	}
	for _, invalid := range []FieldID{{}, {path: "repository.unknown"}} {
		if change, err := NewChange(invalid, "before", "after"); err == nil || change != (Change{}) {
			t.Errorf("constructed change=%+v from invalid field=%+v: %v", change, invalid, err)
		}
	}
}
