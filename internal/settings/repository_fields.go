package settings

type repositoryBoolField struct {
	name    string
	current func(RepositoryState) bool
	desired func(RepositoryDeclaration) *bool
}

var discussionsFields = [...]repositoryBoolField{
	{
		name:    "repository.has_discussions",
		current: func(state RepositoryState) bool { return state.HasDiscussions },
		desired: func(declaration RepositoryDeclaration) *bool { return declaration.HasDiscussions },
	},
}
