package settings

type repositoryField[S any, V bool | string] struct {
	name    string
	current func(S) V
	desired func(RepositoryDeclaration) *V
	valid   func(V) bool
}

var repositoryBoolFields = [...]repositoryField[RepositoryState, bool]{
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

var mergeMethodsBoolFields = [...]repositoryField[MergeMethodsState, bool]{
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
