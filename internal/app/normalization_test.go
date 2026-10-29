package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestExportWorkflowNormalizationIsScopedToAcquiredOptIn(t *testing.T) {
	for _, test := range []struct {
		name    string
		include bool
		failure bool
		raw     string
	}{
		{"requested complete observation", true, false, "FUTURE"},
		{"supported exact observation", true, false, "read"},
		{"unrequested workflow policy", false, false, "FUTURE"},
		{"partial state accompanying failure", true, true, "FUTURE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			observed := github.NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: test.raw, CanApprovePullRequestReviews: false})
			failure := errors.New("complete acquisition denied")
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reads := 0
			reader.WorkflowPermissions = func(repository.Repository) (settings.WorkflowPermissionsState, error) {
				reads++
				if test.failure {
					return observed, failure
				}
				return observed, nil
			}
			var warnings []settings.ObservationNormalization
			saved := false
			err := Export(ExportCheckout{
				Repository:                 checkoutRepository,
				IncludeWorkflowPermissions: test.include,
				NormalizationWarning:       func(normalization settings.ObservationNormalization) { warnings = append(warnings, normalization) },
				PrepareSave: func() (func(settings.Declaration) error, error) {
					return func(declaration settings.Declaration) error {
						saved = true
						ui, err := github.UIFromExportDeclaration(declaration)
						if err != nil {
							t.Fatal(err)
						}
						if test.include && (ui.Settings == nil || ui.Settings.Actions == nil || ui.Settings.Actions.General == nil || ui.Settings.Actions.General.WorkflowPermissions == nil || *ui.Settings.Actions.General.WorkflowPermissions != "read_repository_contents_and_packages_permissions" || ui.Settings.Actions.General.AllowGitHubActionsToCreateAndApprovePullRequests == nil || *ui.Settings.Actions.General.AllowGitHubActionsToCreateAndApprovePullRequests) {
							t.Fatal("fallback or explicit false approval lost", ui)
						}
						return nil
					}, nil
				},
			}, reader)
			var wantErr error
			wantReads, wantWarnings := 0, 0
			if test.include {
				wantReads = 1
				if test.raw == "FUTURE" {
					wantWarnings = 1
				}
			}
			if test.failure {
				wantErr, wantWarnings = failure, 0
			}
			if !errors.Is(err, wantErr) || saved == test.failure || reads != wantReads || len(warnings) != wantWarnings {
				t.Fatalf("export acquisition boundary lost: saved=%v reads=%d warnings=%+v error=%v", saved, reads, warnings, err)
			}
			if wantWarnings == 1 && warnings[0] != observed.Normalization {
				t.Fatal("warning lost raw provenance", warnings)
			}
		})
	}
}

func TestExportCommitMenuWarningsRetainVisibleRawSidecars(t *testing.T) {
	for _, test := range []struct {
		name, failure string
		normalized    bool
		squash, merge bool
		warnings      []settings.CommitMessageMenu
	}{
		{"both visible", "", true, true, true, []settings.CommitMessageMenu{settings.SquashCommitMenu, settings.MergeCommitMenu}},
		{"exact choices", "", false, true, true, nil},
		{"both hidden", "", true, false, false, nil},
		{"squash hidden", "", true, false, true, []settings.CommitMessageMenu{settings.MergeCommitMenu}},
		{"merge hidden", "", true, true, false, []settings.CommitMessageMenu{settings.SquashCommitMenu}},
		{"partial repository acquisition", "repository", true, true, true, nil},
		{"later acquisition fails", "topics", true, true, true, []settings.CommitMessageMenu{settings.SquashCommitMenu, settings.MergeCommitMenu}},
		{"later UI conversion fails", "conversion", true, true, true, []settings.CommitMessageMenu{settings.SquashCommitMenu, settings.MergeCommitMenu}},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := settings.MergeMethodsState{AllowSquashMerge: test.squash, AllowMergeCommit: test.merge, SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}
			if test.normalized {
				source.SquashCommitNormalization = settings.CommitMessageNormalization{Path: "repository.settings.general.pull_requests.squash_merge.default_commit_message", RawTitle: "COMMIT_OR_PR_TITLE", RawMessage: "BLANK", Fallback: "default_message"}
				source.MergeCommitNormalization = settings.CommitMessageNormalization{Path: "repository.settings.general.pull_requests.merge_commit.default_commit_message", RawTitle: "MERGE_MESSAGE", RawMessage: "BLANK", Fallback: "default_message"}
			}
			observed := settings.NewRepositoryObservation(&settings.RepositoryState{DefaultBranch: "main"}, &source, &settings.PRAccessState{PullRequestCreationPolicy: "ALL"}).WithIssueCreationPolicy(settings.IssueCreationPolicyState{Policy: "ALL"}).WithSponsorships(settings.SponsorshipsState{Enabled: false})
			cause := errors.New("acquisition failed")
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(stage string, _ repository.Repository) error {
				if test.failure == stage {
					return cause
				}
				return nil
			})
			reader.ObserveRepository = func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
				if test.failure == "repository" {
					return observed, cause
				}
				return observed, nil
			}
			reader.WorkflowPermissions = func(repository.Repository) (settings.WorkflowPermissionsState, error) {
				t.Fatal("unrequested workflow policy acquired")
				return settings.WorkflowPermissionsState{}, nil
			}
			reader.Rulesets = func(repository.Repository) (settings.RulesetObservation, error) {
				if test.failure != "conversion" {
					t.Fatal("unrequested Rulesets acquired")
				}
				state := appRulesetState(t)
				mode := "always"
				actors := []settings.BypassActor{{ActorType: "OrganizationAdmin", BypassMode: &mode}}
				state.Policy.BypassActors = &actors
				return settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
			}
			var warnings []settings.CommitMessageNormalization
			saved, scalarWarnings := false, 0
			err := Export(ExportCheckout{
				Repository: func() (repository.Repository, error) {
					return repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, nil
				},
				IncludeRulesets:                   test.failure == "conversion",
				NormalizationWarning:              func(settings.ObservationNormalization) { scalarWarnings++ },
				CommitMessageNormalizationWarning: func(record settings.CommitMessageNormalization) { warnings = append(warnings, record) },
				PrepareSave: func() (func(settings.Declaration) error, error) {
					return func(declaration settings.Declaration) error {
						ui, err := github.UIFromExportDeclaration(declaration)
						if err != nil {
							return err
						}
						pulls := ui.Settings.General.PullRequests
						if (pulls.SquashMerge.DefaultCommitMessage != nil) != test.squash || (pulls.MergeCommit.DefaultCommitMessage != nil) != test.merge {
							t.Fatal("hidden menu became an exported choice", ui)
						}
						if test.squash && *pulls.SquashMerge.DefaultCommitMessage != "default_message" || test.merge && *pulls.MergeCommit.DefaultCommitMessage != "default_message" {
							t.Fatal("effective export choice lost", ui)
						}
						saved = true
						return nil
					}, nil
				},
			}, reader)
			var expected []settings.CommitMessageNormalization
			for _, menu := range test.warnings {
				if menu == settings.SquashCommitMenu {
					expected = append(expected, source.SquashCommitNormalization)
				} else {
					expected = append(expected, source.MergeCommitNormalization)
				}
			}
			if (err != nil) != (test.failure != "") || saved != (test.failure == "") || scalarWarnings != 0 || !reflect.DeepEqual(warnings, expected) {
				t.Fatalf("export sidecar scope/later failure lost: saved=%v warnings=%+v error=%v", saved, warnings, err)
			}
			if (test.failure == "repository" || test.failure == "topics") && !errors.Is(err, cause) {
				t.Fatal("original acquisition error lost", err)
			}
			if test.failure == "conversion" && (err == nil || !strings.Contains(err.Error(), "export Rulesets")) {
				t.Fatal("fixture did not reach actual UI conversion failure", err)
			}
		})
	}
}

