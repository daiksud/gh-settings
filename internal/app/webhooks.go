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
		if v.ID == 0 {
			return nil
		}
		ids = append(ids, v.ID)
	}
	return ids
}
func applyWebhooks(result *ApplyResult, repo repository.Repository, desired []settings.Webhook, write Writer) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: WebhooksRead})
	var observed settings.WebhookObservation
	var err error
	ids := webhookIDs(desired)
	if write.ReadWebhooks == nil {
		err = fmt.Errorf("webhook reader unavailable")
	} else {
		observed, err = write.ReadWebhooks(repo, ids)
	}
	var selected []settings.WebhookSelection
	if err == nil {
		selected, _, err = settings.PrepareWebhooks(observed, desired)
	}
	if err == nil {
		for _, s := range selected {
			if s.Desired.ID == 0 && write.CreateWebhook == nil || s.Desired.ID > 0 && write.Webhook == nil {
				err = fmt.Errorf("webhook writer unavailable")
				break
			}
		}
	}
	if err != nil {
		result.failOperation(index, OperationFailedRead)
		return false, fmt.Errorf("prepare webhooks%s: %w", environmentProgress(*result), err)
	}
	result.Operations[index].Status = OperationCompleted
	expected := observed.Clone()
	wrote := false
	fail := func(index int, attempted bool, err error) (bool, error) {
		status := OperationFailedRead
		uncertain := ""
		if attempted {
			status = OperationFailedUnknown
			uncertain = " (unconfirmed effects may remain)"
		}
		result.failOperation(index, status)
		return wrote, fmt.Errorf("apply webhooks%s%s: %w", environmentProgress(*result), uncertain, err)
	}
	for _, s := range selected {
		index = len(result.Operations)
		if s.Desired.ID == 0 {
			result.Operations = append(result.Operations, ApplyOperation{Kind: WebhookCreate})
			applied, e := write.CreateWebhook(repo, expected, s.Desired)
			_, known := expected.Find(applied.RecoveryID)
			newID := applied.Attempted && applied.RecoveryID > 0 && !known
			if newID {
				result.CreatedIDs = append(result.CreatedIDs, CreatedObject{Collection: "webhooks", Name: *s.Desired.PayloadURL, ID: applied.RecoveryID})
			}
			if e == nil {
				state, found := applied.Observation.Find(applied.RecoveryID)
				want, guardErr := expected.WithCreated(state)
				if !newID || !found || !settings.WebhookCreationMatches(state, s.Desired) || guardErr != nil || settings.VerifyWebhookInventory(want, applied.Observation) != nil {
					e = fmt.Errorf("webhook creation lacks independently verified new identity")
				}
			}
			if e != nil {
				return fail(index, applied.Attempted, e)
			}
			expected = applied.Observation.Clone()
			wrote = true
			result.Operations[index].Status = OperationCompleted
			if write.CreatedID != nil {
				if e = write.CreatedID("webhooks", *s.Desired.PayloadURL, applied.RecoveryID); e != nil {
					result.Outcome = ApplyPartialFailure
					return wrote, fmt.Errorf("webhook created with id %d but local ID recording failed: %w", applied.RecoveryID, e)
				}
			}
			continue
		}
		result.Operations = append(result.Operations, ApplyOperation{Kind: WebhookUpdate})
		if ids == nil {
			snapshot := expected.Clone()
			s.Expected = &snapshot
		}
		changed, e := write.Webhook(repo, s)
		if e != nil {
			var pre PreWriteReadFailure
			return fail(index, !errors.As(e, &pre), e)
		}
		result.Operations[index].Status = OperationUnchanged
		if changed {
			result.Operations[index].Status = OperationCompleted
			wrote = true
			if ids == nil {
				expected, e = expected.WithUpdated(s.Desired)
				if e != nil {
					return fail(index, true, e)
				}
			}
		}
	}
	return wrote, nil
}
