package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestAgentsVariablesPlan(t *testing.T) {
	for _, mode := range []string{"absent", "changed", "matching", "denied", "unacquired", "mixed", "mixed-denied"} {
		t.Run(mode, func(t *testing.T) {
			d, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "shared", Value: "agent"}})
			desired := settings.Declaration{AgentsVariables: &d}
			mixed := mode == "mixed" || mode == "mixed-denied"
			if mixed {
				a, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "SHARED", Value: "workflow"}})
				desired.Variables = &a
			}
			var reads []string
			result, err := Plan(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, Reader{
				Variable: func(_ repository.Repository, name string) (settings.VariableObservation, error) {
					if !mixed || name != "SHARED" {
						t.Fatal("unselected Actions read", name)
					}
					reads = append(reads, "actions")
					return settings.NewVariableObservation(settings.RepositoryVariable{Name: name, Value: "workflow-before"})
				},
				AgentVariable: func(repo repository.Repository, name string) (settings.VariableObservation, error) {
					reads = append(reads, "agents")
					if name != "SHARED" || repo.Owner != "owner" || repo.Name != "current" {
						t.Fatal("wrong selection", repo, name)
					}
					switch mode {
					case "absent":
						return settings.NewAbsentVariableObservation(), nil
					case "denied", "mixed-denied":
						return settings.VariableObservation{}, errors.New("denied")
					case "unacquired":
						return settings.VariableObservation{}, nil
					case "matching":
						return settings.NewVariableObservation(settings.RepositoryVariable{Name: name, Value: "agent"})
					default:
						return settings.NewVariableObservation(settings.RepositoryVariable{Name: name, Value: "agent-before"})
					}
				},
			})
			wantReads := []string{"agents"}
			if mixed {
				wantReads = []string{"actions", "agents"}
			}
			if !reflect.DeepEqual(reads, wantReads) {
				t.Fatal(reads)
			}
			if mode == "denied" || mode == "mixed-denied" || mode == "unacquired" {
				var failure *PlanError
				if !errors.As(err, &failure) || failure.Stage != "read-failure" || failure.Path != "repository.settings.agents.variables" || !reflect.DeepEqual(result, PlanResult{}) {
					t.Fatal(result, err)
				}
				return
			}
			managed, changes := 1, 1
			if mixed {
				managed = 2
			}
			if mode == "matching" {
				changes = 0
			}
			if err != nil || result.Comparison.Managed != managed || len(result.Comparison.AgentsVariables) != changes || result.Comparison.HasChanges() != (changes > 0) || len(result.Comparison.Variables) != managed-1 {
				t.Fatal(result, err)
			}
			if changes > 0 {
				change := result.Comparison.AgentsVariables[0]
				before, after := change.Values()
				if change.Name() != "SHARED" || after != "agent" || (before == nil) != (mode == "absent") || before != nil && *before != "agent-before" {
					t.Fatal(change)
				}
			}
			if mixed {
				before, after := result.Comparison.Variables[0].Values()
				if *before != "workflow-before" || after != "workflow" {
					t.Fatal(result.Comparison.Variables)
				}
			}
		})
	}
	for _, empty := range []*settings.VariableDeclarations{nil, new(settings.VariableDeclarations)} {
		_, err := Plan(func() (settings.Declaration, error) { return settings.Declaration{AgentsVariables: empty}, nil }, func() (repository.Repository, error) {
			t.Fatal("resolved empty Agents selection")
			return repository.Repository{}, nil
		}, Reader{})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestAgentsVariablesApply(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wrote   bool
		failure error
		outcome ApplyOutcome
		status  ApplyOperationStatus
	}{
		{"matching", false, nil, ApplyMatchingLabels, OperationUnchanged},
		{"completed", true, nil, ApplyCompleted, OperationCompleted},
		{"pre-read", false, &variableReadFailure{errors.New("denied")}, ApplyFailed, OperationFailedRead},
		{"unknown", false, errors.New("unknown write effect"), ApplyFailed, OperationFailedUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "B", Value: "b"}, {Name: "a", Value: "a"}})
			var names []string
			result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{AgentsVariables: &d}, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, Writer{
				Variable: func(repository.Repository, settings.RepositoryVariable) (bool, error) {
					t.Fatal("unselected Actions write")
					return false, nil
				},
				AgentVariable: func(repo repository.Repository, value settings.RepositoryVariable) (bool, error) {
					names = append(names, value.Name)
					if repo.Name != "current" || value.Value != map[string]string{"A": "a", "B": "b"}[value.Name] {
						t.Fatal(repo, value)
					}
					return tc.wrote, tc.failure
				},
			})
			wantNames := []string{"A", "B"}
			wantStatus := []ApplyOperationStatus{tc.status, tc.status}
			if tc.failure != nil {
				wantNames = wantNames[:1]
				wantStatus[1] = OperationNotExecuted
			}
			if (err != nil) != (tc.failure != nil) || result.Outcome != tc.outcome || !reflect.DeepEqual(names, wantNames) || len(result.Operations) != 2 {
				t.Fatal(result, err, names)
			}
			for i, op := range result.Operations {
				if op.Kind != AgentVariableUpsert || op.Status != wantStatus[i] || op.VariableName != []string{"A", "B"}[i] {
					t.Fatal(result)
				}
			}
			if err != nil && !strings.Contains(err.Error(), "apply Agents variables") {
				t.Fatal(err)
			}
		})
	}
	for _, empty := range []*settings.VariableDeclarations{nil, new(settings.VariableDeclarations)} {
		result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{AgentsVariables: empty}, nil }, func() (repository.Repository, error) {
			t.Fatal("resolved empty Agents selection")
			return repository.Repository{}, nil
		}, Writer{})
		if err != nil || result.Outcome != ApplyUnmanaged {
			t.Fatal(result, err)
		}
	}
}

