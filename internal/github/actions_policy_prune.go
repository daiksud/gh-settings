package github

import (
	"fmt"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func (client *Client) deleteActionsEventPolicy(repo repository.Repository, change settings.ActionsEventPolicyChange) (int64, error) {
	before := change.Before()
	if change.Operation() != "delete" || before == nil || change.ID() <= 0 {
		return 0, &actionsPoliciesPreWriteFailure{fmt.Errorf("invalid Actions policy deletion")}
	}
	if _, err := settings.NewActionsEventPolicyDelete(*before); err != nil {
		return 0, &actionsPoliciesPreWriteFailure{err}
	}
	current, err := client.readActionsEventPolicy(repo, change.ID())
	if err != nil {
		return 0, &actionsPoliciesPreWriteFailure{err}
	}
	if current.Policy == nil || !equalActionsEventPolicy(*current.Policy, *before) {
		return 0, &actionsPoliciesPreWriteFailure{fmt.Errorf("Actions policy identity/state changed or unsupported; refusing deletion")}
	}
	response, err := client.rest.Request(http.MethodDelete, fmt.Sprintf("%s/%d", actionsPoliciesPath(repo), change.ID()), nil)
	if err != nil {
		return 0, fmt.Errorf("delete Actions policy (write may have applied): %w", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return 0, fmt.Errorf("unexpected Actions policy delete status %d (write may have applied)", response.StatusCode)
	}
	fresh, err := client.ReadActionsPolicies(repo)
	if err != nil {
		return 0, fmt.Errorf("verify Actions policy deletion (write may have applied): %w", err)
	}
	values, err := fresh.Values()
	if err != nil {
		return 0, err
	}
	for _, value := range values {
		if value.ID == change.ID() {
			return 0, fmt.Errorf("Actions policy deletion inventory retained selected ID (write may have applied)")
		}
	}
	return change.ID(), nil
}
