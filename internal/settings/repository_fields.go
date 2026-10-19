package settings

type repositoryField[S any, V bool | string] struct {
	name       string
	current    func(S) V
	desired    func(RepositoryDeclaration) *V
	setDesired func(*RepositoryDeclaration, V)
	valid      func(V) bool
}

// RepositoryBoolSetter returns a typed setter for a supported boolean field.
func RepositoryBoolSetter(name string) (func(*RepositoryDeclaration, bool), bool) {
	for _, field := range repositoryBoolFields {
		if field.name == name {
			return field.setDesired, true
		}
	}
	for _, field := range mergeMethodsBoolFields {
		if field.name == name {
			return field.setDesired, true
		}
	}
	return nil, false
}

// RepositoryFieldCount counts the supported scalar repository fields.
func RepositoryFieldCount() int {
	return len(repositoryStringFields) + len(repositoryBoolFields) + len(mergeMethodsBoolFields) + len(mergeMethodsStringFields)
}

// ValidRepositoryFieldValue checks a field's native type and existing value policy.
func ValidRepositoryFieldValue(name string, value any) bool {
	switch value := value.(type) {
	case bool:
		return validRepositoryFieldValue(repositoryBoolFields[:], name, value) || validRepositoryFieldValue(mergeMethodsBoolFields[:], name, value)
	case string:
		return validRepositoryFieldValue(repositoryStringFields[:], name, value) || validRepositoryFieldValue(mergeMethodsStringFields[:], name, value)
	default:
		return false
	}
}

func validRepositoryFieldValue[S any, V bool | string](fields []repositoryField[S, V], name string, value V) bool {
	for _, field := range fields {
		if field.name == name {
			return field.valid == nil || field.valid(value)
		}
	}
	return false
}

var repositoryBoolFields = [...]repositoryField[RepositoryState, bool]{
	{
		name:       "repository.has_issues",
		current:    func(state RepositoryState) bool { return state.HasIssues },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.HasIssues },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.HasIssues = &value },
	},
	{
		name:       "repository.has_projects",
		current:    func(state RepositoryState) bool { return state.HasProjects },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.HasProjects },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.HasProjects = &value },
	},
	{
		name:       "repository.has_wiki",
		current:    func(state RepositoryState) bool { return state.HasWiki },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.HasWiki },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.HasWiki = &value },
	},
	{
		name:       "repository.has_discussions",
		current:    func(state RepositoryState) bool { return state.HasDiscussions },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.HasDiscussions },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.HasDiscussions = &value },
	},
}

var mergeMethodsBoolFields = [...]repositoryField[MergeMethodsState, bool]{
	{
		name:       "repository.allow_merge_commit",
		current:    func(state MergeMethodsState) bool { return state.AllowMergeCommit },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.AllowMergeCommit },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.AllowMergeCommit = &value },
	},
	{
		name:       "repository.allow_squash_merge",
		current:    func(state MergeMethodsState) bool { return state.AllowSquashMerge },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.AllowSquashMerge },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.AllowSquashMerge = &value },
	},
	{
		name:       "repository.allow_rebase_merge",
		current:    func(state MergeMethodsState) bool { return state.AllowRebaseMerge },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.AllowRebaseMerge },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.AllowRebaseMerge = &value },
	},
	{
		name:       "repository.delete_branch_on_merge",
		current:    func(state MergeMethodsState) bool { return state.DeleteBranchOnMerge },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.DeleteBranchOnMerge },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.DeleteBranchOnMerge = &value },
	},
	{
		name:       "repository.allow_auto_merge",
		current:    func(state MergeMethodsState) bool { return state.AllowAutoMerge },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.AllowAutoMerge },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.AllowAutoMerge = &value },
	},
	{
		name:       "repository.allow_update_branch",
		current:    func(state MergeMethodsState) bool { return state.AllowUpdateBranch },
		desired:    func(declaration RepositoryDeclaration) *bool { return declaration.AllowUpdateBranch },
		setDesired: func(declaration *RepositoryDeclaration, value bool) { declaration.AllowUpdateBranch = &value },
	},
}

var repositoryStringFields = [...]repositoryField[RepositoryState, string]{
	{
		name:    "repository.description",
		current: func(state RepositoryState) string { return state.Description },
		desired: func(declaration RepositoryDeclaration) *string { return declaration.Description },
		valid:   func(string) bool { return true },
	},
	{
		name:    "repository.homepage",
		current: func(state RepositoryState) string { return state.Homepage },
		desired: func(declaration RepositoryDeclaration) *string { return declaration.Homepage },
		valid:   func(string) bool { return true },
	},
}

var mergeMethodsStringFields = [...]repositoryField[MergeMethodsState, string]{
	{
		name:    "repository.squash_merge_commit_title",
		current: func(state MergeMethodsState) string { return state.SquashMergeCommitTitle },
		desired: func(declaration RepositoryDeclaration) *string { return declaration.SquashMergeCommitTitle },
		valid:   ValidSquashMergeCommitTitle,
	},
	{
		name:    "repository.squash_merge_commit_message",
		current: func(state MergeMethodsState) string { return state.SquashMergeCommitMessage },
		desired: func(declaration RepositoryDeclaration) *string { return declaration.SquashMergeCommitMessage },
		valid:   ValidSquashMergeCommitMessage,
	},
	{
		name:    "repository.merge_commit_title",
		current: func(state MergeMethodsState) string { return state.MergeCommitTitle },
		desired: func(declaration RepositoryDeclaration) *string { return declaration.MergeCommitTitle },
		valid:   ValidMergeCommitTitle,
	},
	{
		name:    "repository.merge_commit_message",
		current: func(state MergeMethodsState) string { return state.MergeCommitMessage },
		desired: func(declaration RepositoryDeclaration) *string { return declaration.MergeCommitMessage },
		valid:   ValidMergeCommitMessage,
	},
}
