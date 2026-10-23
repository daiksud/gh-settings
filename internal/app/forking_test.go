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
	_, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{AllowForking: &v, Archived: &v}}, nil
	}, func() (repository.Repository, error) { t.Fatal("resolved invalid composition"); return repo, nil }, Writer{})
	if err == nil {
		t.Fatal("accepted archive composition")
	}
}
