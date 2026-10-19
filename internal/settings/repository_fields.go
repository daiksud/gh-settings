package settings

import (
	"fmt"
	"strings"
)

type repositoryField[S any, V bool | string] struct {
	name             string
	current          func(S) V
	desired          func(RepositoryDeclaration) *V
	setDesired       func(*RepositoryDeclaration, V)
	valid            func(V) bool
	valueDescription string
}

func declareRepositoryFields[S any, V bool | string](fields []repositoryField[S, V], state S, declaration *RepositoryDeclaration) {
	for _, field := range fields {
		field.setDesired(declaration, field.current(state))
	}
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

// RepositoryStringSetter returns a validating setter and identifies enum fields.
func RepositoryStringSetter(name string) (set func(*RepositoryDeclaration, string) error, enum, ok bool) {
	for _, field := range repositoryStringFields {
		if field.name == name {
			return validatedStringSetter(field), field.valueDescription != "", true
		}
	}
	for _, field := range mergeMethodsStringFields {
		if field.name == name {
			return validatedStringSetter(field), field.valueDescription != "", true
		}
	}
	return nil, false, false
}

func validatedStringSetter[S any](field repositoryField[S, string]) func(*RepositoryDeclaration, string) error {
	return func(declaration *RepositoryDeclaration, value string) error {
		if !field.valid(value) {
			return fmt.Errorf("%s must be %s", strings.TrimPrefix(field.name, "repository."), field.valueDescription)
		}
		field.setDesired(declaration, value)
		return nil
	}
}

// VisitDeclaredRepositoryFields visits native values in the stable repository field order.
func VisitDeclaredRepositoryFields(declaration RepositoryDeclaration, visitString func(string, string) error, visitBool func(string, bool) error) error {
	if err := visitDeclaredRepositoryFields(repositoryStringFields[:], declaration, visitString); err != nil {
		return err
	}
	if err := visitDeclaredRepositoryFields(repositoryBoolFields[:], declaration, visitBool); err != nil {
		return err
	}
	if err := visitDeclaredRepositoryFields(mergeMethodsBoolFields[:], declaration, visitBool); err != nil {
		return err
	}
	return visitDeclaredRepositoryFields(mergeMethodsStringFields[:], declaration, visitString)
}

func visitDeclaredRepositoryFields[S any, V bool | string](fields []repositoryField[S, V], declaration RepositoryDeclaration, visit func(string, V) error) error {
	for _, field := range fields {
		if value := field.desired(declaration); value != nil {
			if err := visit(field.name, *value); err != nil {
				return err
			}
		}
	}
	return nil
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
		name:       "repository.description",
		current:    func(state RepositoryState) string { return state.Description },
		desired:    func(declaration RepositoryDeclaration) *string { return declaration.Description },
		setDesired: func(declaration *RepositoryDeclaration, value string) { declaration.Description = &value },
		valid:      func(string) bool { return true },
	},
	{
		name:       "repository.homepage",
		current:    func(state RepositoryState) string { return state.Homepage },
		desired:    func(declaration RepositoryDeclaration) *string { return declaration.Homepage },
		setDesired: func(declaration *RepositoryDeclaration, value string) { declaration.Homepage = &value },
		valid:      func(string) bool { return true },
	},
}

var mergeMethodsStringFields = [...]repositoryField[MergeMethodsState, string]{
	{
		name:             "repository.squash_merge_commit_title",
		current:          func(state MergeMethodsState) string { return state.SquashMergeCommitTitle },
		desired:          func(declaration RepositoryDeclaration) *string { return declaration.SquashMergeCommitTitle },
		setDesired:       func(declaration *RepositoryDeclaration, value string) { declaration.SquashMergeCommitTitle = &value },
		valid:            ValidSquashMergeCommitTitle,
		valueDescription: "PR_TITLE or COMMIT_OR_PR_TITLE",
	},
	{
		name:             "repository.squash_merge_commit_message",
		current:          func(state MergeMethodsState) string { return state.SquashMergeCommitMessage },
		desired:          func(declaration RepositoryDeclaration) *string { return declaration.SquashMergeCommitMessage },
		setDesired:       func(declaration *RepositoryDeclaration, value string) { declaration.SquashMergeCommitMessage = &value },
		valid:            ValidSquashMergeCommitMessage,
		valueDescription: "PR_BODY, COMMIT_MESSAGES or BLANK",
	},
	{
		name:             "repository.merge_commit_title",
		current:          func(state MergeMethodsState) string { return state.MergeCommitTitle },
		desired:          func(declaration RepositoryDeclaration) *string { return declaration.MergeCommitTitle },
		setDesired:       func(declaration *RepositoryDeclaration, value string) { declaration.MergeCommitTitle = &value },
		valid:            ValidMergeCommitTitle,
		valueDescription: "PR_TITLE or MERGE_MESSAGE",
	},
	{
		name:             "repository.merge_commit_message",
		current:          func(state MergeMethodsState) string { return state.MergeCommitMessage },
		desired:          func(declaration RepositoryDeclaration) *string { return declaration.MergeCommitMessage },
		setDesired:       func(declaration *RepositoryDeclaration, value string) { declaration.MergeCommitMessage = &value },
		valid:            ValidMergeCommitMessage,
		valueDescription: "PR_BODY, PR_TITLE or BLANK",
	},
}