func TestAgentsVariablesApplyMixedPartialFailure(t *testing.T) {
	a, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "SHARED", Value: "workflow"}})
	d, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "SHARED", Value: "agent"}, {Name: "Z", Value: "later"}})
	var calls []string
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Variables: &a, AgentsVariables: &d}, nil
	}, func() (repository.Repository, error) { return repository.Repository{}, nil }, Writer{
		Variable: func(_ repository.Repository, v settings.RepositoryVariable) (bool, error) {
			calls = append(calls, "actions:"+v.Name+"="+v.Value)
			return true, nil
		},
		AgentVariable: func(_ repository.Repository, v settings.RepositoryVariable) (bool, error) {
			calls = append(calls, "agents:"+v.Name+"="+v.Value)
			return false, errors.New("write uncertain")
		},
	})
	want := []ApplyOperation{{Kind: VariableUpsert, VariableName: "SHARED", Status: OperationCompleted}, {Kind: AgentVariableUpsert, VariableName: "SHARED", Status: OperationFailedUnknown}, {Kind: AgentVariableUpsert, VariableName: "Z", Status: OperationNotExecuted}}
	if err == nil || result.Outcome != ApplyPartialFailure || !reflect.DeepEqual(result.Operations, want) || !reflect.DeepEqual(calls, []string{"actions:SHARED=workflow", "agents:SHARED=agent"}) {
		t.Fatal(result, err, calls)
	}
}

func TestAgentsVariablesApplyArchivedGuard(t *testing.T) {
	d, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "A", Value: "a"}})
	archived := true
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{AgentsVariables: &d, Repository: settings.RepositoryDeclaration{Archived: &archived}}, nil
	}, func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}, Writer{
		ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
			return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
		},
		AgentVariable: func(repository.Repository, settings.RepositoryVariable) (bool, error) {
			t.Fatal("attempted write on archived repository")
			return false, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "archived repository cannot update") || result.Outcome != ApplyFailed {
		t.Fatal(result, err)
	}
}
