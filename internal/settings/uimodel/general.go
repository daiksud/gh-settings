package uimodel

// General groups supported controls under the GitHub General page headings.
// Every pointer distinguishes omission from an explicitly selected value.
type General struct {
	TemplateRepository *bool
	DefaultBranch      *string
	Features           Features
	PullRequests       PullRequests
	Commits            Commits
	DangerZone         DangerZone
	Releases           *SecurityGeneralReleases
}

type Features struct {
	Wikis, Issues, Sponsorships, Discussions, Projects, PullRequests *bool
	IssuePermissions, PullRequestPermissions                         CreationPermissions
}

type CreationAllowedBy string

const (
	AllUsers                            CreationAllowedBy = "all_users"
	CollaboratorsOnly                   CreationAllowedBy = "collaborators_only"
	DefaultIssueCreationAllowedBy                         = AllUsers
	DefaultPullRequestCreationAllowedBy                   = AllUsers
)

type CreationPermissions struct{ CreationAllowedBy *CreationAllowedBy }

type PullRequests struct {
	AllowMergeCommits, AllowSquashMerging, AllowRebaseMerging *bool
	AlwaysSuggestUpdatingPullRequestBranches                  *bool
	AllowAutoMerge, AutomaticallyDeleteHeadBranches           *bool
	MergeCommit, SquashMerge                                  CommitMessage
}

// CommitMessageChoice uses the displayed menu wording in snake_case.
type CommitMessageChoice string

const (
	DefaultMessage                   CommitMessageChoice = "default_message"
	PullRequestTitle                 CommitMessageChoice = "pull_request_title"
	PullRequestTitleAndCommitDetails CommitMessageChoice = "pull_request_title_and_commit_details"
	PullRequestTitleAndDescription   CommitMessageChoice = "pull_request_title_and_description"
	// These observation fallbacks are independent from omitted desired fields.
	DefaultMergeCommitMessage       = DefaultMessage
	DefaultSquashMergeCommitMessage = DefaultMessage
)

type CommitMessage struct{ DefaultCommitMessage *CommitMessageChoice }

type Commits struct {
	RequireContributorsToSignOffOnWebBasedCommits *bool
}

type DangerZone struct {
	ArchiveThisRepository *bool
}
