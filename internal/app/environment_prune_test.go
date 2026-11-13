package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func environmentPruneFixture(t *testing.T, archived bool, names ...string) settings.EnvironmentPruneObservation {
	t.Helper()
	repo := repository.Repository{Owner: "owner", Name: "current"}
	target := settings.ArchiveState{ID: 71, FullName: "owner/current", Archived: archived}
	entries := []settings.EnvironmentPruneCandidate{}
	for i, name := range names {
		timer, flag, reviewers := 0, false, []settings.EnvironmentReviewer{}
		variables, _ := settings.NewVariableInventory(nil)
		candidate, err := settings.NewEnvironmentPruneCandidate(repo, target, settings.EnvironmentState{ID: int64(i + 1), Policy: settings.EnvironmentDeclaration{Name: name, WaitTimer: &timer, PreventSelfReview: &flag, CanAdminsBypass: &flag, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}}, variables, []settings.EnvironmentSecretMetadata{}, true)
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, candidate)
	}
	observed, err := settings.NewEnvironmentPruneObservation(repo, target, entries)
	if err != nil {
		t.Fatal(err)
	}
	return observed
}

func TestEnvironmentPruneOrderingAndFailureEffects(t *testing.T) {
	for _, mode := range []string{"success", "global-denied", "candidate-drift", "policy-failure", "variable-failure", "first-delete-read-failure", "second-delete-unknown", "unarchive"} {
		t.Run(mode, func(t *testing.T) {
			repo := repository.Repository{Owner: "owner", Name: "current"}
			events := []string{}
			reads := 0
			policies, _ := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "kept-policy"}})
			variables, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "KEY", Value: "value"}})
			desired := settings.Declaration{Environments: &policies, EnvironmentVariables: []settings.EnvironmentVariableDeclaration{{Environment: "kept-vars", Variables: variables}}}
			if mode == "global-denied" {
				value := "update"
				desired.Repository.Description = &value
			}
			if mode == "unarchive" {
				value := false
				desired.Repository.Archived = &value
			}
			initial := environmentPruneFixture(t, mode == "unarchive", "old-a", "old-b")
			writer := Writer{
				ReadEnvironmentPrune: func(_ repository.Repository, keep []string) (settings.EnvironmentPruneObservation, error) {
					events = append(events, "prune-read")
					reads++
					if !reflect.DeepEqual(keep, []string{"kept-policy", "kept-vars"}) {
						t.Fatal(keep)
					}
					if mode == "global-denied" {
						return settings.EnvironmentPruneObservation{}, errors.New("denied")
					}
					if mode == "candidate-drift" && reads > 1 {
						return environmentPruneFixture(t, false, "new"), nil
					}
					if mode == "unarchive" && reads > 1 {
						return initial.AfterUnarchive(), nil
					}
					return initial, nil
				},
				Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
					t.Fatal("repository write after incomplete preflight")
					return false, nil
				},
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					events = append(events, "archive-read")
					return settings.ArchiveState{ID: 71, FullName: "owner/current", Archived: true}, nil
				},
				Archive: func(repository.Repository, settings.ArchiveState, bool) error {
					events = append(events, "unarchive")
					return nil
				},
				ReadEnvironments: func(repository.Repository, []string) (settings.EnvironmentObservation, error) {
					events = append(events, "policy-read")
					return settings.NewEnvironmentObservation(nil)
				},
				Environment: func(repository.Repository, settings.EnvironmentDeclaration) (bool, error) {
					events = append(events, "policy-write")
					if mode == "policy-failure" {
						return false, errors.New("unknown")
					}
					return true, nil
				},
				EnvironmentVariable: func(repository.Repository, string, settings.RepositoryVariable) (bool, error) {
					events = append(events, "variable-write")
					if mode == "variable-failure" {
						return false, errors.New("unknown")
					}
					return true, nil
				},
				DeleteEnvironment: func(_ repository.Repository, c settings.EnvironmentPruneCandidate) error {
					events = append(events, "delete-"+c.Name())
					if mode == "first-delete-read-failure" {
						return &environmentTestReadFailure{errors.New("read failed")}
					}
					if mode == "second-delete-unknown" && c.Name() == "old-b" {
						return errors.New("unknown")
					}
					if err := c.ValidateTarget(repo, settings.ArchiveState{ID: 71, FullName: "owner/current"}); err != nil {
						t.Fatal(err)
					}
					return nil
				},
			}
			prune, _ := settings.ParsePruneScopes("environments")
			result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) { return repo, nil }, writer, prune)
			if (err == nil) != (mode == "success" || mode == "unarchive") {
				t.Fatal(events, result, err)
			}
			joined := strings.Join(events, ",")
			switch mode {
			case "success":
				if joined != "prune-read,prune-read,policy-read,policy-write,variable-write,delete-old-a,delete-old-b" {
					t.Fatal(events)
				}
			case "global-denied":
				if joined != "prune-read" {
					t.Fatal(events)
				}
			case "candidate-drift":
				if joined != "prune-read,prune-read" {
					t.Fatal(events)
				}
			case "policy-failure", "variable-failure":
				if strings.Contains(joined, "delete-") {
					t.Fatal(events)
				}
			case "first-delete-read-failure":
				if result.Outcome != ApplyPartialFailure || result.Operations[len(result.Operations)-2].Status != OperationFailedRead || result.Operations[len(result.Operations)-1].Status != OperationNotExecuted {
					t.Fatal(result)
				}
			case "second-delete-unknown":
				if result.Outcome != ApplyPartialFailure || result.Operations[len(result.Operations)-1].Status != OperationFailedUnknown {
					t.Fatal(result)
				}
			case "unarchive":
				if !strings.HasPrefix(joined, "prune-read,archive-read,unarchive,prune-read,") {
					t.Fatal(events)
				}
			}
		})
	}
}

