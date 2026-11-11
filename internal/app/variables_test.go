package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"testing"
)

func TestRepositoryVariablesPlanSelectedNames(t *testing.T) {
	for _, tc := range []struct {
		name    string
		before  *string
		value   string
		fail    bool
		changes int
	}{
		{"absent", nil, "create-value", false, 1},
		{"changed", variableString("old"), "日本語-v2", false, 1},
		{"matching", variableString("same"), "same", false, 0},
		{"denied", nil, "wanted", true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "display_name", Value: tc.value}})
			reads := 0
			result, err := Plan(func() (settings.Declaration, error) { return settings.Declaration{Variables: &d}, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, Reader{Variable: func(repo repository.Repository, name string) (settings.VariableObservation, error) {
				reads++
				if name != "DISPLAY_NAME" || repo.Owner != "owner" {
					t.Fatal("wrong selection")
				}
				if tc.fail {
					return settings.VariableObservation{}, errors.New("denied")
				}
				if tc.before == nil {
					return settings.NewAbsentVariableObservation(), nil
				}
				return settings.NewVariableObservation(settings.RepositoryVariable{Name: name, Value: *tc.before})
			}})
			if (err != nil) != tc.fail || reads != 1 || len(result.Comparison.Variables) != tc.changes {
				t.Fatalf("result=%#v err=%v reads=%d", result, err, reads)
			}
			if tc.fail {
				if result.Comparison.Managed != 0 {
					t.Fatal("partial plan exposed")
				}
				return
			}
			if result.Comparison.Managed != 1 || result.Comparison.HasChanges() != (tc.changes > 0) {
				t.Fatal(result.Comparison)
			}
			if tc.changes > 0 {
				c := result.Comparison.Variables[0]
				before, after := c.Values()
				if c.Name() != "DISPLAY_NAME" || after != tc.value || (before == nil) != (tc.before == nil) || before != nil && *before != *tc.before {
					t.Fatal("lost values")
				}
			}
		})
	}
	for _, empty := range []*settings.VariableDeclarations{nil, new(settings.VariableDeclarations)} {
		_, err := Plan(func() (settings.Declaration, error) { return settings.Declaration{Variables: empty}, nil }, func() (repository.Repository, error) {
			t.Fatal("resolved empty selection")
			return repository.Repository{}, nil
		}, Reader{})
		if err != nil {
			t.Fatal(err)
		}
	}
}
func variableString(value string) *string { return &value }

type variableReadFailure struct{ error }

func (*variableReadFailure) ReadFailedBeforeWrite() {}
func TestRepositoryVariablesApplyProgress(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wrote   bool
		fail    error
		outcome ApplyOutcome
		status  ApplyOperationStatus
		calls   int
	}{
		{"matching", false, nil, ApplyMatchingLabels, OperationUnchanged, 2},
		{"completed", true, nil, ApplyCompleted, OperationCompleted, 2},
		{"pre read failed", false, &variableReadFailure{errors.New("denied")}, ApplyFailed, OperationFailedRead, 1},
		{"write unknown", false, errors.New("unknown write effect"), ApplyFailed, OperationFailedUnknown, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "B", Value: "b"}, {Name: "A", Value: "a"}})
			var names []string
			result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Variables: &d}, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, Writer{Variable: func(repo repository.Repository, value settings.RepositoryVariable) (bool, error) {
				names = append(names, value.Name)
				if repo.Name != "current" || value.Value != map[string]string{"A": "a", "B": "b"}[value.Name] {
					t.Fatal("wrong target/value")
				}
				return tc.wrote, tc.fail
			}})
			if (err != nil) != (tc.fail != nil) || result.Outcome != tc.outcome || len(names) != tc.calls || names[0] != "A" || len(result.Operations) != 2 || result.Operations[0].Status != tc.status {
				t.Fatalf("result=%#v err=%v names=%v", result, err, names)
			}
			if tc.fail != nil && result.Operations[1].Status != OperationNotExecuted {
				t.Fatal(result)
			}
		})
	}
	d, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "A", Value: "a"}, {Name: "B", Value: "b"}, {Name: "C", Value: "c"}})
	calls := 0
	result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Variables: &d}, nil }, func() (repository.Repository, error) { return repository.Repository{}, nil }, Writer{Variable: func(repository.Repository, settings.RepositoryVariable) (bool, error) {
		calls++
		if calls == 2 {
			return false, errors.New("later write failed")
		}
		return true, nil
	}})
	if err == nil || calls != 2 || result.Outcome != ApplyPartialFailure || result.Operations[0].Status != OperationCompleted || result.Operations[1].Status != OperationFailedUnknown || result.Operations[2].Status != OperationNotExecuted {
		t.Fatal(result, err, calls)
	}
	for _, empty := range []*settings.VariableDeclarations{nil, new(settings.VariableDeclarations)} {
		result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Variables: empty}, nil }, func() (repository.Repository, error) {
			t.Fatal("resolved empty selection")
			return repository.Repository{}, nil
		}, Writer{})
		if err != nil || result.Outcome != ApplyUnmanaged {
			t.Fatal(result, err)
		}
	}
}