func TestPlanRetainsNormalizedCommitMenuDecisions(t *testing.T) {
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, nil
	}
	for _, test := range []struct {
		menu           settings.CommitMessageMenu
		title, message string
		fields         []string
	}{
		{settings.SquashCommitMenu, "COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES", []string{"repository.squash_merge_commit_title", "repository.squash_merge_commit_message"}},
		{settings.MergeCommitMenu, "MERGE_MESSAGE", "PR_TITLE", []string{"repository.merge_commit_title", "repository.merge_commit_message"}},
	} {
		state := settings.MergeMethodsState{AllowMergeCommit: true, AllowSquashMerge: true, SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE", SquashCommitNormalization: settings.CommitMessageNormalization{Path: "repository.settings.general.pull_requests.squash_merge.default_commit_message", RawTitle: "COMMIT_OR_PR_TITLE", RawMessage: "BLANK", Fallback: "default_message"}, MergeCommitNormalization: settings.CommitMessageNormalization{Path: "repository.settings.general.pull_requests.merge_commit.default_commit_message", RawTitle: "MERGE_MESSAGE", RawMessage: "BLANK", Fallback: "default_message"}}
		var desired settings.Declaration
		if test.menu == settings.SquashCommitMenu {
			desired.Repository.SquashMergeCommitTitle, desired.Repository.SquashMergeCommitMessage = &test.title, &test.message
		} else {
			desired.Repository.MergeCommitTitle, desired.Repository.MergeCommitMessage = &test.title, &test.message
		}
		var requested []string
		read := Reader{ObserveRepository: func(_ repository.Repository, fields []settings.FieldID) (settings.RepositoryObservation, error) {
			for _, field := range fields {
				requested = append(requested, field.Path())
			}
			return settings.NewRepositoryObservation(nil, &state, nil), nil
		}}
		result, err := Plan(func() (settings.Declaration, error) { return desired, nil }, resolve, read)
		if err != nil || result.Comparison.Managed != 2 || len(result.Comparison.CommitMessages) != 1 || len(result.Comparison.Changes) != 0 || result.Comparison.CommitMessages[0].Menu() != test.menu || !reflect.DeepEqual(requested, test.fields) {
			t.Fatal("plan dropped normalized menu and falsely exposed zero changes", result, requested, err)
		}
		no := false
		desired.Repository = settings.RepositoryDeclaration{AllowSquashMerge: &no}
		result, err = Plan(func() (settings.Declaration, error) { return desired, nil }, resolve, read)
		if err != nil || result.Comparison.Managed != 1 || len(result.Comparison.CommitMessages) != 0 || len(result.Comparison.Changes) != 1 {
			t.Fatal("unmanaged menu entered plan output", result, err)
		}
	}
}
