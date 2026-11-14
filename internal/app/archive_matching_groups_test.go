package app

import (
	"errors"
	"fmt"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedMixedGroupsRequireCompleteMatchingEvidence(t *testing.T) {
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	yes, description, effort, approval, days := true, "same", "balanced", "first_time_contributors", 30
	title, message := "MERGE_MESSAGE", "PR_TITLE"
	mode := "legacy"
	no := false
	exemptions, err := settings.NewPullRequestExemptionsDeclaration([]string{})
	if err != nil {
		t.Fatal(err)
	}
	declaration := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes, Description: &description, HasIssues: &yes, AllowMergeCommit: &yes, MergeCommitTitle: &title, MergeCommitMessage: &message}, PullRequestLimits: settings.PullRequestLimitsDeclaration{Enabled: &yes}, PullRequestExemptions: &exemptions, CopilotReviewEffort: &effort, Actions: settings.ActionsPolicyDeclaration{RetentionDays: &days, ForkApprovalPolicy: &approval}, Pages: settings.PagesDeclaration{BuildType: &mode, Source: &settings.PagesSource{Branch: "none"}}, Security: settings.SecurityDeclaration{PrivateVulnerabilityReporting: &no}}
	cases := []string{"match", "repository-diff", "repository-missing", "repository-denied", "merge-missing", "merge-diff", "merge-normalized", "limits-diff", "limits-missing", "limits-denied", "exemptions-diff", "exemptions-missing", "exemptions-denied", "effort-diff", "effort-missing", "effort-denied", "effort-unverified-default", "retention-diff", "retention-missing", "retention-denied", "fork-diff", "fork-missing", "fork-denied", "fork-normalized", "pages-diff", "pages-missing", "pages-denied", "security-diff", "security-missing", "identity-drift", "unsupported"}
	for _, scenario := range cases {
		t.Run(scenario, func(t *testing.T) {
			d := declaration
			if scenario == "effort-unverified-default" {
				v := "default"
				d.CopilotReviewEffort = &v
			}
			if scenario == "unsupported" {
				d.CopilotMCP = &yes
			}
			reads, writes := 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					reads++
					id := int64(1)
					if scenario == "identity-drift" && reads == 2 {
						id = 2
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
				},
				ObserveRepository: func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
					if scenario == "repository-denied" {
						return settings.RepositoryObservation{}, errors.New("denied")
					}
					if scenario == "repository-missing" {
						return settings.RepositoryObservation{}, nil
					}
					value := description
					if scenario == "repository-diff" {
						value = "different"
					}
					merge := &settings.MergeMethodsState{AllowMergeCommit: true, MergeCommitTitle: title, MergeCommitMessage: message}
					if scenario == "merge-missing" {
						merge = nil
					}
					if scenario == "merge-diff" {
						merge.MergeCommitMessage = "PR_BODY"
					}
					if scenario == "merge-normalized" {
						fallback, ok := settings.CommitMessageChoiceForPair(settings.MergeCommitMenu, settings.CommitMessagePair{Title: title, Message: message})
						if !ok {
							t.Fatal("fixture choice")
						}
						merge.MergeCommitNormalization = settings.CommitMessageNormalization{Path: "repository.settings.general.pull_requests.merge_commit.default_commit_message", RawTitle: "FUTURE", RawMessage: "FUTURE", Fallback: fallback}
					}
					return settings.NewRepositoryObservation(&settings.RepositoryState{Description: value, HasIssues: true}, merge, nil), nil
				},
				ReadPullRequestLimits: func(repository.Repository) (settings.PullRequestLimitsObservation, error) {
					if scenario == "limits-denied" {
						return settings.PullRequestLimitsObservation{}, errors.New("denied")
					}
					if scenario == "limits-missing" {
						return settings.PullRequestLimitsObservation{}, nil
					}
					return settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: scenario != "limits-diff", Maximum: 5})
				},
				ReadPullRequestExemptions: func(repository.Repository) (settings.PullRequestExemptionsObservation, error) {
					if scenario == "exemptions-denied" {
						return settings.PullRequestExemptionsObservation{}, errors.New("denied")
					}
					if scenario == "exemptions-missing" {
						return settings.PullRequestExemptionsObservation{}, nil
					}
					users := []settings.PullRequestExemptionUser{}
					if scenario == "exemptions-diff" {
						users = append(users, settings.PullRequestExemptionUser{ID: 1, Login: "sample", Type: "User"})
					}
					return settings.NewPullRequestExemptionsObservation(users)
				},
				ReadCopilotReviewEffort: func(repository.Repository) (settings.CopilotReviewEffortObservation, error) {
					if scenario == "effort-denied" {
						return settings.CopilotReviewEffortObservation{}, errors.New("denied")
					}
					if scenario == "effort-missing" {
						return settings.CopilotReviewEffortObservation{}, nil
					}
					if scenario == "effort-unverified-default" {
						return settings.NewCopilotReviewEffortObservation(nil, "balanced", []string{"balanced", "lite"})
					}
					v := "balanced"
					if scenario == "effort-diff" {
						v = "lite"
					}
					return settings.NewCopilotReviewEffortObservation(&v, v, []string{"balanced", "lite"})
				},
				ReadRetention: func(repository.Repository) (settings.RetentionObservation, error) {
					if scenario == "retention-denied" {
						return settings.RetentionObservation{}, errors.New("denied")
					}
					if scenario == "retention-missing" {
						return settings.RetentionObservation{}, nil
					}
					v := 30
					if scenario == "retention-diff" {
						v = 20
					}
					return settings.NewRetentionObservation(v, 90)
				},
				ReadForkApproval: func(repository.Repository) (settings.ForkApprovalObservation, error) {
					if scenario == "fork-denied" {
						return settings.ForkApprovalObservation{}, errors.New("denied")
					}
					if scenario == "fork-missing" {
						return settings.ForkApprovalObservation{}, nil
					}
					if scenario == "fork-normalized" {
						return settings.NewForkApprovalObservationWithNormalization("first_time_contributors", settings.ObservationNormalization{Path: settings.ForkApprovalPath, RawValue: "FUTURE", Fallback: string(settings.DefaultForkApprovalChoice)})
					}
					v := approval
					if scenario == "fork-diff" {
						v = "all_external_contributors"
					}
					return settings.NewForkApprovalObservation(v)
				},
				ReadPages: func(repository.Repository) (settings.PagesObservation, error) {
					if scenario == "pages-denied" {
						return settings.PagesObservation{}, errors.New("denied")
					}
					if scenario == "pages-missing" {
						return settings.PagesObservation{}, nil
					}
					if scenario == "pages-diff" {
						return settings.NewPagesObservation(settings.PagesState{BuildType: "workflow"})
					}
					return settings.NewAbsentPagesObservation(), nil
				},
				ReadPrivateReporting: func(repository.Repository) (settings.PrivateReportingObservation, error) {
					if scenario == "security-missing" {
						return settings.PrivateReportingObservation{}, nil
					}
					return settings.NewPrivateReportingObservation(scenario == "security-diff"), nil
				},
				Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
					writes++
					return false, errors.New("unexpected mutation")
				},
				Archive: func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				Topics:  func(repository.Repository, settings.TopicsDeclaration) (bool, error) { writes++; return false, nil },
				PullRequestLimits: func(repository.Repository, settings.PullRequestLimitsDeclaration) (bool, error) {
					writes++
					return false, nil
				},
				PullRequestExemptions: func(repository.Repository, settings.PullRequestExemptionsDeclaration) (bool, error) {
					writes++
					return false, nil
				},
				CopilotReviewEffort: func(repository.Repository, string) (bool, error) { writes++; return false, nil },
				Retention:           func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				ForkApproval:        func(repository.Repository, string) (bool, error) { writes++; return false, nil },
				Pages:               func(repository.Repository, settings.PagesDeclaration) (bool, error) { writes++; return false, nil },
				PrivateReporting:    func(repository.Repository, settings.SecurityChange) (bool, error) { writes++; return false, nil },
				CreatedID:           func(string, string, int64) error { writes++; return nil },
				BindCollaboratorID:  func(string, int64) error { writes++; return nil },
			}
			r, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (scenario == "match") {
				t.Fatalf("outcome=%v err=%v", r.Outcome, err)
			}
			if writes != 0 {
				t.Fatalf("unexpected remote mutation or local binding: %d", writes)
			}
			if scenario == "match" && reads != 2 {
				t.Fatal(fmt.Sprintf("expected 2 archive reads, got %d", reads))
			}
		})
	}
}
