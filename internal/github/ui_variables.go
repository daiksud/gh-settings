package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func VariablesDeclaration(values *[]uimodel.RepositoryVariable) (*settings.VariableDeclarations, error) {
	if values == nil {
		return nil, nil
	}
	entries := make([]settings.RepositoryVariable, len(*values))
	for i, value := range *values {
		if value.Value == nil {
			return nil, fmt.Errorf("repository variable %q requires value", value.Name)
		}
		entries[i] = settings.RepositoryVariable{Name: value.Name, Value: *value.Value}
	}
	result, err := settings.NewVariableDeclarations(entries)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
