package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// exportRepository acquires each complete core group independently, retaining
// valid previous intent for unavailable groups. Retained values are not observed.
func exportRepository(checkout ExportCheckout, read Reader, repo repository.Repository) (settings.RepositoryDeclaration, error) {
	declaration := checkout.Existing.Repository
	var infoError error
	var creationNormalizations []settings.ObservationNormalization
	var commitNormalizations []settings.CommitMessageNormalization
	for _, group := range settings.RepositoryExportGroups() {
		var observed settings.RepositoryObservation
		err := fmt.Errorf("reader unavailable")
		if read.ObserveRepository != nil {
			observed, err = read.ObserveRepository(repo, group.Fields())
		}
		candidate := declaration
		if err == nil {
			candidate, err = observed.ExportDeclaration(group, declaration)
		}
		if err == nil && checkout.Check != nil {
			var acquired settings.RepositoryDeclaration
			acquired, err = observed.ExportDeclaration(group, settings.RepositoryDeclaration{})
			if err == nil {
				err = checkout.Check(settings.Declaration{Repository: acquired})
			}
		}
		if err != nil {
			if checkout.Info != nil {
				infoError = errors.Join(infoError, checkout.Info(repositoryExportPath(group), err))
			}
			continue
		}
		// The complete group's values replace every owned field, including false
		// and empty values. Suppression below then removes disabled details.
		declaration = candidate
		switch group {
		case settings.RepositoryExportMergeMethods:
			commitNormalizations = observed.CommitMessageNormalizations()
		case settings.RepositoryExportPullRequests, settings.RepositoryExportIssuePolicy:
			for _, normalization := range observed.CreationPolicyNormalizations() {
				if group == settings.RepositoryExportPullRequests && normalization.Path == settings.PullRequestPermissionsControl.Path() || group == settings.RepositoryExportIssuePolicy && normalization.Path == settings.IssuePermissionsControl.Path() {
					creationNormalizations = append(creationNormalizations, normalization)
				}
			}
		}
	}
	// Issues and its policy belong to independent groups. A newly disabled
	// feature must also suppress a retained policy that is no longer valid.
	if exportDisabled(declaration.HasIssues) {
		declaration.IssueCreationPolicy = nil
	}
	if exportDisabled(declaration.HasPullRequests) {
		declaration.PullRequestCreationPolicy = nil
	}
	if exportDisabled(declaration.AllowMergeCommit) {
		declaration.MergeCommitTitle, declaration.MergeCommitMessage = nil, nil
	}
	if exportDisabled(declaration.AllowSquashMerge) {
		declaration.SquashMergeCommitTitle, declaration.SquashMergeCommitMessage = nil, nil
	}
	if checkout.NormalizationWarning != nil {
		for _, normalization := range creationNormalizations {
			if normalization.Path == settings.IssuePermissionsControl.Path() && exportDisabled(declaration.HasIssues) || normalization.Path == settings.PullRequestPermissionsControl.Path() && exportDisabled(declaration.HasPullRequests) {
				continue
			}
			checkout.NormalizationWarning(normalization)
		}
	}
	if checkout.CommitMessageNormalizationWarning != nil {
		for _, normalization := range commitNormalizations {
			if normalization.Path == settings.SquashCommitMenu.Path() && exportDisabled(declaration.AllowSquashMerge) || normalization.Path == settings.MergeCommitMenu.Path() && exportDisabled(declaration.AllowMergeCommit) {
				continue
			}
			checkout.CommitMessageNormalizationWarning(normalization)
		}
	}
	return declaration, infoError
}

func exportDisabled(value *bool) bool { return value != nil && !*value }

func repositoryExportPath(group settings.RepositoryExportGroup) string {
	switch group {
	case settings.RepositoryExportBase:
		return "repository"
	case settings.RepositoryExportMergeMethods:
		return "repository.settings.general.pull_requests"
	case settings.RepositoryExportPullRequests:
		return "repository.settings.general.features.pull_requests"
	case settings.RepositoryExportIssuePolicy:
		return "repository.settings.general.features.issue_permissions"
	case settings.RepositoryExportSponsorships:
		return "repository.settings.general.features.sponsorships"
	default:
		return "repository"
	}
}
