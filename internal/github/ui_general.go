package github

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// GeneralDeclaration translates supported UI controls to adapter intent.
// Archive intent must still use the existing lifecycle preflight/write path.
func GeneralDeclaration(general uimodel.General) (settings.RepositoryDeclaration, error) {
	if general.DefaultBranch != nil && !settings.ValidDefaultBranch(*general.DefaultBranch) {
		return settings.RepositoryDeclaration{}, fmt.Errorf("general.default_branch must be a valid branch name")
	}
	if general.Features.Issues != nil && !*general.Features.Issues && general.Features.IssuePermissions.CreationAllowedBy != nil {
		return settings.RepositoryDeclaration{}, fmt.Errorf("issue_permissions require enabled Issues or an omitted feature checkbox")
	}
	if general.Features.PullRequests != nil && !*general.Features.PullRequests && general.Features.PullRequestPermissions.CreationAllowedBy != nil {
		return settings.RepositoryDeclaration{}, fmt.Errorf("pull_request_permissions require enabled Pull requests or an omitted feature checkbox")
	}
	result := settings.RepositoryDeclaration{
		IsTemplate: copyUIValue(general.TemplateRepository), DefaultBranch: copyUIValue(general.DefaultBranch),
		HasWiki: copyUIValue(general.Features.Wikis), HasIssues: copyUIValue(general.Features.Issues),
		HasProjects: copyUIValue(general.Features.Projects), HasDiscussions: copyUIValue(general.Features.Discussions),
		HasSponsorships: copyUIValue(general.Features.Sponsorships), HasPullRequests: copyUIValue(general.Features.PullRequests),
		AllowMergeCommit: copyUIValue(general.PullRequests.AllowMergeCommits), AllowSquashMerge: copyUIValue(general.PullRequests.AllowSquashMerging),
		AllowRebaseMerge: copyUIValue(general.PullRequests.AllowRebaseMerging), AllowUpdateBranch: copyUIValue(general.PullRequests.AlwaysSuggestUpdatingPullRequestBranches),
		AllowAutoMerge: copyUIValue(general.PullRequests.AllowAutoMerge), DeleteBranchOnMerge: copyUIValue(general.PullRequests.AutomaticallyDeleteHeadBranches),
		WebCommitSignoffRequired: copyUIValue(general.Commits.RequireContributorsToSignOffOnWebBasedCommits),
		Archived:                 copyUIValue(general.DangerZone.ArchiveThisRepository),
	}
	var err error
	result.MergeCommitTitle, result.MergeCommitMessage, err = declareCommitMessage(general.PullRequests.MergeCommit.DefaultCommitMessage, general.PullRequests.AllowMergeCommits, settings.MergeCommitMenu)
	if err != nil {
		return settings.RepositoryDeclaration{}, err
	}
	result.SquashMergeCommitTitle, result.SquashMergeCommitMessage, err = declareCommitMessage(general.PullRequests.SquashMerge.DefaultCommitMessage, general.PullRequests.AllowSquashMerging, settings.SquashCommitMenu)
	if err != nil {
		return settings.RepositoryDeclaration{}, err
	}
	result.IssueCreationPolicy, err = declareCreationPolicy(general.Features.IssuePermissions.CreationAllowedBy, "issue_permissions")
	if err != nil {
		return settings.RepositoryDeclaration{}, err
	}
	result.PullRequestCreationPolicy, err = declareCreationPolicy(general.Features.PullRequestPermissions.CreationAllowedBy, "pull_request_permissions")
	if err != nil {
		return settings.RepositoryDeclaration{}, err
	}
	return result, nil
}

// GeneralConditionalValues retains owned API values behind conditional controls.
// Nil values remain independently unacquired; these never supply desired intent.
type GeneralConditionalValues struct {
	IssueCreationPolicy, PullRequestCreationPolicy   *string
	MergeCommitTitle, MergeCommitMessage             *string
	SquashMergeCommitTitle, SquashMergeCommitMessage *string
}
type PreparedGeneral struct {
	uimodel.General
	RawConditionalValues GeneralConditionalValues
}

