package app

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Never adopt a partial inventory or turn an unsupported value into absence.
func exportVariableInventory(inventory settings.VariableInventory) (settings.VariableDeclarations, error) {
	values, err := inventory.Values()
	if err != nil {
		return settings.VariableDeclarations{}, err
	}
	slices.SortFunc(values, func(a, b settings.RepositoryVariable) int { return strings.Compare(a.Name, b.Name) })
	return settings.NewVariableDeclarations(values)
}

func exportAllEnvironmentVariables(checkout ExportCheckout, read Reader, repo repository.Repository) ([]settings.EnvironmentVariableDeclaration, error) {
	prior := checkout.Existing.EnvironmentVariables
	info := func(path string, err error) error {
		if checkout.Info != nil {
			return checkout.Info(path, err)
		}
		return nil
	}
	const path = "repository.settings.environments.variables"
	if read.EnvironmentNames == nil {
		return prior, info(path, fmt.Errorf("environment inventory reader unavailable"))
	}
	names, err := read.EnvironmentNames(repo)
	if err == nil && names == nil {
		err = fmt.Errorf("missing complete environment inventory")
	}
	if err != nil {
		return prior, info(path, err)
	}
	// Complete names prove absent parents, but a failed per-parent variable list
	// retains that parent's valid prior declaration while other parents continue.
	result := make([]settings.EnvironmentVariableDeclaration, 0, len(names))
	var infoError error
	for _, name := range names {
		var old *settings.EnvironmentVariableDeclaration
		for _, entry := range prior {
			if settings.EnvironmentIdentity(entry.Environment) == settings.EnvironmentIdentity(name) {
				value := entry
				old = &value
				break
			}
		}
		var inventory settings.VariableInventory
		if read.EnvironmentVariables == nil {
			err = fmt.Errorf("variable inventory reader unavailable")
		} else {
			inventory, err = read.EnvironmentVariables(repo, name)
		}
		var d settings.VariableDeclarations
		if err == nil {
			_, err = inventory.EnvironmentID(name)
		}
		if err == nil {
			d, err = exportVariableInventory(inventory)
		}
		if err != nil {
			infoError = errors.Join(infoError, info(fmt.Sprintf("repository.settings.environments[%q].variables", name), err))
			if old != nil {
				result = append(result, *old)
			}
			continue
		}
		result = append(result, settings.EnvironmentVariableDeclaration{Environment: name, Variables: d})
	}
	return result, infoError
}
