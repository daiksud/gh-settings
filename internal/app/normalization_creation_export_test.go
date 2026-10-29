package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestExportCreationWarningsUseVisibleOriginalSidecars(t *testing.T) {
	for _, test := range []struct {
		name, failure           string
		normalized, issues, prs bool
		warnings                []string
	}{
		{"visible", "", true, true, true, []string{"pr", "issue"}},
		{"canonical", "", false, true, true, nil},
		{"hidden", "", true, false, false, nil},
		{"issues hidden", "", true, false, true, []string{"pr"}},
		{"PRs hidden", "", true, true, false, []string{"issue"}},
		{"failed repository acquisition", "repository", true, true, true, nil},
		{"later failure", "topics", true, true, true, []string{"pr", "issue"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			pr := settings.PRAccessState{HasPullRequests: test.prs, PullRequestCreationPolicy: "ALL"}
			issue := settings.IssueCreationPolicyState{Policy: "ALL"}
			if test.normalized {
				pr.Normalization = settings.ObservationNormalization{Path: "repository.settings.general.features.pull_request_permissions.creation_allowed_by", RawValue: "FUTURE_PR", Fallback: "all_users"}
				issue.Normalization = settings.ObservationNormalization{Path: "repository.settings.general.features.issue_permissions.creation_allowed_by", RawValue: "", Fallback: "all_users"}
			}
			base := settings.RepositoryState{DefaultBranch: "main", HasIssues: test.issues}
			merge := settings.MergeMethodsState{MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE", SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES"}
			observed := settings.NewRepositoryObservation(&base, &merge, &pr).WithIssueCreationPolicy(issue).WithSponsorships(settings.SponsorshipsState{})
			cause := errors.New("fixture acquisition denied")
			reader := exportFixtureReader(t, base, func(string, repository.Repository) error { return nil })
			reader.ObserveRepository = func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
				if test.failure == "repository" {
					return observed, cause
				}
				return observed, nil
			}
			if test.failure == "topics" {
				reader.Topics = func(repository.Repository) ([]string, error) { return nil, cause }
			}
			reader.WorkflowPermissions = func(repository.Repository) (settings.WorkflowPermissionsState, error) {
				t.Fatal("unrequested workflow read")
				return settings.WorkflowPermissionsState{}, nil
			}
			var warnings []settings.ObservationNormalization
			saved := false
			err := Export(ExportCheckout{Repository: checkoutRepository, NormalizationWarning: func(n settings.ObservationNormalization) { warnings = append(warnings, n) }, CommitMessageNormalizationWarning: func(settings.CommitMessageNormalization) { t.Fatal("creation warning misrouted") }, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(declaration settings.Declaration) error {
					ui, err := github.UIFromExportDeclaration(declaration)
					if err != nil {
						return err
					}
					features := ui.Settings.General.Features
					if (features.IssuePermissions.CreationAllowedBy != nil) != test.issues || (features.PullRequestPermissions.CreationAllowedBy != nil) != test.prs {
						t.Fatal("hidden policy became export intent", ui)
					}
					if test.issues && *features.IssuePermissions.CreationAllowedBy != "all_users" || test.prs && *features.PullRequestPermissions.CreationAllowedBy != "all_users" {
						t.Fatal("export lost effective UI choices", ui)
					}
					saved = true
					return nil
				}, nil
			}}, reader)
			var want []settings.ObservationNormalization
			for _, control := range test.warnings {
				if control == "pr" {
					want = append(want, pr.Normalization)
				} else {
					want = append(want, issue.Normalization)
				}
			}
			if saved != (test.failure == "") || !reflect.DeepEqual(warnings, want) || (err != nil) != (test.failure != "") {
				t.Fatalf("saved=%v warnings=%+v want=%+v err=%v", saved, warnings, want, err)
			}
			if test.failure != "" && !errors.Is(err, cause) {
				t.Fatal("acquisition failure lost", err)
			}
		})
	}
}
