package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestForkingSelectivePlanAndMissingObservation(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, v := range []bool{false, true} {
		d := settings.Declaration{Repository: settings.RepositoryDeclaration{AllowForking: &v}}
		result, err := Plan(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, Reader{Forking: func(repository.Repository) (settings.ForkingObservation, error) {
			return settings.NewForkingObservation(repo, 42, 7, "owner/current", !v)
		}})
		if err != nil || result.Comparison.Managed != 1 || len(result.Comparison.Changes) != 1 {
			t.Fatal(result, err)
		}
		result, err = Plan(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, Reader{Forking: func(repository.Repository) (settings.ForkingObservation, error) {
			return settings.ForkingObservation{}, nil
		}})
		if err == nil || result.Comparison.Managed != 0 {
			t.Fatal("missing became false", result, err)
		}
	}
}

type forkingReadError struct{ error }

func (*forkingReadError) ReadFailedBeforeWrite() {}
func TestForkingApplyProgressAndLocalComposition(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	v := false
	desc := "changed"
	for _, earlier := range []bool{false, true} {
		for _, failure := range []string{"read", "write", "match", "updated"} {
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{AllowForking: &v}}
			if earlier {
				d.Repository.Description = &desc
			}
			ordinary, forking := 0, 0
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, Writer{Repository: func(_ repository.Repository, got settings.RepositoryDeclaration) error {
				ordinary++
				if got.AllowForking != nil {
					t.Fatal("forking leaked into bulk write")
				}
				return nil
			}, Forking: func(_ repository.Repository, got bool) (bool, error) {
				forking++
				if got {
					t.Fatal("false lost")
				}
				switch failure {
				case "read":
					return false, &forkingReadError{errors.New("denied")}
				case "write":
					return false, errors.New("uncertain")
				case "updated":
					return true, nil
				default:
					return false, nil
				}
			}})
			if forking != 1 || (ordinary == 1) != earlier {
				t.Fatal(ordinary, forking)
			}
			if failure == "read" || failure == "write" {
				want := ApplyFailed
				if earlier {
					want = ApplyPartialFailure
				}
				if err == nil || result.Outcome != want || !strings.Contains(err.Error(), "earlier successful writes remain applied") {
					t.Fatal(result, err)
				}
				status := OperationFailedUnknown
				if failure == "read" {
					status = OperationFailedRead
				}
				if result.Operations[len(result.Operations)-1].Status != status {
					t.Fatal(result)
				}
			} else if err != nil || !result.Completed() {
				t.Fatal(result, err)
			}
		}
	}
	archived := true
	_, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{AllowForking: &v, Archived: &archived}}, nil
	}, func() (repository.Repository, error) { t.Fatal("resolved invalid composition"); return repo, nil }, Writer{})
	if err == nil {
		t.Fatal("accepted archive composition")
	}
}

func TestForkingExplicitUnarchivePlanAndApplyOrder(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	active, desiredForking := false, false
	d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &active, AllowForking: &desiredForking}}
	plan, err := Plan(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, Reader{
		Archive: func(repository.Repository) (settings.ArchiveState, error) {
			return settings.ArchiveState{ID: 42, FullName: "owner/current", Archived: true}, nil
		},
		Forking: func(repository.Repository) (settings.ForkingObservation, error) {
			return settings.NewForkingObservation(repo, 42, 7, "owner/current", true)
		},
	})
	if err != nil || plan.Comparison.Managed != 2 || len(plan.Comparison.Changes) != 2 || plan.Comparison.Changes[0].Field().Path() != "repository.archived" || plan.Comparison.Changes[1].Field().Path() != "repository.allow_forking" {
		t.Fatal(plan, err)
	}
	events := []string{}
	state := settings.ArchiveState{ID: 42, FullName: "owner/current", Archived: true}
	result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, Writer{
		ReadArchive: func(repository.Repository) (settings.ArchiveState, error) { return state, nil },
		Archive: func(_ repository.Repository, _ settings.ArchiveState, value bool) error {
			events = append(events, "unarchive")
			state.Archived = value
			return nil
		},
		Forking: func(_ repository.Repository, value bool) (bool, error) {
			if state.Archived || value {
				t.Fatal("forking update ran before unarchive or lost false", state, value)
			}
			events = append(events, "forking")
			return true, nil
		},
	})
	if err != nil || !result.Completed() || strings.Join(events, ",") != "unarchive,forking" {
		t.Fatal(events, result, err)
	}
}

func TestForkingExplicitUnarchiveFailureProgress(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	active, desiredForking := false, false
	d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &active, AllowForking: &desiredForking}}
	for _, stage := range []string{"unarchive", "forking"} {
		t.Run(stage, func(t *testing.T) {
			forkingCalls := 0
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					return settings.ArchiveState{ID: 42, FullName: "owner/current", Archived: true}, nil
				},
				Archive: func(repository.Repository, settings.ArchiveState, bool) error {
					if stage == "unarchive" {
						return errors.New("unarchive denied")
					}
					return nil
				},
				Forking: func(repository.Repository, bool) (bool, error) {
					forkingCalls++
					return false, &forkingReadError{errors.New("fork policy denied")}
				},
			})
			if err == nil {
				t.Fatal(result)
			}
			if stage == "unarchive" {
				if result.Outcome != ApplyFailed || forkingCalls != 0 || strings.Contains(err.Error(), "repository was unarchived") {
					t.Fatal(result, err, forkingCalls)
				}
				return
			}
			if result.Outcome != ApplyPartialFailure || forkingCalls != 1 || !strings.Contains(err.Error(), "repository was unarchived") || !strings.Contains(err.Error(), "fork policy denied") {
				t.Fatal(result, err, forkingCalls)
			}
		})
	}
}
