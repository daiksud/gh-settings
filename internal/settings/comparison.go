package settings

// Comparison counts all declared fields and records only changed values.
type Comparison struct {
	Managed int
	Changes []Change
}

// Change contains a resource-qualified field path and native string, boolean or string-list values.
type Change struct {
	Field         string
	Before, After any
}

// CompareRepository compares declared fields in a stable order without rendering them.
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
	boolField := func(name string, current bool, desired *bool) {
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
	boolField("repository.has_issues", current.HasIssues, desired.HasIssues)
	boolField("repository.has_projects", current.HasProjects, desired.HasProjects)
	boolField("repository.has_wiki", current.HasWiki, desired.HasWiki)
	return result
}
