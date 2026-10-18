package settings

// RepositoryState contains the five complete values from the repository REST read.
type RepositoryState struct {
	Description string
	Homepage    string
	HasIssues   bool
	HasProjects bool
	HasWiki     bool
}

// RepositoryDeclaration leaves nil fields unmanaged; false and empty strings are declared values.
type RepositoryDeclaration struct {
	Description      *string
	Homepage         *string
	HasIssues        *bool
	HasProjects      *bool
	HasWiki          *bool
	AllowMergeCommit *bool
	AllowSquashMerge *bool
	AllowRebaseMerge *bool
}

// Empty reports whether every field is unmanaged, regardless of declared values.
func (d RepositoryDeclaration) Empty() bool {
	return d == (RepositoryDeclaration{})
}

// BaseEmpty reports whether the REST-observed repository fields are unmanaged.
func (d RepositoryDeclaration) BaseEmpty() bool {
	return d.Description == nil && d.Homepage == nil && d.HasIssues == nil && d.HasProjects == nil && d.HasWiki == nil
}

// MergeMethodsEmpty keeps the GraphQL read optional for existing declarations.
func (d RepositoryDeclaration) MergeMethodsEmpty() bool {
	return d.AllowMergeCommit == nil && d.AllowSquashMerge == nil && d.AllowRebaseMerge == nil
}

// MergeMethodsState is complete only after all three GraphQL values are validated.
type MergeMethodsState struct{ AllowMergeCommit, AllowSquashMerge, AllowRebaseMerge bool }

// CompareMergeMethods uses only declared fields, in the repository's stable order.
func CompareMergeMethods(current MergeMethodsState, desired RepositoryDeclaration) Comparison {
	var result Comparison
	for _, field := range []struct {
		name   string
		before bool
		after  *bool
	}{
		{"repository.allow_merge_commit", current.AllowMergeCommit, desired.AllowMergeCommit},
		{"repository.allow_squash_merge", current.AllowSquashMerge, desired.AllowSquashMerge},
		{"repository.allow_rebase_merge", current.AllowRebaseMerge, desired.AllowRebaseMerge},
	} {
		if field.after == nil {
			continue
		}
		result.Managed++
		if field.before != *field.after {
			result.Changes = append(result.Changes, Change{Field: field.name, Before: field.before, After: *field.after})
		}
	}
	return result
}
