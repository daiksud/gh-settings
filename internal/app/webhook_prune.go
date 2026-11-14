package app

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func applyWebhookDeletes(result *ApplyResult, repo repository.Repository, expected settings.WebhookPruneObservation, write Writer) (bool, error) {
	ids, err := expected.Candidates()
	if err != nil {
		return false, err
	}
	wrote := false
	for _, id := range ids {
		index := len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: WebhookDelete})
		attempted, err := write.DeleteWebhook(repo, expected, id)
		if err != nil {
			status := OperationFailedUnknown
			if !attempted {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			uncertain := ""
			if status == OperationFailedUnknown {
				uncertain = " (unconfirmed effects may remain)"
			}
			return wrote, fmt.Errorf("delete webhooks%s: %w", uncertain, err)
		}
		if !attempted {
			result.failOperation(index, OperationFailedRead)
			return wrote, fmt.Errorf("webhook deletion did not confirm an attempted operation")
		}
		result.Operations[index].Status = OperationCompleted
		wrote = true
		expected, err = expected.Without(id)
		if err != nil {
			return wrote, err
		}
	}
	return wrote, nil
}
