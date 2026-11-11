package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func environmentVariableIntent(environment string, values ...settings.RepositoryVariable) settings.EnvironmentVariableDeclaration {
	variables, _ := settings.NewVariableDeclarations(values)
	return settings.EnvironmentVariableDeclaration{Environment: environment, Variables: variables}
}
func TestEnvironmentVariablesPlanIsolation(t *testing.T) {
	desired := settings.Declaration{EnvironmentVariables: []settings.EnvironmentVariableDeclaration{
		environmentVariableIntent("production", settings.RepositoryVariable{Name: "shared", Value: "env-value"}),
		environmentVariableIntent("staging", settings.RepositoryVariable{Name: "shared", Value: "other-value"}),
	}}
	var reads []string
	result, err := Plan(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}, Reader{EnvironmentVariable: func(_ repository.Repository, environment, name string) (settings.VariableObservation, error) {
		reads = append(reads, environment+":"+name)
		return settings.NewAbsentVariableObservation(), nil
	}})
	if err != nil || result.Comparison.Managed != 1 || len(result.Comparison.EnvironmentVariables) != 2 || !result.Comparison.HasChanges() || !reflect.DeepEqual(reads, []string{"production:SHARED", "staging:SHARED"}) {
		t.Fatalf("result=%+v reads=%v err=%v", result, reads, err)
	}
	// A variable-only intent has no policy reader, repository or family reader.
	for _, empty := range [][]settings.EnvironmentVariableDeclaration{nil, {environmentVariableIntent("production")}} {
		_, err := Plan(func() (settings.Declaration, error) { return settings.Declaration{EnvironmentVariables: empty}, nil }, func() (repository.Repository, error) {
			t.Fatal("resolved empty intent")
			return repository.Repository{}, nil
		}, Reader{})
		if err != nil {
			t.Fatal(err)
		}
	}
}
func TestEnvironmentVariablesApplyProgress(t *testing.T) {
	for _, pre := range []bool{false, true} {
		t.Run(map[bool]string{false: "unknown", true: "pre-write"}[pre], func(t *testing.T) {
			desired := settings.Declaration{EnvironmentVariables: []settings.EnvironmentVariableDeclaration{
				environmentVariableIntent("staging", settings.RepositoryVariable{Name: "z", Value: "not-attempted"}),
				environmentVariableIntent("production", settings.RepositoryVariable{Name: "b", Value: "failure-value"}, settings.RepositoryVariable{Name: "a", Value: "saved-value"}),
			}}
			var calls []string
			result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, Writer{EnvironmentVariable: func(_ repository.Repository, environment string, value settings.RepositoryVariable) (bool, error) {
				calls = append(calls, environment+":"+value.Name)
				if value.Name == "B" {
					if pre {
						return false, &variableReadFailure{errors.New("denied")}
					}
					return false, errors.New("uncertain")
				}
				return true, nil
			}})
			status := OperationFailedUnknown
			if pre {
				status = OperationFailedRead
			}
			want := []ApplyOperation{{Kind: EnvironmentVariableUpsert, EnvironmentName: "production", VariableName: "A", Status: OperationCompleted}, {Kind: EnvironmentVariableUpsert, EnvironmentName: "production", VariableName: "B", Status: status}, {Kind: EnvironmentVariableUpsert, EnvironmentName: "staging", VariableName: "Z", Status: OperationNotExecuted}}
			if err == nil || result.Outcome != ApplyPartialFailure || !reflect.DeepEqual(result.Operations, want) || !reflect.DeepEqual(calls, []string{"production:A", "production:B"}) {
				t.Fatalf("result=%+v calls=%v err=%v", result, calls, err)
			}
		})
	}
}
func TestEnvironmentVariablesExportSelection(t *testing.T) {
	for _, mode := range []string{"selected", "ordinary", "missing", "denied", "invalid", "unacquired"} {
		t.Run(mode, func(t *testing.T) {
			reader := exportFixtureReader(t, settings.RepositoryState{}, func(string, repository.Repository) error { return nil })
			var reads []string
			reader.EnvironmentVariable = func(_ repository.Repository, environment, name string) (settings.VariableObservation, error) {
				reads = append(reads, environment+":"+name)
				switch mode {
				case "missing":
					return settings.NewAbsentVariableObservation(), nil
				case "denied":
					return settings.VariableObservation{}, errors.New("denied")
				case "unacquired":
					return settings.VariableObservation{}, nil
				}
				return settings.NewVariableObservation(settings.RepositoryVariable{Name: name, Value: environment + "-value"})
			}
			selections := []settings.EnvironmentVariableSelection{{Environment: "production", Name: "shared"}, {Environment: "production", Name: "second"}, {Environment: "staging", Name: "shared"}}
			if mode == "ordinary" {
				selections = nil
			}
			if mode == "invalid" {
				selections = append(selections, settings.EnvironmentVariableSelection{Environment: "PRODUCTION", Name: "SHARED"})
			}
			saved, resolved := false, false
			err := Export(ExportCheckout{EnvironmentVariables: selections, Repository: func() (repository.Repository, error) {
				resolved = true
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					saved = true
					if d.Environments != nil {
						t.Fatal("variable selection fabricated policy intent")
					}
					if mode == "ordinary" {
						if len(d.EnvironmentVariables) != 0 {
							t.Fatal(d)
						}
						return nil
					}
					if len(d.EnvironmentVariables) != 2 || d.EnvironmentVariables[0].Environment != "production" || d.EnvironmentVariables[0].Variables.Len() != 2 || d.EnvironmentVariables[1].Variables.Len() != 1 {
						t.Fatal(d.EnvironmentVariables)
					}
					return nil
				}, nil
			}}, reader)
			fail := mode != "ordinary" && mode != "selected"
			if (err != nil) != fail || saved == fail || mode == "invalid" && resolved || mode == "ordinary" && len(reads) != 0 {
				t.Fatalf("err=%v saved=%v resolved=%v reads=%v", err, saved, resolved, reads)
			}
		})
	}
}

