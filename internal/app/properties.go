package app

import (
	"errors"
	"fmt"
	"sort"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func propertyNames(d settings.PropertyDeclarations) []string {
	names := make([]string, 0, len(d))
	for name := range d {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func applyProperties(result *ApplyResult, repo repository.Repository, d settings.PropertyDeclarations, write Writer) (bool, error) {
	i := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: PropertiesRead})
	o, err := write.ReadProperties(repo, propertyNames(d))
	if err == nil {
		_, err = settings.CompareProperties(o, d)
	}
	if err != nil {
		result.failOperation(i, OperationFailedRead)
		return false, fmt.Errorf("preflight custom properties (earlier successful writes remain applied): %w", err)
	}
	result.Operations[i].Status = OperationCompleted
	i = len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: PropertiesUpdate})
	wrote, err := write.Properties(repo, d)
	if err != nil {
		status := OperationFailedUnknown
		var pre PreWriteReadFailure
		if errors.As(err, &pre) {
			status = OperationFailedRead
		}
		result.failOperation(i, status)
		return false, fmt.Errorf("apply custom properties (earlier successful writes remain applied): %w", err)
	}
	if wrote {
		result.Operations[i].Status = OperationCompleted
	}
	return wrote, nil
}