func TestEnvironmentPruneEmptyCollectionAndOmittedAll(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	empty, _ := settings.NewEnvironmentDeclarations(nil)
	all, _ := settings.ParsePruneScopes("all")
	for _, present := range []bool{false, true} {
		d := settings.Declaration{}
		if present {
			d.Environments = &empty
		}
		reads := 0
		observation := environmentPruneFixture(t, false)
		result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, Writer{ReadEnvironmentPrune: func(repository.Repository, []string) (settings.EnvironmentPruneObservation, error) {
			reads++
			return observation, nil
		}}, all)
		if err != nil || reads != map[bool]int{false: 0, true: 2}[present] || result.Completed() != present {
			t.Fatal(present, reads, result, err)
		}
		comparison := settings.Comparison{}
		if present {
			if err := planEnvironmentPrune(repo, d, Reader{EnvironmentPrune: func(repository.Repository, []string) (settings.EnvironmentPruneObservation, error) {
				return environmentPruneFixture(t, false, "old"), nil
			}}, &comparison); err != nil || comparison.Managed != 1 || len(comparison.Environments) != 1 {
				t.Fatal(comparison, err)
			}
		}
	}
}

func TestEnvironmentPruneFailureWithoutEarlierWritesAndArchivedPreflight(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	empty, _ := settings.NewEnvironmentDeclarations(nil)
	scope, _ := settings.ParsePruneScopes("environments")
	for _, archived := range []bool{false, true} {
		declaration := settings.Declaration{Environments: &empty}
		if archived {
			value := "must not write"
			declaration.Repository.Description = &value
		}
		reads, deletes := 0, 0
		observed := environmentPruneFixture(t, archived, "a", "b")
		result, err := Apply(func() (settings.Declaration, error) { return declaration, nil }, func() (repository.Repository, error) { return repo, nil }, Writer{
			ReadEnvironmentPrune: func(repository.Repository, []string) (settings.EnvironmentPruneObservation, error) {
				reads++
				return observed, nil
			},
			Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
				t.Fatal("write after archived preflight")
				return true, nil
			},
			DeleteEnvironment: func(repository.Repository, settings.EnvironmentPruneCandidate) error {
				deletes++
				return errors.New("unconfirmed delete")
			},
		}, scope)
		if err == nil || result.Outcome != ApplyFailed || strings.Contains(err.Error(), "earlier resource writes") {
			t.Fatal(result, err)
		}
		if archived {
			if reads != 1 || deletes != 0 {
				t.Fatal(reads, deletes)
			}
		} else if deletes != 1 || result.Operations[1].Status != OperationFailedUnknown || result.Operations[2].Status != OperationNotExecuted {
			t.Fatal(result)
		}
	}
}