func TestEnvironmentVariablesCombinedCreation(t *testing.T) {
	policies, _ := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "production"}})
	desired := settings.Declaration{Environments: &policies, EnvironmentVariables: []settings.EnvironmentVariableDeclaration{environmentVariableIntent("PRODUCTION", settings.RepositoryVariable{Name: "one", Value: "new-value"})}}
	load := func() (settings.Declaration, error) { return desired, nil }
	resolve := func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}
	read := func(repository.Repository, []string) (settings.EnvironmentObservation, error) {
		return settings.NewEnvironmentObservation([]settings.EnvironmentState{})
	}
	result, err := Plan(load, resolve, Reader{Environments: read, EnvironmentVariable: func(repository.Repository, string, string) (settings.VariableObservation, error) {
		t.Fatal("read variable of not-yet-created Environment")
		return settings.VariableObservation{}, nil
	}})
	if err != nil || result.Comparison.Managed != 2 || len(result.Comparison.Environments) != 1 || len(result.Comparison.EnvironmentVariables) != 1 || result.Comparison.EnvironmentVariables[0].Change.Operation() != "create" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	var calls []string
	applied, err := Apply(load, resolve, Writer{ReadEnvironments: read, Environment: func(_ repository.Repository, policy settings.EnvironmentDeclaration) (bool, error) {
		calls = append(calls, "policy:"+policy.Name)
		return true, nil
	}, EnvironmentVariable: func(_ repository.Repository, environment string, value settings.RepositoryVariable) (bool, error) {
		calls = append(calls, "variable:"+environment+":"+value.Name)
		return true, nil
	}})
	if err != nil || !applied.Completed() || !reflect.DeepEqual(calls, []string{"policy:production", "variable:PRODUCTION:ONE"}) {
		t.Fatalf("result=%+v err=%v calls=%v", applied, err, calls)
	}
}
