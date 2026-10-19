package settings

type repositoryBoolField[S any] struct {
	name    string
	current func(S) bool
	desired func(RepositoryDeclaration) *bool
}

var repositoryBoolFields = [...]repositoryBoolField[RepositoryState]{
	{
		name:    "repository.has_issues",
		current: func(state RepositoryState) bool { return state.HasIssues },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.HasIssues },
	},
	{
		name:    "repository.has_projects",
		current: func(state RepositoryState) bool { return state.HasProjects },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.HasProjects },
	},
	{
		name:    "repository.has_wiki",
		current: func(state RepositoryState) bool { return state.HasWiki },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.HasWiki },
	},
	{
		name:    "repository.has_discussions",
		current: func(state RepositoryState) bool { return state.HasDiscussions },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.HasDiscussions },
	},
}

var mergeMethodsBoolFields = [...]repositoryBoolField[MergeMethodsState]{
	{
		name:    "repository.allow_merge_commit",
		current: func(state MergeMethodsState) bool { return state.AllowMergeCommit },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.AllowMergeCommit },
	},
	{
		name:    "repository.allow_squash_merge",
		current: func(state MergeMethodsState) bool { return state.AllowSquashMerge },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.AllowSquashMerge },
	},
	{
		name:    "repository.allow_rebase_merge",
		current: func(state MergeMethodsState) bool { return state.AllowRebaseMerge },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.AllowRebaseMerge },
	},
	{
		name:    "repository.delete_branch_on_merge",
		current: func(state MergeMethodsState) bool { return state.DeleteBranchOnMerge },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.DeleteBranchOnMerge },
	},
	{
		name:    "repository.allow_auto_merge",
		current: func(state MergeMethodsState) bool { return state.AllowAutoMerge },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.AllowAutoMerge },
	},
	{
		name:    "repository.allow_update_branch",
		current: func(state MergeMethodsState) bool { return state.AllowUpdateBranch },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.AllowUpdateBranch },
	},
}
