package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func webhookIDs(desired []settings.Webhook) []int64 {
	ids := make([]int64, 0, len(desired))
	for _, v := range desired {
		ids = append(ids, v.ID)
	}
	return ids
}
func applyWebhooks(result *ApplyResult, repo repository.Repository, desired []settings.Webhook, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: WebhooksRead})
	var observed settings.WebhookObservation
	var err error
	if write.ReadWebhooks == nil || write.Webhook == nil {
		err = fmt.Errorf("webhook adapter unavailable")
	} else {
		observed, err = write.ReadWebhooks(repo, webhookIDs(desired))
	}
	var selected []settings.WebhookSelection
	if err == nil {
		selected, _, err = settings.PrepareWebhooks(observed, desired)
	}
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("prepare webhooks%s: %w", environmentProgress(*result), err)
	}
	result.Operations[index].Status = OperationCompleted
	wrote := false
	for _, s := range selected {
		index = len(result.Operations)
		result.Operations = append(result.Operations, ApplyOperation{Kind: WebhookUpdate})
		changed, e := write.Webhook(repo, s)
		if e != nil {
			status := OperationFailedUnknown
			var pre PreWriteReadFailure
			if errors.As(e, &pre) {
				status = OperationFailedRead
			}
			result.failOperation(index, status)
			uncertain := ""
			if status == OperationFailedUnknown {
				uncertain = " (unconfirmed effects may remain)"
			}
			return wrote, fmt.Errorf("apply webhooks%s%s: %w", environmentProgress(*result), uncertain, e)
		}
		result.Operations[index].Status = OperationUnchanged
		if changed {
			result.Operations[index].Status = OperationCompleted
			wrote = true
		}
	}
	return wrote, nil
}