func TestRepositoryVariablesExportSelection(t *testing.T) {
	for _, mode := range []string{"ordinary", "selected", "missing", "denied", "invalid", "observed-empty"} {
		t.Run(mode, func(t *testing.T) {
			reader := exportFixtureReader(t, settings.RepositoryState{}, func(string, repository.Repository) error { return nil })
			var reads []string
			reader.Variable = func(repo repository.Repository, name string) (settings.VariableObservation, error) {
				reads = append(reads, name)
				if mode == "missing" {
					return settings.NewAbsentVariableObservation(), nil
				}
				if mode == "denied" {
					return settings.VariableObservation{}, errors.New("denied")
				}
				value := map[string]string{"DISPLAY_NAME": "日本語-v2", "SECOND": "present"}[name]
				if mode == "observed-empty" {
					value = ""
				}
				return settings.NewVariableObservation(settings.RepositoryVariable{Name: name, Value: value})
			}
			names := []string{"display_name", "SECOND"}
			if mode == "ordinary" {
				names = nil
			}
			if mode == "invalid" {
				names = []string{"a", "A"}
			}
			resolved, saved := 0, false
			err := Export(ExportCheckout{Variables: names, Repository: func() (repository.Repository, error) {
				resolved++
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					saved = true
					if mode == "ordinary" {
						if d.Variables != nil {
							t.Fatal("ordinary export included variables")
						}
						return nil
					}
					values := d.Variables.Values()
					if len(values) != 2 || values[0].Name != "DISPLAY_NAME" || values[0].Value != "日本語-v2" || values[1].Name != "SECOND" || values[1].Value != "present" {
						t.Fatal(values)
					}
					return nil
				}, nil
			}}, reader)
			if mode == "selected" && !reflect.DeepEqual(reads, []string{"DISPLAY_NAME", "SECOND"}) {
				t.Fatal("unexpected reads", reads)
			}
			fail := mode == "missing" || mode == "denied" || mode == "invalid" || mode == "observed-empty"
			if (err != nil) != fail || saved == fail {
				t.Fatal(mode, err, saved)
			}
			if mode == "ordinary" && len(reads) != 0 || mode == "invalid" && (resolved != 0 || len(reads) != 0) {
				t.Fatal("unselected/invalid remote access", reads, resolved)
			}
		})
	}
}

func TestRepositoryVariablesEmptyYAMLBeforeTarget(t *testing.T) {
	appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {settings: {actions: {variables: [{name: DISPLAY_NAME, value: ''}]}}}\n", false)
	resolve := func() (repository.Repository, error) {
		t.Fatal("resolved invalid empty variable")
		return repository.Repository{}, nil
	}
	if _, err := Plan(checkoutDeclaration, resolve, Reader{}); err == nil {
		t.Fatal("accepted empty variable")
	}
	if _, err := Apply(checkoutDeclaration, resolve, Writer{}); err == nil {
		t.Fatal("accepted empty variable")
	}
}
