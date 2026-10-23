package settings

import "errors"

// RepositoryState contains the nine complete values from the repository REST read.
type RepositoryState struct {
	Description              string
	Homepage                 string
	DefaultBranch            string
	HasIssues                bool
	HasProjects              bool
	HasWiki                  bool
	HasDiscussions           bool
	WebCommitSignoffRequired bool
	IsTemplate               bool
}

// Declaration explicitly manages every field in the complete REST observation.
func (state RepositoryState) Declaration() RepositoryDeclaration {
	var declaration RepositoryDeclaration
	declareRepositoryFields(repositoryStringFields[:], state, &declaration)
	declareRepositoryFields(repositoryBoolFields[:], state, &declaration)
	return declaration
}

// RepositoryDeclaration leaves nil fields unmanaged; false and empty strings are declared values.
type RepositoryDeclaration struct {
	Description               *string
	Homepage                  *string
	DefaultBranch             *string
	HasIssues                 *bool
	HasProjects               *bool
	HasWiki                   *bool
	HasDiscussions            *bool
	WebCommitSignoffRequired  *bool
	IsTemplate                *bool
	AllowMergeCommit          *bool
	AllowSquashMerge          *bool
	AllowRebaseMerge          *bool
	DeleteBranchOnMerge       *bool
	AllowAutoMerge            *bool
	AllowUpdateBranch         *bool
	SquashMergeCommitTitle    *string
	SquashMergeCommitMessage  *string
	MergeCommitTitle          *string
	MergeCommitMessage        *string
	HasPullRequests           *bool
	PullRequestCreationPolicy *string
	IssueCreationPolicy       *string
	HasSponsorships           *bool
	Archived                  *bool
	AllowForking              *bool
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
	return d.Description == nil && d.Homepage == nil && d.DefaultBranch == nil && d.HasIssues == nil && d.HasProjects == nil && d.HasWiki == nil && d.HasDiscussions == nil && d.WebCommitSignoffRequired == nil && d.IsTemplate == nil
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

// WithMergeMethods explicitly includes the complete merge observation and preserves the base fields.
func (d RepositoryDeclaration) WithMergeMethods(state MergeMethodsState) RepositoryDeclaration {
	declareRepositoryFields(mergeMethodsBoolFields[:], state, &d)
	declareRepositoryFields(mergeMethodsStringFields[:], state, &d)
	return d
}

// CompareMergeMethods uses only declared fields, in the repository's stable order.
func CompareMergeMethods(current MergeMethodsState, desired RepositoryDeclaration) Comparison {
	var result Comparison
	compareRepositoryFields(&result, mergeMethodsBoolFields[:], current, desired, boolChange)
	compareRepositoryFields(&result, mergeMethodsStringFields[:], current, desired, stringChange)
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

// PRAccessState is the complete observation of repository pull request access.
type PRAccessState struct {
	HasPullRequests           bool
	PullRequestCreationPolicy string
}

func ValidPullRequestCreationPolicy(value string) bool {
	return value == "ALL" || value == "COLLABORATORS_ONLY"
}
func (d RepositoryDeclaration) PRAccessEmpty() bool {
	return d.HasPullRequests == nil && d.PullRequestCreationPolicy == nil
}
func (d RepositoryDeclaration) WithPRAccess(state PRAccessState) RepositoryDeclaration {
	declareRepositoryFields(prAccessBoolFields[:], state, &d)
	declareRepositoryFields(prAccessStringFields[:], state, &d)
	return d
}

// IssueCreationPolicyState contains a complete independently acquired policy.
type IssueCreationPolicyState struct{ Policy string }

func ValidIssueCreationPolicy(value string) bool {
	return value == "ALL" || value == "COLLABORATORS_ONLY"
}
func (d RepositoryDeclaration) RepositoryAccessEmpty() bool {
	return d.PRAccessEmpty() && d.IssueCreationPolicy == nil && d.HasSponsorships == nil
}

func (d RepositoryDeclaration) ValidateForkingComposition() error {
	if d.AllowForking != nil && d.Archived != nil {
		return errors.New("allow_forking cannot be combined with archived; manage lifecycle separately")
	}
	return nil
}
