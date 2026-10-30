package github

import (
	"encoding/json"

	"github.com/daiksud/gh-settings/internal/settings"
)

func (client *Client) warnRulesetEnforcementReadback(id int64, raw map[string]json.RawMessage) {
	if client.RulesetNormalizationWarning == nil {
		return
	}
	policy, record, err := decodeObservedRulesetPolicy(raw)
	if err == nil && record != (settings.ObservationNormalization{}) {
		client.RulesetNormalizationWarning(id, policy.Name, record)
	}
}
