package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentPlanAndApplyNameOwnershipWithoutIDWriteback(t *testing.T) {
	d, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "new"}, {Name: "production"}})
	if err != nil {
		t.Fatal(err)
	}
	load := func() (settings.Declaration, error) { return settings.Declaration{Environments: &d}, nil }
	repo := repository.Repository{Owner: "owner", Name: "current"}
	resolve := func() (repository.Repository, error) { return repo, nil }
	empty, _ := settings.NewEnvironmentObservation(nil)
	read := func(target repository.Repository, names []string) (settings.EnvironmentObservation, error) {
		if target != repo || !reflect.DeepEqual(names, []string{"new", "production"}) {
			t.Fatal(target, names)
		}
		return empty, nil
	}
	plan, err := Plan(load, resolve, Reader{Environments: read})
	if err != nil || plan.Comparison.Managed != 1 || len(plan.Comparison.Environments) != 2 {
		t.Fatal(plan, err)
	}
	for _, failure := range []bool{false, true} {
		calls := []string{}
		result, err := Apply(load, resolve, Writer{ReadEnvironments: read, CreatedID: func(string, string, int64) error { t.Fatal("environment numeric ID writeback"); return nil }, Environment: func(target repository.Repository, p settings.EnvironmentDeclaration) (bool, error) {
			calls = append(calls, p.Name)
			if failure && p.Name == "production" {
				return false, errors.New("write uncertainty")
			}
			if *p.WaitTimer != 0 || *p.PreventSelfReview || len(*p.Reviewers) != 0 || p.DeploymentBranchPolicy != nil {
				t.Fatal("lost whole-object defaults", p)
			}
			return true, nil
		}})
		if (err != nil) != failure || !reflect.DeepEqual(calls, []string{"new", "production"}) || len(result.CreatedIDs) != 0 {
			t.Fatal(result, calls, err)
		}
		if failure && (result.Outcome != ApplyPartialFailure || result.Operations[2].Status != OperationFailedUnknown) {
			t.Fatal(result)
		}
		if !failure && !result.Completed() {
			t.Fatal(result)
		}
	}
}

func TestEnvironmentReadFailureAndEmptySelectionDoNotWrite(t *testing.T) {
	for _, count := range []int{0, 1} {
		entries := []settings.EnvironmentDeclaration{}
		if count == 1 {
			entries = append(entries, settings.EnvironmentDeclaration{Name: "production"})
		}
		d, err := settings.NewEnvironmentDeclarations(entries)
		if err != nil {
			t.Fatal(err)
		}
		load := func() (settings.Declaration, error) { return settings.Declaration{Environments: &d}, nil }
		reads := 0
		resolve := func() (repository.Repository, error) {
			if count == 0 {
				t.Fatal("empty selection resolved target")
			}
			return repository.Repository{}, nil
		}
		result, err := Apply(load, resolve, Writer{ReadEnvironments: func(repository.Repository, []string) (settings.EnvironmentObservation, error) {
			reads++
			return settings.EnvironmentObservation{}, errors.New("unreadable protections")
		}, Environment: func(repository.Repository, settings.EnvironmentDeclaration) (bool, error) {
			t.Fatal("wrote after failed read")
			return false, nil
		}})
		if count == 0 {
			if err != nil || reads != 0 || result.Outcome != ApplyUnmanaged {
				t.Fatal(result, err)
			}
		} else if err == nil || reads != 1 || result.Outcome != ApplyFailed || result.Operations[0].Status != OperationFailedRead {
			t.Fatal(result, err)
		}
	}
}

func TestEnvironmentFreshMatchBeforeLaterFailureIsNotClaimedAsWrite(t *testing.T) {
	d, _ := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "a"}, {Name: "b"}})
	empty, _ := settings.NewEnvironmentObservation(nil)
	result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Environments: &d}, nil }, func() (repository.Repository, error) { return repository.Repository{}, nil }, Writer{
		ReadEnvironments: func(repository.Repository, []string) (settings.EnvironmentObservation, error) { return empty, nil },
		Environment: func(_ repository.Repository, p settings.EnvironmentDeclaration) (bool, error) {
			if p.Name == "a" {
				return false, nil
			}
			return false, errors.New("later uncertain write")
		},
	})
	if err == nil || result.Outcome != ApplyFailed || result.Operations[1].Status != OperationUnchanged {
		t.Fatal(result, err)
	}
}

func TestEnvironmentExportIsOptInAndReadFailureDoesNotSave(t *testing.T) {
	for _, mode := range []string{"ordinary", "selected", "failure"} {
		reads, saves := 0, 0
		reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
		d, _ := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "production"}})
		reader.Environments = func(_ repository.Repository, names []string) (settings.EnvironmentObservation, error) {
			reads++
			if names != nil {
				t.Fatal("export did not select all environments", names)
			}
			if mode == "failure" {
				return settings.EnvironmentObservation{}, errors.New("unsupported policy")
			}
			return settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 71, Policy: d.Values()[0]}})
		}
		err := Export(ExportCheckout{IncludeEnvironments: mode != "ordinary", Repository: func() (repository.Repository, error) { return repository.Repository{}, nil }, PrepareSave: func() (func(settings.Declaration) error, error) {
			return func(saved settings.Declaration) error {
				saves++
				if mode == "selected" && !reflect.DeepEqual(saved.Environments, &d) {
					t.Fatal(saved.Environments)
				}
				if mode == "ordinary" && saved.Environments != nil {
					t.Fatal("ordinary export included environments")
				}
				return nil
			}, nil
		}}, reader)
		if mode == "ordinary" {
			if err != nil || reads != 0 || saves != 1 {
				t.Fatal(reads, saves, err)
			}
		} else if mode == "failure" {
			if err == nil || reads != 1 || saves != 0 {
				t.Fatal(reads, saves, err)
			}
		} else if err != nil || reads != 1 || saves != 1 {
			t.Fatal(reads, saves, err)
		}
	}
}
