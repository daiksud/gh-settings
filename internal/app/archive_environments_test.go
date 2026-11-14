package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestArchivedEnvironmentsVerifySelectedOwnershipAndIdentity(t *testing.T) {
	for _, scenario := range []string{"matching", "name only", "omitted companions", "omitted unknown mode", "omitted unknown pattern", "two matching", "missing", "timer differs", "pattern differs", "app differs", "mode normalized", "pattern normalized", "first denied", "second denied", "incomplete", "environment identity changed", "repository identity changed", "prune"} {
		t.Run(scenario, func(t *testing.T) {
			zero, off := 0, false
			reviewers := []settings.EnvironmentReviewer{}
			apps := []int64{9}
			patterns := []settings.EnvironmentPattern{{Type: "branch", Name: "main"}}
			mode := &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
			want := settings.EnvironmentDeclaration{Name: "Production", WaitTimer: &zero, CustomRules: &apps, DeploymentBranchPolicy: mode, DeploymentBranchPatterns: &patterns}
			if scenario == "name only" {
				want = settings.EnvironmentDeclaration{Name: "Production"}
			}
			if scenario == "omitted companions" || scenario == "omitted unknown mode" || scenario == "omitted unknown pattern" {
				want = settings.EnvironmentDeclaration{Name: "Production", WaitTimer: &zero}
			}
			entries := []settings.EnvironmentDeclaration{want}
			if scenario == "two matching" {
				entries = append(entries, settings.EnvironmentDeclaration{Name: "Staging"})
			}
			desired, err := settings.NewEnvironmentDeclarations(entries)
			if err != nil {
				t.Fatal(err)
			}
			archived := true
			declaration := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, Environments: &desired}
			repo := repository.Repository{Owner: "owner", Name: "current"}
			repoReads, envReads, writes, local := 0, 0, 0, 0
			read := func(_ repository.Repository, names []string) (settings.EnvironmentObservation, error) {
				envReads++
				if len(names) != len(entries) {
					t.Fatalf("unexpected selection: %v", names)
				}
				if scenario == "first denied" || scenario == "second denied" && envReads == 2 {
					return settings.EnvironmentObservation{}, errors.New("denied")
				}
				if scenario == "incomplete" {
					return settings.EnvironmentObservation{}, nil
				}
				if scenario == "missing" {
					return settings.NewEnvironmentObservation(nil)
				}
				timer := 0
				if scenario == "timer differs" {
					timer = 1
				}
				state := settings.EnvironmentState{ID: 71, CustomRules: []settings.EnvironmentCustomRule{{ID: 10, AppID: 9, NodeID: "rule", AppNodeID: "app", Slug: "guard", IntegrationURL: "https://api.github.com/apps/guard"}}, Policy: settings.EnvironmentDeclaration{Name: "production", WaitTimer: &timer, PreventSelfReview: &off, CanAdminsBypass: &off, Reviewers: &reviewers, DeploymentBranchPolicy: mode, DeploymentBranchPatterns: &patterns}, Patterns: []settings.EnvironmentPatternState{{ID: 17, Pattern: patterns[0]}}}
				if scenario == "pattern differs" {
					different := []settings.EnvironmentPattern{{Type: "branch", Name: "release"}}
					state.Policy.DeploymentBranchPatterns = &different
					state.Patterns[0].Pattern = different[0]
				}
				if scenario == "app differs" {
					state.CustomRules = []settings.EnvironmentCustomRule{}
				}
				if scenario == "environment identity changed" && envReads == 2 {
					state.ID = 72
				}
				if scenario == "mode normalized" || scenario == "omitted unknown mode" {
					state.Policy.DeploymentBranchPolicy = nil
					state.Policy.DeploymentBranchPolicyDeclared = true
					state.Policy.DeploymentBranchPatterns = nil
					state.DeploymentModeNormalization = settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: settings.EnvironmentBranchPolicy{ProtectedBranches: true, CustomBranchPolicies: true}, Fallback: uimodel.NoRestriction}
				}
				if scenario == "pattern normalized" || scenario == "omitted unknown pattern" {
					state.Patterns[0].RefTypeNormalization = settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "FUTURE", Fallback: "branch"}
				}
				states := []settings.EnvironmentState{state}
				if scenario == "two matching" {
					states = append(states, settings.EnvironmentState{ID: 73, CustomRules: []settings.EnvironmentCustomRule{}, Policy: settings.EnvironmentDeclaration{Name: "staging", WaitTimer: &zero, PreventSelfReview: &off, CanAdminsBypass: &off, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}})
				}
				return settings.NewEnvironmentObservation(states)
			}
			writer := Writer{ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
				repoReads++
				id := int64(1)
				if scenario == "repository identity changed" && repoReads == 2 {
					id = 2
				}
				return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
			}, ReadEnvironments: read, Environment: func(repository.Repository, settings.EnvironmentDeclaration) (bool, error) { writes++; return true, nil }, Archive: func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil }, CreatedID: func(string, string, int64) error { local++; return nil }}
			writer.ReadEnvironmentPrune = func(repository.Repository, []string) (settings.EnvironmentPruneObservation, error) {
				return environmentPruneFixture(t, true), nil
			}
			var prune settings.PruneScopes
			if scenario == "prune" {
				prune, _ = settings.ParsePruneScopes("environments")
			}
			result, err := Apply(func() (settings.Declaration, error) { return declaration, nil }, func() (repository.Repository, error) { return repo, nil }, writer, prune)
			matching := scenario == "matching" || scenario == "name only" || scenario == "omitted companions" || scenario == "omitted unknown mode" || scenario == "omitted unknown pattern" || scenario == "two matching"
			if (err == nil) != matching || writes != 0 || local != 0 {
				t.Fatalf("outcome=%v err=%v remote=%d local=%d", result.Outcome, err, writes, local)
			}
			if matching && (envReads != 2 || repoReads != 2 || result.Outcome != ApplyMatchingLabels) {
				t.Fatalf("outcome=%v envReads=%d repoReads=%d", result.Outcome, envReads, repoReads)
			}
		})
	}
}
