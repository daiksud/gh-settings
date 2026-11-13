package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func TestVariablePrunePlanKeepsFamiliesAndEmptyListsSeparate(t *testing.T) {
	empty, _ := settings.NewVariableDeclarations(nil)
	d := settings.Declaration{Variables: &empty, AgentsVariables: &empty, EnvironmentVariables: []settings.EnvironmentVariableDeclaration{environmentVariableIntent("prod/blue")}}
	var calls []string
	result, err := Plan(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Reader{
		Variables: func(repository.Repository) (settings.VariableInventory, error) {
			calls = append(calls, "actions")
			return settings.NewVariableInventory([]settings.RepositoryVariable{{Name: "SAME", Value: "actions"}})
		},
		AgentVariables: func(repository.Repository) (settings.VariableInventory, error) {
			calls = append(calls, "agents")
			return settings.NewVariableInventory([]settings.RepositoryVariable{{Name: "SAME", Value: "agents"}})
		},
		EnvironmentVariables: func(_ repository.Repository, environment string) (settings.VariableInventory, error) {
			calls = append(calls, environment)
			return settings.NewEnvironmentVariableInventory(environment, 7, []settings.RepositoryVariable{{Name: "SAME", Value: "env"}})
		},
	}, mustPrune(t, "actions-variables,agents-variables,environment-variables"))
	if err != nil || result.Comparison.Managed != 3 || len(result.Comparison.Variables) != 1 || len(result.Comparison.AgentsVariables) != 1 || len(result.Comparison.EnvironmentVariables) != 1 || !reflect.DeepEqual(calls, []string{"actions", "agents", "prod/blue"}) {
		t.Fatal(result, calls, err)
	}
	for _, c := range []settings.VariableChange{result.Comparison.Variables[0], result.Comparison.AgentsVariables[0], result.Comparison.EnvironmentVariables[0].Change} {
		if c.Operation() != "delete" {
			t.Fatal(c)
		}
	}
}
func TestVariablePruneAllCandidatesPrecedeUnrelatedWrites(t *testing.T) {
	empty, _ := settings.NewVariableDeclarations(nil)
	yes := true
	d := settings.Declaration{Variables: &empty, AgentsVariables: &empty}
	d.Repository.HasIssues = &yes
	calls := []string{}
	_, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
		ReadVariables: func(repository.Repository) (settings.VariableInventory, error) {
			calls = append(calls, "actions")
			return settings.NewVariableInventory([]settings.RepositoryVariable{{Name: "OLD", Value: "x"}})
		},
		ReadAgentVariables: func(repository.Repository) (settings.VariableInventory, error) {
			calls = append(calls, "agents")
			return settings.VariableInventory{}, errors.New("page two incomplete")
		},
		Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
			t.Fatal("write before complete candidate acquisition")
			return true, nil
		},
	}, mustPrune(t, "actions-variables,agents-variables"))
	if err == nil || !strings.Contains(err.Error(), "before any writes") || !reflect.DeepEqual(calls, []string{"actions", "agents"}) {
		t.Fatal(err, calls)
	}
}
func TestVariablePruneUpsertsBeforeFixedDeletionAndPartialFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		d, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "NEW", Value: "new"}})
		var calls []string
		result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Variables: &d}, nil }, checkoutRepository, Writer{
			ReadVariables: func(repository.Repository) (settings.VariableInventory, error) {
				return settings.NewVariableInventory([]settings.RepositoryVariable{{Name: "OLD", Value: "old"}})
			},
			Variable: func(_ repository.Repository, value settings.RepositoryVariable) (bool, error) {
				calls = append(calls, "upsert:"+value.Name)
				return true, nil
			},
			DeleteVariable: func(_ repository.Repository, c settings.VariableChange) error {
				calls = append(calls, "delete:"+c.Name())
				if fail {
					return errors.New("readback denied; write may have applied")
				}
				return nil
			},
		}, mustPrune(t, "actions-variables"))
		if (err != nil) != fail || !reflect.DeepEqual(calls, []string{"upsert:NEW", "delete:OLD"}) {
			t.Fatal(result, calls, err)
		}
		if fail && (result.Outcome != ApplyPartialFailure || result.Operations[len(result.Operations)-1].Kind != VariableDelete || result.Operations[len(result.Operations)-1].Status != OperationFailedUnknown) {
			t.Fatal(result)
		}
	}
}
func TestEnvironmentVariablePruneRejectsParentReplacementBeforeFamilyWrite(t *testing.T) {
	reads := 0
	_, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{EnvironmentVariables: []settings.EnvironmentVariableDeclaration{environmentVariableIntent("prod")}}, nil
	}, checkoutRepository, Writer{
		ReadEnvironmentVariables: func(_ repository.Repository, env string) (settings.VariableInventory, error) {
			reads++
			return settings.NewEnvironmentVariableInventory(env, int64(reads), []settings.RepositoryVariable{{Name: "OLD", Value: "old"}})
		},
		DeleteEnvironmentVariable: func(repository.Repository, string, int64, settings.VariableChange) error {
			t.Fatal("replaced parent variable deleted")
			return nil
		},
	}, mustPrune(t, "environment-variables"))
	if err == nil || !strings.Contains(err.Error(), "Environment identity changed") {
		t.Fatal(err)
	}
}
func TestPruneCanAccompanyExplicitNewEnvironmentWithoutImplicitCreation(t *testing.T) {
	environments, _ := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "new"}})
	d := settings.Declaration{Environments: &environments, EnvironmentVariables: []settings.EnvironmentVariableDeclaration{environmentVariableIntent("new", settings.RepositoryVariable{Name: "NEW", Value: "value"})}}
	created := false
	result, err := Apply(func() (settings.Declaration, error) { return d, nil }, checkoutRepository, Writer{
		ReadEnvironments: func(repository.Repository, []string) (settings.EnvironmentObservation, error) {
			return settings.NewEnvironmentObservation(nil)
		},
		Environment: func(repository.Repository, settings.EnvironmentDeclaration) (bool, error) {
			created = true
			return true, nil
		},
		ReadEnvironmentVariables: func(_ repository.Repository, env string) (settings.VariableInventory, error) {
			if !created {
				t.Fatal("nonexistent parent read as an existing empty collection")
			}
			return settings.NewEnvironmentVariableInventory(env, 7, nil)
		},
		EnvironmentVariable: func(_ repository.Repository, env string, value settings.RepositoryVariable) (bool, error) {
			if !created || env != "new" || value.Name != "NEW" {
				t.Fatal(created, env, value)
			}
			return true, nil
		},
		DeleteEnvironmentVariable: func(repository.Repository, string, int64, settings.VariableChange) error {
			t.Fatal("new object deleted")
			return nil
		},
	}, mustPrune(t, "environment-variables"))
	if err != nil || !result.Completed() {
		t.Fatal(result, err)
	}
}
