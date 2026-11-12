package settings

import "fmt"

// EnvironmentVariableDeclaration selects non-sensitive values independently of
// the Environment's deployment policy. Empty variables select no remote work.
type EnvironmentVariableDeclaration struct {
	Environment string
	Variables   VariableDeclarations
}

type EnvironmentVariableSelection struct {
	Environment string
	Name        string
}

// CanonicalEnvironmentVariableSelections retains Environment spelling while
// rejecting equivalent Environment/name identities before remote work.
func CanonicalEnvironmentVariableSelections(values []EnvironmentVariableSelection) ([]EnvironmentVariableSelection, error) {
	result := make([]EnvironmentVariableSelection, len(values))
	seen := map[EnvironmentVariableSelection]bool{}
	for i, value := range values {
		if err := validateEnvironment(EnvironmentDeclaration{Name: value.Environment}, false); err != nil {
			return nil, err
		}
		name, err := VariableName(value.Name)
		if err != nil {
			return nil, err
		}
		key := EnvironmentVariableSelection{Environment: EnvironmentIdentity(value.Environment), Name: name}
		if seen[key] {
			return nil, fmt.Errorf("duplicate environment variable %q in %q", name, value.Environment)
		}
		seen[key] = true
		result[i] = EnvironmentVariableSelection{Environment: value.Environment, Name: name}
	}
	return result, nil
}

// EnvironmentVariableChange keeps identical variable names in separate scopes.
type EnvironmentVariableChange struct {
	Environment string
	Change      VariableChange
}

func NewEnvironmentVariableChange(environment string, change VariableChange) (EnvironmentVariableChange, error) {
	if err := validateEnvironment(EnvironmentDeclaration{Name: environment}, false); err != nil {
		return EnvironmentVariableChange{}, err
	}
	before, after := change.Values()
	var validated VariableChange
	var err error
	if change.Operation() == "delete" && before != nil {
		validated, err = NewVariableDelete(RepositoryVariable{Name: change.Name(), Value: *before})
	} else {
		validated, err = NewVariableChange(change.Name(), before, after)
	}
	if err != nil {
		return EnvironmentVariableChange{}, err
	}
	return EnvironmentVariableChange{Environment: environment, Change: validated}, nil
}

func EnvironmentVariablesEmpty(values []EnvironmentVariableDeclaration) bool {
	for _, value := range values {
		if value.Variables.Len() != 0 {
			return false
		}
	}
	return true
}