// GeneralObservation projects acquired, validated adapter values, preserving
// which independently requested fields were actually observed.
func GeneralObservation(state settings.RepositoryDeclaration) (PreparedGeneral, error) {
	result := uimodel.General{
		TemplateRepository: copyUIValue(state.IsTemplate), DefaultBranch: copyUIValue(state.DefaultBranch),
		Features: uimodel.Features{
			Wikis: copyUIValue(state.HasWiki), Issues: copyUIValue(state.HasIssues), Projects: copyUIValue(state.HasProjects),
			Discussions: copyUIValue(state.HasDiscussions), Sponsorships: copyUIValue(state.HasSponsorships), PullRequests: copyUIValue(state.HasPullRequests),
		},
		PullRequests: uimodel.PullRequests{
			AllowMergeCommits: copyUIValue(state.AllowMergeCommit), AllowSquashMerging: copyUIValue(state.AllowSquashMerge),
			AllowRebaseMerging: copyUIValue(state.AllowRebaseMerge), AlwaysSuggestUpdatingPullRequestBranches: copyUIValue(state.AllowUpdateBranch),
			AllowAutoMerge: copyUIValue(state.AllowAutoMerge), AutomaticallyDeleteHeadBranches: copyUIValue(state.DeleteBranchOnMerge),
		},
		Commits:    uimodel.Commits{RequireContributorsToSignOffOnWebBasedCommits: copyUIValue(state.WebCommitSignoffRequired)},
		DangerZone: uimodel.DangerZone{ArchiveThisRepository: copyUIValue(state.Archived)},
	}
	var err error
	result.PullRequests.MergeCommit.DefaultCommitMessage, err = observeCommitMessage(state.MergeCommitTitle, state.MergeCommitMessage, settings.MergeCommitMenu)
	if err != nil {
		return PreparedGeneral{}, err
	}
	result.PullRequests.SquashMerge.DefaultCommitMessage, err = observeCommitMessage(state.SquashMergeCommitTitle, state.SquashMergeCommitMessage, settings.SquashCommitMenu)
	if err != nil {
		return PreparedGeneral{}, err
	}
	result.Features.IssuePermissions.CreationAllowedBy, err = observeCreationPolicy(state.IssueCreationPolicy, "issue_permissions")
	if err != nil {
		return PreparedGeneral{}, err
	}
	result.Features.PullRequestPermissions.CreationAllowedBy, err = observeCreationPolicy(state.PullRequestCreationPolicy, "pull_request_permissions")
	if err != nil {
		return PreparedGeneral{}, err
	}
	if state.HasIssues != nil && !*state.HasIssues {
		result.Features.IssuePermissions.CreationAllowedBy = nil
	}
	if state.HasPullRequests != nil && !*state.HasPullRequests {
		result.Features.PullRequestPermissions.CreationAllowedBy = nil
	}
	if state.AllowMergeCommit != nil && !*state.AllowMergeCommit {
		result.PullRequests.MergeCommit.DefaultCommitMessage = nil
	}
	if state.AllowSquashMerge != nil && !*state.AllowSquashMerge {
		result.PullRequests.SquashMerge.DefaultCommitMessage = nil
	}
	return PreparedGeneral{General: result, RawConditionalValues: GeneralConditionalValues{
		IssueCreationPolicy: copyUIValue(state.IssueCreationPolicy), PullRequestCreationPolicy: copyUIValue(state.PullRequestCreationPolicy),
		MergeCommitTitle: copyUIValue(state.MergeCommitTitle), MergeCommitMessage: copyUIValue(state.MergeCommitMessage),
		SquashMergeCommitTitle: copyUIValue(state.SquashMergeCommitTitle), SquashMergeCommitMessage: copyUIValue(state.SquashMergeCommitMessage),
	}}, nil
}
