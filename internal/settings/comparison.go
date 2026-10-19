package settings

// Comparison counts all declared fields and records only changed values.
type Comparison struct {
	Managed int
	Changes []Change
	Labels  LabelChanges
}

// Change contains a supported field identifier and a matching typed value pair.
type Change struct {
	field  FieldID
	values changeValues
}

// Field returns the validated field identity without exposing a mutable path.
func (change Change) Field() FieldID { return change.field }

// CompareRepository compares the REST-observed base fields without rendering them.
func CompareRepository(current RepositoryState, desired RepositoryDeclaration) Comparison {
	var result Comparison
	compareRepositoryFields(&result, repositoryStringFields[:], current, desired, stringChange)
	compareRepositoryFields(&result, repositoryBoolFields[:], current, desired, boolChange)
	return result
}

func compareRepositoryFields[S any, V bool | string](result *Comparison, fields []repositoryField[S, V], current S, desired RepositoryDeclaration, change func(FieldID, V, V) Change) {
	for _, field := range fields {
		after := field.desired(desired)
		if after == nil {
			continue
		}
		result.Managed++
		before := field.current(current)
		if before != *after {
			result.Changes = append(result.Changes, change(FieldID{path: field.name}, before, *after))
		}
	}
}
