package app

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedCacheAndEnvironmentRequireBothMatches(t *testing.T) {
	for _, scenario := range []string{"matching", "cache differs", "environment differs", "environment unavailable", "environment identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			yes, no, days, zero := true, false, 30, 0
			envs, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "production", WaitTimer: &zero}})
			if err != nil {
				t.Fatal(err)
			}
			desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, Environments: &envs, Actions: settings.ActionsPolicyDeclaration{CacheRetentionDays: &days}}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			archiveReads, envReads, cacheReads, writes := 0, 0, 0, 0
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archiveReads++
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
				},
				ReadEnvironments: func(_ repository.Repository, names []string) (settings.EnvironmentObservation, error) {
					envReads++
					if len(names) != 1 || names[0] != "production" {
						t.Fatalf("unexpected selection: %v", names)
					}
					if scenario == "environment unavailable" {
						return settings.EnvironmentObservation{}, nil
					}
					timer, id := 0, int64(71)
					if scenario == "environment differs" {
						timer = 1
					}
					if scenario == "environment identity drift" && envReads == 2 {
						id = 72
					}
					reviewers := []settings.EnvironmentReviewer{}
					return settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: id, CustomRules: []settings.EnvironmentCustomRule{}, Policy: settings.EnvironmentDeclaration{Name: "production", WaitTimer: &timer, PreventSelfReview: &no, CanAdminsBypass: &no, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}}})
				},
				ReadCacheRetention: func(repository.Repository) (settings.CacheRetentionObservation, error) {
					cacheReads++
					value := 30
					if scenario == "cache differs" {
						value = 31
					}
					return settings.NewCacheRetentionObservation(value, "public")
				},
				Environment: func(repository.Repository, settings.EnvironmentDeclaration) (bool, error) {
					writes++
					return false, nil
				},
				CacheRetention:     func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				Archive:            func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID:          func(string, string, int64) error { writes++; return nil },
				BindCollaboratorID: func(string, int64) error { writes++; return nil },
			}
			_, err = Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) { return repo, nil }, writer)
			if (err == nil) != (scenario == "matching") || writes != 0 {
				t.Fatalf("err=%v writes=%d reads=%d/%d/%d", err, writes, archiveReads, envReads, cacheReads)
			}
			if scenario == "matching" && (archiveReads != 2 || envReads != 2 || cacheReads != 1) {
				t.Fatalf("missing matching evidence: archive=%d environment=%d cache=%d", archiveReads, envReads, cacheReads)
			}
		})
	}
}
