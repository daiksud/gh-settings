package app

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedEnvironmentVariablesRequireSelectedMatchesWithoutSideEffects(t *testing.T) {
	const value = "fixture-desired-private-canary"
	const other = "fixture-observed-private-canary"
	repo := repository.Repository{Owner: "owner", Name: "current"}
	for _, scenario := range []string{"match", "multiple parents", "policy match", "policy identity mismatch", "different", "absent", "unacquired", "wrong name", "incomplete", "invalid parent", "denied", "parent drift", "no reader", "repository drift", "repository unarchived", "prune", "environment variable prune"} {
		t.Run(scenario, func(t *testing.T) {
			archived := true
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, EnvironmentVariables: []settings.EnvironmentVariableDeclaration{
				environmentVariableIntent("Production", settings.RepositoryVariable{Name: "two", Value: value}, settings.RepositoryVariable{Name: "one", Value: value}),
			}}
			if scenario == "multiple parents" {
				d.EnvironmentVariables = append(d.EnvironmentVariables, environmentVariableIntent("staging", settings.RepositoryVariable{Name: "item", Value: value}))
			}
			if scenario == "policy match" || scenario == "policy identity mismatch" {
				policies, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "production"}})
				if err != nil {
					t.Fatal(err)
				}
				d.Environments = &policies
			}
			identityReads, policyReads, writes, local := 0, 0, 0, 0
			var selected []string
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					identityReads++
					id, active := int64(1), true
					if identityReads == 2 && scenario == "repository drift" {
						id = 2
					}
					if identityReads == 2 && scenario == "repository unarchived" {
						active = false
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: active}, nil
				},
				ReadEnvironments: func(_ repository.Repository, names []string) (settings.EnvironmentObservation, error) {
					policyReads++
					zero, no := 0, false
					reviewers := []settings.EnvironmentReviewer{}
					return settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 71, CustomRules: []settings.EnvironmentCustomRule{}, Policy: settings.EnvironmentDeclaration{Name: "production", WaitTimer: &zero, PreventSelfReview: &no, CanAdminsBypass: &no, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}}})
				},
				ReadEnvironmentVariables: func(_ repository.Repository, environment string) (settings.VariableInventory, error) {
					return settings.NewEnvironmentVariableInventory(environment, 71, []settings.RepositoryVariable{{Name: "ONE", Value: value}, {Name: "TWO", Value: value}})
				},
				ReadArchivedEnvironmentVariables: func(_ repository.Repository, environment string, names []string) (int64, []settings.VariableObservation, error) {
					selected = append(selected, environment+":"+strings.Join(names, ","))
					if scenario == "denied" || scenario == "parent drift" {
						return 0, nil, errors.New(other)
					}
					id := int64(71)
					if environment == "staging" {
						id = 72
					}
					if scenario == "policy identity mismatch" {
						id = 73
					}
					if scenario == "invalid parent" {
						id = 0
					}
					var observations []settings.VariableObservation
					for _, name := range names {
						v := value
						if scenario == "different" {
							v = other
						}
						if scenario == "wrong name" {
							name = "OTHER"
						}
						observation, err := settings.NewVariableObservation(settings.RepositoryVariable{Name: name, Value: v})
						if err != nil {
							t.Fatal(err)
						}
						if scenario == "absent" {
							observation = settings.NewAbsentVariableObservation()
						}
						if scenario == "unacquired" {
							observation = settings.VariableObservation{}
						}
						observations = append(observations, observation)
					}
					if scenario == "incomplete" {
						observations = observations[:1]
					}
					return id, observations, nil
				},
				EnvironmentVariable: func(repository.Repository, string, settings.RepositoryVariable) (bool, error) {
					writes++
					return true, nil
				},
				Environment:        func(repository.Repository, settings.EnvironmentDeclaration) (bool, error) { writes++; return true, nil },
				Archive:            func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID:          func(string, string, int64) error { local++; return nil },
				BindCollaboratorID: func(string, int64) error { local++; return nil },
			}
			if scenario == "no reader" {
				writer.ReadArchivedEnvironmentVariables = nil
			}
			var prune settings.PruneScopes
			if scenario == "prune" || scenario == "environment variable prune" {
				var err error
				scope := "labels"
				if scenario == "environment variable prune" {
					scope = "environment-variables"
				}
				prune, err = settings.ParsePruneScopes(scope)
				if err != nil {
					t.Fatal(err)
				}
			}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, writer, prune)
			matching := scenario == "match" || scenario == "multiple parents" || scenario == "policy match"
			if (err == nil) != matching || writes != 0 || local != 0 {
				t.Fatalf("outcome=%v err=%v writes=%d local=%d", result.Outcome, err, writes, local)
			}
			if strings.Contains(fmt.Sprintf("%+v %v", result, err), value) || strings.Contains(fmt.Sprintf("%+v %v", result, err), other) {
				t.Fatal("variable value leaked")
			}
			if matching && (identityReads != 2 || result.Outcome != ApplyMatchingLabels) {
				t.Fatalf("identity reads=%d outcome=%v", identityReads, result.Outcome)
			}
			if matching {
				want := []string{"Production:ONE,TWO"}
				if scenario == "multiple parents" {
					want = append(want, "staging:ITEM")
				}
				if !reflect.DeepEqual(selected, want) {
					t.Fatalf("selected %v, want %v", selected, want)
				}
			}
			if scenario == "policy match" && policyReads != 2 {
				t.Fatalf("policy reads=%d", policyReads)
			}
			if (scenario == "prune" || scenario == "environment variable prune") && len(selected) != 0 {
				t.Fatalf("prune must not enter matching: %v", selected)
			}
		})
	}
}

