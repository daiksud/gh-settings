package github

import (
	"encoding/json"

	"github.com/daiksud/gh-settings/internal/settings"
)

func (client *Client) warnRulesetReadbackNormalizations(id int64, raw map[string]json.RawMessage) {
	if client.RulesetNormalizationWarning == nil {
		return
	}
	rules, quality, err := prepareRulesetQualityRules(raw["rules"])
	if err != nil {
		return
	}
	projected := make(map[string]json.RawMessage, len(raw))
	for key, value := range raw {
		projected[key] = value
	}
	projected["rules"] = rules
	policy, enforcement, err := decodeObservedRulesetPolicy(projected)
	if err != nil {
		return
	}
	for _, record := range []settings.ObservationNormalization{enforcement, quality} {
		if record != (settings.ObservationNormalization{}) {
			client.RulesetNormalizationWarning(id, policy.Name, record)
		}
	}
}
