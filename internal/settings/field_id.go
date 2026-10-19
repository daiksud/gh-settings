package settings

import "fmt"

// FieldID identifies a supported repository field or the topics collection.
// Its zero value is invalid; resolve external paths with ParseFieldID.
type FieldID struct{ path string }

var topicsField = FieldID{path: "topics.names"}

// TopicsField identifies the managed topic names collection.
func TopicsField() FieldID { return topicsField }

// ParseFieldID resolves an exact report path through the supported definitions.
func ParseFieldID(path string) (FieldID, error) {
	if path == topicsField.path {
		return topicsField, nil
	}
	if _, ok := RepositoryBoolSetter(path); ok {
		return FieldID{path: path}, nil
	}
	if _, _, ok := RepositoryStringSetter(path); ok {
		return FieldID{path: path}, nil
	}
	return FieldID{}, fmt.Errorf("unknown comparison field %q", path)
}

// Path returns the existing resource-qualified report and display path.
func (field FieldID) Path() string { return field.path }