func TestUnarchivedEnvironmentVariablesUseOrdinaryWriter(t *testing.T) {
	archived := true
	d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, EnvironmentVariables: []settings.EnvironmentVariableDeclaration{environmentVariableIntent("production", settings.RepositoryVariable{Name: "ITEM", Value: "fixture"})}}
	repo := repository.Repository{Owner: "owner", Name: "current"}
	var writes, matchingReads int
	result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, Writer{
		ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
			return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: false}, nil
		},
		ReadArchivedEnvironmentVariables: func(repository.Repository, string, []string) (int64, []settings.VariableObservation, error) {
			matchingReads++
			return 0, nil, nil
		},
		EnvironmentVariable: func(_ repository.Repository, env string, value settings.RepositoryVariable) (bool, error) {
			writes++
			if env != "production" || value.Name != "ITEM" {
				t.Fatal("unexpected ordinary write")
			}
			return true, nil
		},
		Archive: func(repository.Repository, settings.ArchiveState, bool) error { return nil },
	})
	if err != nil || result.Outcome != ApplyCompleted || writes != 1 || matchingReads != 0 {
		t.Fatalf("outcome=%v err=%v writes=%d matchingReads=%d", result.Outcome, err, writes, matchingReads)
	}
}

func TestArchivedEnvironmentVariablesOmittedAndEmptyRemainUnmanaged(t *testing.T) {
	for _, empty := range []bool{false, true} {
		archived := true
		d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}}
		if empty {
			d.EnvironmentVariables = []settings.EnvironmentVariableDeclaration{environmentVariableIntent("production")}
		}
		reads, writes := 0, 0
		result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) {
			return repository.Repository{Owner: "owner", Name: "current"}, nil
		}, Writer{
			ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
				return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
			},
			ReadArchivedEnvironmentVariables: func(repository.Repository, string, []string) (int64, []settings.VariableObservation, error) {
				reads++
				return 0, nil, nil
			},
			EnvironmentVariable: func(repository.Repository, string, settings.RepositoryVariable) (bool, error) {
				writes++
				return true, nil
			},
		})
		if err != nil || result.Outcome != ApplyMatchingLabels || reads != 0 || writes != 0 {
			t.Fatalf("empty=%v outcome=%v err=%v reads=%d writes=%d", empty, result.Outcome, err, reads, writes)
		}
	}
}
