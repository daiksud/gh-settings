package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedRunnersApplyReadOnly(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, mode := range []string{"match", "empty custom labels", "empty outer", "omitted", "different", "unavailable", "no reader", "identity drift", "mixed", "mixed differs", "prune"} {
		t.Run(mode, func(t *testing.T) {
			labels := []string{"gpu"}
			if mode == "empty custom labels" {
				labels = []string{}
			}
			values := []settings.RunnerLabels{{ID: 23, CustomLabels: &labels}}
			if mode == "empty outer" {
				values = []settings.RunnerLabels{}
			}
			yes := true
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, Runners: &values}
			if mode == "omitted" {
				d.Runners = nil
			}
			if mode == "mixed" || mode == "mixed differs" {
				d.Topics.Names = &[]string{"one"}
			}
			calls, archives, writes := 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archives++
					id := int64(1)
					if mode == "identity drift" && archives == 2 {
						id = 2
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
				},
				VerifyArchivedRunners: func(_ repository.Repository, ds []settings.RunnerLabels, a settings.ArchiveState) error {
					calls++
					if a.ID != 1 || len(ds) != 1 || ds[0].ID != 23 {
						t.Fatal("incorrect selection")
					}
					if mode == "different" || mode == "unavailable" {
						return errors.New("observation differs")
					}
					return nil
				},
				ReadTopics: func(repository.Repository) ([]string, error) {
					if mode == "mixed differs" {
						return []string{"other"}, nil
					}
					return []string{"one"}, nil
				},
				ReadRunnerPrune: func(repository.Repository) (settings.RunnerPruneObservation, error) {
					return settings.RunnerPruneObservation{}, errors.New("prune unavailable")
				},
				RunnerLabels: func(repository.Repository, settings.RunnerLabels) (bool, error) { writes++; return true, nil },
				Archive:      func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
			}
			if mode == "no reader" {
				w.VerifyArchivedRunners = nil
			}
			var prune settings.PruneScopes
			if mode == "prune" {
				prune, _ = settings.ParsePruneScopes("runners")
			}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w, prune)
			success := mode == "match" || mode == "empty custom labels" || mode == "empty outer" || mode == "omitted" || mode == "mixed"
			if (err == nil) != success || writes != 0 || len(result.CreatedIDs) != 0 {
				t.Fatalf("err=%v writes=%d", err, writes)
			}
			if (mode == "empty outer" || mode == "omitted" || mode == "mixed differs" || mode == "prune") && calls != 0 {
				t.Fatalf("unexpected reads: %d", calls)
			}
		})
	}
}
