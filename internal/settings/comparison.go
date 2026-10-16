package settings

// RepositoryComparison counts all declared fields and records only changed values.
type RepositoryComparison struct {
	Managed int
	Changes []RepositoryChange
}

// RepositoryChange contains a field name and a string/string or bool/bool value pair.
type RepositoryChange struct {
	Field         string
	Before, After any
}

// CompareRepository compares declared fields in a stable order without rendering them.
func CompareRepository(current RepositoryState, desired RepositoryDeclaration) RepositoryComparison {
	var result RepositoryComparison
	stringField := func(name, current string, desired *string) {
		if desired == nil {
			return
		}
		result.Managed++
		if current != *desired {
			result.Changes = append(result.Changes, RepositoryChange{Field: name, Before: current, After: *desired})
		}
	}
	boolField := func(name string, current bool, desired *bool) {
		if desired == nil {
			return
		}
		result.Managed++
		if current != *desired {
			result.Changes = append(result.Changes, RepositoryChange{Field: name, Before: current, After: *desired})
		}
	}
	stringField("description", current.Description, desired.Description)
	stringField("homepage", current.Homepage, desired.Homepage)
	boolField("has_issues", current.HasIssues, desired.HasIssues)
	boolField("has_projects", current.HasProjects, desired.HasProjects)
	boolField("has_wiki", current.HasWiki, desired.HasWiki)
	return result
}
