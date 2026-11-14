package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedCacheSelectedUnits(t *testing.T) {
	for _, scenario := range []string{"matching", "retention only", "storage only", "retention differs", "storage differs", "retention unavailable", "storage unavailable", "retention denied", "storage denied", "public limit", "prune", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			yes, days, gb := true, 30, 10
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, Actions: settings.ActionsPolicyDeclaration{CacheRetentionDays: &days, CacheSizeGB: &gb}}
			if scenario == "retention only" {
				d.Actions.CacheSizeGB = nil
			}
			if scenario == "storage only" {
				d.Actions.CacheRetentionDays = nil
			}
			if scenario == "public limit" {
				days = 91
			}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			archiveReads, retentionReads, storageReads, writes := 0, 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archiveReads++
					id := int64(1)
					if scenario == "identity drift" && archiveReads == 2 {
						id = 2
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
				},
				ReadCacheRetention: func(repository.Repository) (settings.CacheRetentionObservation, error) {
					retentionReads++
					if scenario == "retention denied" {
						return settings.CacheRetentionObservation{}, errors.New("denied")
					}
					if scenario == "retention unavailable" {
						return settings.CacheRetentionObservation{}, nil
					}
					v := 30
					if scenario == "retention differs" {
						v = 31
					}
					return settings.NewCacheRetentionObservation(v, "public")
				},
				ReadCacheStorage: func(repository.Repository) (settings.CacheStorageObservation, error) {
					storageReads++
					if scenario == "storage denied" {
						return settings.CacheStorageObservation{}, errors.New("denied")
					}
					if scenario == "storage unavailable" {
						return settings.CacheStorageObservation{}, nil
					}
					v := 10
					if scenario == "storage differs" {
						v = 11
					}
					return settings.NewCacheStorageObservation(v)
				},
				CacheRetention:     func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				CacheStorage:       func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				Archive:            func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID:          func(string, string, int64) error { writes++; return nil },
				BindCollaboratorID: func(string, int64) error { writes++; return nil },
			}
			var prune settings.PruneScopes
			var err error
			if scenario == "prune" {
				prune, err = settings.ParsePruneScopes("labels")
				if err != nil {
					t.Fatal(err)
				}
			}
			_, err = Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w, prune)
			want := scenario == "matching" || scenario == "retention only" || scenario == "storage only"
			if (err == nil) != want || writes != 0 {
				t.Fatalf("err=%v writes=%d", err, writes)
			}
			if scenario == "retention only" && storageReads != 0 || scenario == "storage only" && retentionReads != 0 {
				t.Fatalf("omitted read: %d %d", retentionReads, storageReads)
			}
			if scenario == "matching" && (archiveReads != 2 || retentionReads != 1 || storageReads != 1) {
				t.Fatalf("reads: %d %d %d", archiveReads, retentionReads, storageReads)
			}
		})
	}
}

func TestArchivedCacheAndOIDCRequireAllSelectedMatches(t *testing.T) {
	for _, scenario := range []string{"matching", "OIDC differs", "cache differs"} {
		t.Run(scenario, func(t *testing.T) {
			yes, no, days := true, false, 30
			desiredOIDC := settings.OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}
			immutable := no
			if scenario == "OIDC differs" {
				immutable = yes
			}
			observed, err := settings.NewOIDCSubjectObservation(settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &immutable})
			if err != nil {
				t.Fatal(err)
			}
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, OIDCSubject: &desiredOIDC, Actions: settings.ActionsPolicyDeclaration{CacheRetentionDays: &days}}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			writes, oidcReads, cacheReads := 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
				},
				ReadArchivedOIDCSubject: func(repository.Repository, settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error) {
					oidcReads++
					return [2]settings.OIDCSubjectObservation{observed, observed}, nil
				},
				ReadCacheRetention: func(repository.Repository) (settings.CacheRetentionObservation, error) {
					cacheReads++
					v := 30
					if scenario == "cache differs" {
						v = 31
					}
					return settings.NewCacheRetentionObservation(v, "public")
				},
				OIDCSubject:    func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error) { writes++; return false, nil },
				CacheRetention: func(repository.Repository, int) (bool, error) { writes++; return false, nil },
			}
			_, err = Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (scenario == "matching") || writes != 0 || oidcReads != 1 || scenario == "matching" && cacheReads != 1 {
				t.Fatalf("err=%v writes=%d reads=%d/%d", err, writes, oidcReads, cacheReads)
			}
		})
	}
}
