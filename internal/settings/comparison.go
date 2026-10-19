package settings

// Comparison counts all declared fields and records only changed values.
type Comparison struct {
	Managed int
	Changes []Change
	Labels  []LabelChange
}

// Change contains a resource-qualified field path and native string, boolean or string-list values.
type Change struct {
	Field         string
	Before, After any
}

// CompareRepository compares the REST-observed base fields without rendering them.
func CompareRepository(current RepositoryState, desired RepositoryDeclaration) Comparison {
	var result Comparison
	stringField := func(name, current string, desired *string) {
		if desired == nil {
			return
		}
		result.Managed++
		if current != *desired {
			result.Changes = append(result.Changes, Change{Field: name, Before: current, After: *desired})
		}
	}
	stringField("repository.description", current.Description, desired.Description)
	stringField("repository.homepage", current.Homepage, desired.Homepage)
	compareRepositoryBoolFields(&result, repositoryBoolFields[:], current, desired)
	return result
}

func compareRepositoryBoolFields[S any](result *Comparison, fields []repositoryBoolField[S], current S, desired RepositoryDeclaration) {
	for _, field := range fields {
		after := field.desired(desired)
		if after == nil {
			continue
		}
		result.Managed++
		before := field.current(current)
		if before != *after {
			result.Changes = append(result.Changes, Change{Field: field.name, Before: before, After: *after})
		}
	}
}
