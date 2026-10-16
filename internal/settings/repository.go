package settings

// RepositoryState contains all managed values from a successful repository read.
type RepositoryState struct {
	Description string
	Homepage    string
	HasIssues   bool
	HasProjects bool
	HasWiki     bool
}

// RepositoryDeclaration leaves nil fields unmanaged; false and empty strings are declared values.
type RepositoryDeclaration struct {
	Description *string
	Homepage    *string
	HasIssues   *bool
	HasProjects *bool
	HasWiki     *bool
}

// Empty reports whether every field is unmanaged, regardless of declared values.
func (d RepositoryDeclaration) Empty() bool {
	return d == (RepositoryDeclaration{})
}
