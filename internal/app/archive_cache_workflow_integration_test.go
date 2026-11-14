package app

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Neither independently selected family can substitute for the other's evidence.
func TestArchivedWorkflowAndCacheRequireBothMatches(t *testing.T) {
	for _, scenario := range []string{"matching", "workflow differs", "workflow unavailable", "cache differs", "cache unavailable", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			yes, no, permission, days := true, false, "read", 30
			desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, Actions: settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &permission, CanApprovePullRequestReviews: &no, CacheRetentionDays: &days}}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			archives, workflowReads, cacheReads, writes := 0, 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archives++
					id := int64(1)
					if scenario == "identity drift" && archives == 2 {
						id = 2
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
				},
				ReadWorkflowPermissions: func(repository.Repository) (settings.WorkflowPermissionsState, error) {
					workflowReads++
					if scenario == "workflow unavailable" {
						return settings.WorkflowPermissionsState{}, nil
					}
					value := "read"
					if scenario == "workflow differs" {
						value = "write"
					}
					return settings.WorkflowPermissionsState{DefaultWorkflowPermissions: value}, nil
				},
				ReadCacheRetention: func(repository.Repository) (settings.CacheRetentionObservation, error) {
					cacheReads++
					if scenario == "cache unavailable" {
						return settings.CacheRetentionObservation{}, nil
					}
					value := 30
					if scenario == "cache differs" {
						value = 31
					}
					return settings.NewCacheRetentionObservation(value, "public")
				},
				WorkflowPermissions: func(repository.Repository, settings.ActionsPolicyDeclaration) (bool, error) {
					writes++
					return false, nil
				},
				CacheRetention:     func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				Archive:            func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID:          func(string, string, int64) error { writes++; return nil },
				BindCollaboratorID: func(string, int64) error { writes++; return nil },
			}
			_, err := Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (scenario == "matching") || writes != 0 {
				t.Fatalf("err=%v writes=%d reads archive=%d cache=%d workflow=%d", err, writes, archives, cacheReads, workflowReads)
			}
			if scenario == "matching" && (archives != 2 || cacheReads != 1 || workflowReads != 1) {
				t.Fatalf("incomplete evidence: archive=%d cache=%d workflow=%d", archives, cacheReads, workflowReads)
			}
			if (scenario == "workflow differs" || scenario == "workflow unavailable") && workflowReads != 1 {
				t.Fatalf("workflow response not reached: %d", workflowReads)
			}
			if (scenario == "cache differs" || scenario == "cache unavailable") && cacheReads != 1 {
				t.Fatalf("retention response not reached: %d", cacheReads)
			}
		})
	}
}
