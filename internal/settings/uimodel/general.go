package uimodel

// General groups supported controls under the GitHub General page headings.
// Every pointer distinguishes omission from an explicitly selected value.
type General struct {
	TemplateRepository *bool
	DefaultBranch      *string
	Releases           *SecurityGeneralReleases
	Features           Features
	PullRequests       PullRequests
	Commits            Commits
	DangerZone         DangerZone
}

type Features struct {
	Wikis                  WikiSettings
	Issues                 *bool
	IssuePermissions       CreationPermissions
	Sponsorships           *bool
	Discussions            *bool
	Projects               *bool
	PullRequests           *bool
	PullRequestPermissions CreationPermissions
}

type WikiSettings struct{ Enabled *bool }

type CreationAllowedBy string

const (
	AllUsers                            CreationAllowedBy = "all_users"
	CollaboratorsOnly                   CreationAllowedBy = "collaborators_only"
	DefaultIssueCreationAllowedBy                         = AllUsers
	DefaultPullRequestCreationAllowedBy                   = AllUsers
)

type CreationPermissions struct{ CreationAllowedBy *CreationAllowedBy }

type PullRequests struct {
	AllowMergeCommits                        *bool
	MergeCommits                             CommitMessage
	AllowSquashMerging                       *bool
	SquashMerging                            CommitMessage
	AllowRebaseMerging                       *bool
	AlwaysSuggestUpdatingPullRequestBranches *bool
	AllowAutoMerge                           *bool
	AutomaticallyDeleteHeadBranches          *bool
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
