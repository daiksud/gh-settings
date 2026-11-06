package app

import (
	"errors"
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyCacheStorage(result *ApplyResult, repo repository.Repository, d settings.ActionsPolicyDeclaration, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CacheStorageRead})
	observed, err := write.ReadCacheStorage(repo)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("read cache storage for apply (earlier writes remain applied): %w", err)
	}
	_, changes, err := settings.CompareCacheStorage(observed, d)
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("preflight cache storage (earlier writes remain applied): %w", err)
	}
	result.Operations[index].Status = OperationCompleted
	if len(changes) == 0 {
		return false, nil
	}
	index = len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: CacheStorageUpdate})
	wrote, err := write.CacheStorage(repo, *d.CacheSizeGB)
	if err != nil {
		status := OperationFailedUnknown
		var readFailure PreWriteReadFailure
		if errors.As(err, &readFailure) {
			status = OperationFailedRead
		}
		result.failOperation(index, status)
		return false, fmt.Errorf("apply cache storage (earlier successful writes remain applied): %w", err)
	}
	if wrote {
		result.Operations[index].Status = OperationCompleted
	} else {
		result.Operations[index].Status = OperationUnchanged
	}
	return wrote, nil
}
