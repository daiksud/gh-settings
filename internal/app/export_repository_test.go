package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestExportRepositoryUnavailableGroupsRetainOnlyPriorIntent(t *testing.T) {
	for _, failure := range []string{"repository", "merge", "pr", "issue", "sponsorships", "all"} {
		for _, mode := range []string{"error", "unacquired", "projection"} {
			for _, existing := range []bool{false, true} {
				name := failure + "/" + mode + map[bool]string{false: "/initial", true: "/existing"}[existing]
				t.Run(name, func(t *testing.T) {
					yes := true
					var prior settings.Declaration
					if existing {
						prior.Repository = settings.RepositoryDeclaration{Description: appString("retained"), AllowAutoMerge: &yes, HasPullRequests: &yes, IssueCreationPolicy: appString("COLLABORATORS_ONLY"), HasSponsorships: &yes, Archived: &yes}
					}
					var calls, infos []string
					cause := errors.New("unavailable observation")
					reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main", HasIssues: true}, func(name string, _ repository.Repository) error {
						calls = append(calls, name)
						if mode == "error" && (name == failure || failure == "all" && name != "topics" && name != "labels") {
							return cause
						}
						return nil
					})
					read := reader.ObserveRepository
					reader.ObserveRepository = func(repo repository.Repository, fields []settings.FieldID) (settings.RepositoryObservation, error) {
						observed, err := read(repo, fields)
						if mode == "unacquired" && (calls[len(calls)-1] == failure || failure == "all") {
							return settings.RepositoryObservation{}, nil
						}
						return observed, err
					}
					var saved settings.Declaration
					err := Export(ExportCheckout{
						Existing: prior, Repository: checkoutRepository,
						Check: func(d settings.Declaration) error {
							if mode == "projection" && d.Topics.Names == nil && d.Labels == nil && (calls[len(calls)-1] == failure || failure == "all" && len(calls) <= 5) {
								return cause
							}
							return github.CheckExportAcquisition(d)
						},
						Info: func(path string, err error) error {
							if len(calls) <= 5 {
								infos = append(infos, path)
								if mode != "unacquired" && !errors.Is(err, cause) {
									t.Fatal("observation error lost", err)
								}
							}
							return nil
						},
						PrepareSave: func() (func(settings.Declaration) error, error) {
							return func(d settings.Declaration) error { saved = d; return nil }, nil
						},
					}, reader)
					if err != nil || !reflect.DeepEqual(calls, []string{"repository", "merge", "pr", "issue", "sponsorships", "topics", "labels"}) || saved.Topics.Names == nil || saved.Labels == nil {
						t.Fatalf("independent acquisition stopped: calls=%v, err=%v", calls, err)
					}
					wantInfo := 1
					if failure == "all" {
						wantInfo = 5
					}
					if len(infos) != wantInfo || !reflect.DeepEqual(saved.Repository.Archived, prior.Repository.Archived) {
						t.Fatalf("info=%v archive=%v", infos, saved.Repository.Archived)
					}
					for _, test := range []struct {
						group       string
						got, before any
					}{
						{"repository", saved.Repository.Description, prior.Repository.Description},
						{"merge", saved.Repository.AllowAutoMerge, prior.Repository.AllowAutoMerge},
						{"pr", saved.Repository.HasPullRequests, prior.Repository.HasPullRequests},
						{"issue", saved.Repository.IssueCreationPolicy, prior.Repository.IssueCreationPolicy},
						{"sponsorships", saved.Repository.HasSponsorships, prior.Repository.HasSponsorships},
					} {
						retained := reflect.DeepEqual(test.got, test.before)
						if retained != (failure == test.group || failure == "all") {
							t.Fatalf("%s retention=%v: got=%v before=%v", test.group, retained, test.got, test.before)
						}
					}
					if failure == "repository" || failure == "all" {
						if saved.Repository.Homepage != nil || saved.Repository.HasIssues != nil || saved.Repository.DefaultBranch != nil {
							t.Fatal("unavailable base invented values for omitted prior fields")
						}
					}
					if _, err := github.UIFromExportDeclaration(saved); err != nil {
						t.Fatal("retained intent cannot be serialized", err)
					}
					if existing && *prior.Repository.Description != "retained" {
						t.Fatal("export mutated previous intent")
					}
				})
			}
		}
	}
}

func TestExportRepositoryDisabledFeaturesRemoveRetainedDetails(t *testing.T) {
	yes := true
	prior := settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &yes, IssueCreationPolicy: appString("COLLABORATORS_ONLY"), HasPullRequests: &yes, PullRequestCreationPolicy: appString("COLLABORATORS_ONLY"), AllowMergeCommit: &yes, MergeCommitTitle: appString("PR_TITLE"), MergeCommitMessage: appString("PR_BODY"), AllowSquashMerge: &yes, SquashMergeCommitTitle: appString("PR_TITLE"), SquashMergeCommitMessage: appString("PR_BODY")}}
	reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(name string, _ repository.Repository) error {
		if name == "issue" {
			return errors.New("denied")
		}
		return nil
	})
	read := reader.ObserveRepository
	reader.ObserveRepository = func(repo repository.Repository, fields []settings.FieldID) (settings.RepositoryObservation, error) {
		if fields[0].Path() == "repository.allow_merge_commit" {
			return settings.NewRepositoryObservation(nil, &settings.MergeMethodsState{MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE", SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES"}, nil), nil
		}
		return read(repo, fields)
	}
	var saved settings.Declaration
	err := Export(ExportCheckout{Existing: prior, Check: github.CheckExportAcquisition, Repository: checkoutRepository, PrepareSave: func() (func(settings.Declaration) error, error) {
		return func(d settings.Declaration) error { saved = d; return nil }, nil
	}}, reader)
	if err != nil || saved.Repository.IssueCreationPolicy != nil || saved.Repository.PullRequestCreationPolicy != nil || saved.Repository.MergeCommitTitle != nil || saved.Repository.MergeCommitMessage != nil || saved.Repository.SquashMergeCommitTitle != nil || saved.Repository.SquashMergeCommitMessage != nil {
		t.Fatalf("disabled feature retained incompatible details: %+v, %v", saved.Repository, err)
	}
	if !*prior.Repository.HasIssues || prior.Repository.IssueCreationPolicy == nil {
		t.Fatal("suppression mutated previous declaration")
	}
}

