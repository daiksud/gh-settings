package app

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedCacheAndArtifactRetentionRequireBothMatches(t *testing.T) {
	for _, scenario := range []string{"matching", "cache differs", "artifact differs", "artifact unavailable", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			yes, days := true, 30
			desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, Actions: settings.ActionsPolicyDeclaration{CacheRetentionDays: &days, RetentionDays: &days}}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			archiveReads, artifactReads, cacheReads, writes := 0, 0, 0, 0
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archiveReads++
					id := int64(1)
					if scenario == "identity drift" && archiveReads == 2 {
						id = 2
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
				},
				ReadRetention: func(repository.Repository) (settings.RetentionObservation, error) {
					artifactReads++
					if scenario == "artifact unavailable" {
						return settings.RetentionObservation{}, nil
					}
					value := 30
					if scenario == "artifact differs" {
						value = 31
					}
					return settings.NewRetentionObservation(value, 90)
				},
				ReadCacheRetention: func(repository.Repository) (settings.CacheRetentionObservation, error) {
					cacheReads++
					value := 30
					if scenario == "cache differs" {
						value = 31
					}
					return settings.NewCacheRetentionObservation(value, "public")
				},
				Retention:          func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				CacheRetention:     func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				Archive:            func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID:          func(string, string, int64) error { writes++; return nil },
				BindCollaboratorID: func(string, int64) error { writes++; return nil },
			}
			_, err := Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) { return repo, nil }, writer)
			if (err == nil) != (scenario == "matching") || writes != 0 {
				t.Fatalf("err=%v writes=%d reads=%d/%d/%d", err, writes, archiveReads, artifactReads, cacheReads)
			}
			if scenario == "matching" && (archiveReads != 2 || artifactReads != 1 || cacheReads != 1) {
				t.Fatalf("missing matching evidence: archive=%d artifact=%d cache=%d", archiveReads, artifactReads, cacheReads)
			}
		})
	}
}
