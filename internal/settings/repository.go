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
	Description              *string
	Homepage                 *string
	HasIssues                *bool
	HasProjects              *bool
	HasWiki                  *bool
	AllowMergeCommit         *bool
	AllowSquashMerge         *bool
	AllowRebaseMerge         *bool
	DeleteBranchOnMerge      *bool
	AllowAutoMerge           *bool
	AllowUpdateBranch        *bool
	SquashMergeCommitTitle   *string
	SquashMergeCommitMessage *string
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
	return d.AllowMergeCommit == nil && d.AllowSquashMerge == nil && d.AllowRebaseMerge == nil && d.DeleteBranchOnMerge == nil && d.AllowAutoMerge == nil && d.AllowUpdateBranch == nil && d.SquashMergeCommitTitle == nil && d.SquashMergeCommitMessage == nil
}

// MergeMethodsState is complete only after all merge-related GraphQL values are validated.
type MergeMethodsState struct {
	AllowMergeCommit, AllowSquashMerge, AllowRebaseMerge, DeleteBranchOnMerge, AllowAutoMerge, AllowUpdateBranch bool
	SquashMergeCommitTitle, SquashMergeCommitMessage                                                             string
}

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
		{"repository.delete_branch_on_merge", current.DeleteBranchOnMerge, desired.DeleteBranchOnMerge},
		{"repository.allow_auto_merge", current.AllowAutoMerge, desired.AllowAutoMerge},
		{"repository.allow_update_branch", current.AllowUpdateBranch, desired.AllowUpdateBranch},
	} {
		if field.after == nil {
			continue
		}
		result.Managed++
		if field.before != *field.after {
			result.Changes = append(result.Changes, Change{Field: field.name, Before: field.before, After: *field.after})
		}
	}
	for _, field := range []struct {
		name, before string
		after        *string
	}{
		{"repository.squash_merge_commit_title", current.SquashMergeCommitTitle, desired.SquashMergeCommitTitle},
		{"repository.squash_merge_commit_message", current.SquashMergeCommitMessage, desired.SquashMergeCommitMessage},
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

// ValidSquashMergeCommitTitle recognizes the GitHub API's exact title enum.
func ValidSquashMergeCommitTitle(value string) bool {
	return value == "PR_TITLE" || value == "COMMIT_OR_PR_TITLE"
}

// ValidSquashMergeCommitMessage recognizes the GitHub API's exact message enum.
func ValidSquashMergeCommitMessage(value string) bool {
	return value == "PR_BODY" || value == "COMMIT_MESSAGES" || value == "BLANK"
}
