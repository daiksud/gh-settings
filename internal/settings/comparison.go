package settings

// Comparison counts all declared fields and records only changed values.
type Comparison struct {
	Managed int
	Changes []Change
	Labels  []LabelChange
}

// Change contains a supported field identifier and native string, boolean or string-list values.
type Change struct {
	field         FieldID
	Before, After any
}

// NewChange constructs a comparison entry only for a supported field identifier.
func NewChange(field FieldID, before, after any) (Change, error) {
	if _, err := ParseFieldID(field.path); err != nil {
		return Change{}, err
	}
	return Change{field: field, Before: before, After: after}, nil
}

// Field returns the validated field identity without exposing a mutable path.
func (change Change) Field() FieldID { return change.field }

// CompareRepository compares the REST-observed base fields without rendering them.
func CompareRepository(current RepositoryState, desired RepositoryDeclaration) Comparison {
	var result Comparison
	compareRepositoryFields(&result, repositoryStringFields[:], current, desired)
	compareRepositoryFields(&result, repositoryBoolFields[:], current, desired)
	return result
}

func compareRepositoryFields[S any, V bool | string](result *Comparison, fields []repositoryField[S, V], current S, desired RepositoryDeclaration) {
	for _, field := range fields {
		after := field.desired(desired)
		if after == nil {
			continue
		}
		result.Managed++
		before := field.current(current)
		if before != *after {
			result.Changes = append(result.Changes, Change{field: FieldID{path: field.name}, Before: before, After: *after})
		}
	}
}
