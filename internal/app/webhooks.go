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
func applyWebhooks(result *ApplyResult, repo repository.Repository, desired []settings.Webhook, write Writer, frozen ...*settings.WebhookPruneObservation) (bool, error) {
	index := len(result.Operations)
	result.Operations = append(result.Operations, ApplyOperation{Kind: WebhooksRead})
	var observed settings.WebhookObservation
	var err error
	ids := webhookIDs(desired)
	var prune *settings.WebhookPruneObservation
	if len(frozen) > 0 && frozen[0] != nil {
		value := frozen[0].Clone()
		prune = &value
	}
	if prune != nil {
		if write.ReadWebhookPrune == nil || write.DeleteWebhook == nil {
			err = fmt.Errorf("webhook prune reader or writer unavailable")
		} else {
			var current settings.WebhookPruneObservation
			current, err = write.ReadWebhookPrune(repo)
			if err == nil {
				err = prune.Verify(current)
			}
			if err == nil {
				observed = prune.Inventory()
			}
		}
	} else if write.ReadWebhooks == nil {
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
	strictRead := func() (settings.WebhookPruneObservation, error) {
		if prune == nil {
			return settings.WebhookPruneObservation{}, nil
		}
		fresh, e := write.ReadWebhookPrune(repo)
		if e == nil {
			e = prune.Verify(fresh)
		}
		return fresh, e
	}
	for _, s := range selected {
		if prune != nil {
			_, e := strictRead()
			if e != nil {
				i := len(result.Operations)
				result.Operations = append(result.Operations, ApplyOperation{Kind: WebhooksRead})
				result.failOperation(i, OperationFailedRead)
				return wrote, fmt.Errorf("webhook inventory changed before write%s: %w", environmentProgress(*result), e)
			}
		}
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
			if prune != nil {
				fresh, readErr := write.ReadWebhookPrune(repo)
				state, found := applied.Observation.Find(applied.RecoveryID)
				if readErr == nil && found {
					var next settings.WebhookPruneObservation
					next, readErr = prune.WithCreated(state, fresh)
					if readErr == nil {
						prune = &next
						expected = next.Inventory()
					}
				} else if readErr == nil {
					readErr = fmt.Errorf("created webhook evidence unavailable")
				}
				if readErr != nil {
					i := len(result.Operations)
					result.Operations = append(result.Operations, ApplyOperation{Kind: WebhooksRead})
					result.failOperation(i, OperationFailedRead)
					return wrote, fmt.Errorf("webhook created but prune inventory confirmation failed%s: %w", environmentProgress(*result), readErr)
				}
			}
			continue
		}
		result.Operations = append(result.Operations, ApplyOperation{Kind: WebhookUpdate})
		if ids == nil || prune != nil {
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
			if ids == nil && prune == nil {
				expected, e = expected.WithUpdated(s.Desired)
				if e != nil {
					return fail(index, true, e)
				}
			}
		}
		if prune != nil {
			fresh, readErr := write.ReadWebhookPrune(repo)
			if readErr == nil {
				if changed {
					var next settings.WebhookPruneObservation
					next, readErr = prune.WithUpdated(s.Desired, fresh)
					if readErr == nil {
						prune = &next
						expected = next.Inventory()
					}
				} else {
					readErr = prune.Verify(fresh)
				}
			}
			if readErr != nil {
				i := len(result.Operations)
				result.Operations = append(result.Operations, ApplyOperation{Kind: WebhooksRead})
				result.failOperation(i, OperationFailedRead)
				return wrote, fmt.Errorf("webhook prune inventory confirmation failed%s: %w", environmentProgress(*result), readErr)
			}
		}
	}
	if prune != nil {
		deleted, e := applyWebhookDeletes(result, repo, *prune, write)
		return wrote || deleted, e
	}
	return wrote, nil
}
