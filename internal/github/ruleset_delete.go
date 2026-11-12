package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type rulesetDeletePreWriteFailure struct{ cause error }

func (e *rulesetDeletePreWriteFailure) Error() string        { return e.cause.Error() }
func (e *rulesetDeletePreWriteFailure) Unwrap() error        { return e.cause }
func (*rulesetDeletePreWriteFailure) ReadFailedBeforeWrite() {}

func (client *Client) deleteRuleset(repo repository.Repository, change settings.RulesetChange) (resultID int64, err error) {
	sent := false
	defer func() {
		if err != nil && !sent {
			err = &rulesetDeletePreWriteFailure{cause: err}
		}
	}()
	before := change.Before()
	if change.ID() <= 0 || before == nil || before.ID == nil || *before.ID != change.ID() || before.Delete != nil || change.After() != nil {
		return 0, fmt.Errorf("invalid Ruleset deletion")
	}
	if err := settings.ValidateRuleset(*before, true); err != nil {
		return 0, err
	}
	path := fmt.Sprintf("%s/%d", rulesetsPath(repo), change.ID())
	var raw map[string]json.RawMessage
	if err := client.rest.Get(path+"?includes_parents=false", &raw); err != nil {
		return 0, fmt.Errorf("recheck selected Ruleset before deletion: %w", err)
	}
	owned, err := rulesetOwner(raw, repo)
	if err != nil || !owned {
		return 0, fmt.Errorf("selected Ruleset deletion ownership unavailable or changed: %v", err)
	}
	var id int64
	if err := requiredRuleJSON(raw, "id", &id); err != nil || id != change.ID() {
		return 0, fmt.Errorf("selected Ruleset deletion identity changed")
	}
	policy, err := decodeRulesetPolicy(raw)
	if err != nil {
		return 0, fmt.Errorf("recheck complete Ruleset policy before deletion: %w", err)
	}
	policy.ID = before.ID
	// Compare the complete raw policy through domain-owned canonical ordering.
	// No declaration defaults or observation normalization authorize deletion.
	if !change.MatchesDeletionPolicy(policy) {
		return 0, fmt.Errorf("selected Ruleset policy changed before deletion")
	}
	sent = true
	response, err := client.rest.Request(http.MethodDelete, path, nil)
	if err != nil {
		return 0, fmt.Errorf("delete Ruleset (write may have applied): %w", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		return 0, fmt.Errorf("unexpected Ruleset delete status %d (write may have applied)", response.StatusCode)
	}
	fresh, err := client.ReadRulesets(repo)
	if err != nil {
		return 0, fmt.Errorf("verify Ruleset deletion inventory (write may have applied): %w", err)
	}
	states, err := fresh.Values()
	if err != nil {
		return 0, fmt.Errorf("verify Ruleset deletion inventory (write may have applied): %w", err)
	}
	if fresh.HasUnmanagedID(id) {
		return 0, fmt.Errorf("Ruleset deletion inventory retained selected ID outside ownership (write may have applied)")
	}
	for _, state := range states {
		if state.ID == id {
			return 0, fmt.Errorf("Ruleset deletion inventory retained selected ID (write may have applied)")
		}
	}
	return id, nil
}