func TestExportRepositoryInfoFailureStillReadsSiblingsButPreventsSave(t *testing.T) {
	cause := errors.New("diagnostic write failed")
	var calls []string
	reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(name string, _ repository.Repository) error {
		calls = append(calls, name)
		if name == "merge" {
			return errors.New("denied")
		}
		return nil
	})
	err := Export(ExportCheckout{Repository: checkoutRepository, Info: func(path string, _ error) error {
		if path == "repository.settings.general.pull_requests" {
			return cause
		}
		return nil
	}, PrepareSave: func() (func(settings.Declaration) error, error) {
		return func(settings.Declaration) error { t.Fatal("saved after diagnostic failure"); return nil }, nil
	}}, reader)
	if !errors.Is(err, cause) || !reflect.DeepEqual(calls, []string{"repository", "merge", "pr", "issue", "sponsorships", "topics", "labels"}) {
		t.Fatal("diagnostic failure boundary changed", calls, err)
	}
}

func TestExportRepositoryWarningsUseOnlySuccessfulGroups(t *testing.T) {
	for _, failure := range []string{"repository", "merge", "pr", "issue", "all"} {
		for _, rejectedByCheck := range []bool{false, true} {
			t.Run(failure+map[bool]string{false: "/read", true: "/projection"}[rejectedByCheck], func(t *testing.T) {
				prRecord := settings.ObservationNormalization{Path: settings.PullRequestPermissionsControl.Path(), RawValue: "FUTURE", Fallback: "all_users"}
				issueRecord := settings.ObservationNormalization{Path: settings.IssuePermissionsControl.Path(), RawValue: "FUTURE", Fallback: "all_users"}
				commitRecord := settings.CommitMessageNormalization{Path: settings.MergeCommitMenu.Path(), RawTitle: "MERGE_MESSAGE", RawMessage: "BLANK", Fallback: "default_message"}
				base := settings.RepositoryState{DefaultBranch: "main", HasIssues: true}
				merge := settings.MergeMethodsState{AllowMergeCommit: true, MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE", SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitNormalization: commitRecord}
				pr := settings.PRAccessState{HasPullRequests: true, PullRequestCreationPolicy: "ALL", Normalization: prRecord}
				observed := settings.NewRepositoryObservation(&base, &merge, &pr).WithIssueCreationPolicy(settings.IssueCreationPolicyState{Policy: "ALL", Normalization: issueRecord}).WithSponsorships(settings.SponsorshipsState{})
				groups := []string{"repository", "merge", "pr", "issue", "sponsorships"}
				index := -1
				var creation []settings.ObservationNormalization
				var commits []settings.CommitMessageNormalization
				var saved settings.Declaration
				err := Export(ExportCheckout{Repository: checkoutRepository,
					Check: func(d settings.Declaration) error {
						if rejectedByCheck && (groups[index] == failure || failure == "all") {
							return errors.New("projection unavailable")
						}
						return github.CheckExportAcquisition(d)
					},
					NormalizationWarning:              func(record settings.ObservationNormalization) error { creation = append(creation, record); return nil },
					CommitMessageNormalizationWarning: func(record settings.CommitMessageNormalization) error { commits = append(commits, record); return nil },
					PrepareSave: func() (func(settings.Declaration) error, error) {
						return func(d settings.Declaration) error { saved = d; return nil }, nil
					},
				}, Reader{ObserveRepository: func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
					index++
					if !rejectedByCheck && (groups[index] == failure || failure == "all") {
						return observed, errors.New("read unavailable")
					}
					return observed, nil
				}})
				var wantCreation []settings.ObservationNormalization
				var wantCommits []settings.CommitMessageNormalization
				if failure != "all" {
					if failure != "pr" {
						wantCreation = append(wantCreation, prRecord)
					}
					if failure != "issue" {
						wantCreation = append(wantCreation, issueRecord)
					}
					if failure != "merge" {
						wantCommits = append(wantCommits, commitRecord)
					}
				}
				if err != nil || !reflect.DeepEqual(creation, wantCreation) || !reflect.DeepEqual(commits, wantCommits) {
					t.Fatalf("failed/retained group emitted warning: creation=%v commits=%v err=%v", creation, commits, err)
				}
				if (failure == "repository" || failure == "all") && saved.Repository.HasIssues != nil {
					t.Fatal("warning path fabricated an unavailable Issues gate")
				}
			})
		}
	}
}
