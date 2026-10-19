package settings

import "errors"

// RepositoryState contains the six complete values from the repository REST read.
type RepositoryState struct {
	Description    string
	Homepage       string
	HasIssues      bool
	HasProjects    bool
	HasWiki        bool
	HasDiscussions bool
}

// RepositoryDeclaration leaves nil fields unmanaged; false and empty strings are declared values.
type RepositoryDeclaration struct {
	Description              *string
	Homepage                 *string
	HasIssues                *bool
	HasProjects              *bool
	HasWiki                  *bool
	HasDiscussions           *bool
	AllowMergeCommit         *bool
	AllowSquashMerge         *bool
	AllowRebaseMerge         *bool
	DeleteBranchOnMerge      *bool
	AllowAutoMerge           *bool
	AllowUpdateBranch        *bool
	SquashMergeCommitTitle   *string
	SquashMergeCommitMessage *string
	MergeCommitTitle         *string
	MergeCommitMessage       *string
}

// Empty reports whether every field is unmanaged, regardless of declared values.
func (d RepositoryDeclaration) Empty() bool {
	return d == (RepositoryDeclaration{})
}

// ValidateCommitDependencies requires the corresponding title for each declared message.
func (d RepositoryDeclaration) ValidateCommitDependencies() error {
	if d.SquashMergeCommitMessage != nil && d.SquashMergeCommitTitle == nil {
		return errors.New("squash_merge_commit_message requires squash_merge_commit_title")
	}
	if d.MergeCommitMessage != nil && d.MergeCommitTitle == nil {
		return errors.New("merge_commit_message requires merge_commit_title")
	}
	return nil
}

// BaseEmpty reports whether the REST-observed repository fields are unmanaged.
func (d RepositoryDeclaration) BaseEmpty() bool {
	return d.Description == nil && d.Homepage == nil && d.HasIssues == nil && d.HasProjects == nil && d.HasWiki == nil && d.HasDiscussions == nil
}

// MergeMethodsEmpty keeps the GraphQL read optional for existing declarations.
func (d RepositoryDeclaration) MergeMethodsEmpty() bool {
	return d.AllowMergeCommit == nil && d.AllowSquashMerge == nil && d.AllowRebaseMerge == nil && d.DeleteBranchOnMerge == nil && d.AllowAutoMerge == nil && d.AllowUpdateBranch == nil && d.SquashMergeCommitTitle == nil && d.SquashMergeCommitMessage == nil && d.MergeCommitTitle == nil && d.MergeCommitMessage == nil
}

// MergeMethodsState is complete only after all merge-related GraphQL values are validated.
type MergeMethodsState struct {
	AllowMergeCommit, AllowSquashMerge, AllowRebaseMerge, DeleteBranchOnMerge, AllowAutoMerge, AllowUpdateBranch bool
	SquashMergeCommitTitle, SquashMergeCommitMessage, MergeCommitTitle, MergeCommitMessage                       string
}

// CompareMergeMethods uses only declared fields, in the repository's stable order.
func CompareMergeMethods(current MergeMethodsState, desired RepositoryDeclaration) Comparison {
	var result Comparison
	compareRepositoryFields(&result, mergeMethodsBoolFields[:], current, desired)
	compareRepositoryFields(&result, mergeMethodsStringFields[:], current, desired)
	return result
}

// ValidSquashMergeCommitTitle recognizes the GitHub API's exact title enum.
func ValidSquashMergeCommitTitle(value string) bool {
	return value == "PR_TITLE" || value == "COMMIT_OR_PR_TITLE"
}

// ValidSquashMergeCommitMessage recognizes the GitHub API's exact message enum.
func ValidSquashMergeCommitMessage(value string) bool {
	return value == "PR_BODY" || value == "COMMIT_MESSAGES" || value == "BLANK"
}

// ValidMergeCommitTitle recognizes the GitHub API's exact merge title enum.
func ValidMergeCommitTitle(value string) bool { return value == "PR_TITLE" || value == "MERGE_MESSAGE" }

// ValidMergeCommitMessage recognizes the GitHub API's exact merge body enum.
func ValidMergeCommitMessage(value string) bool {
	return value == "PR_TITLE" || value == "PR_BODY" || value == "BLANK"
}
