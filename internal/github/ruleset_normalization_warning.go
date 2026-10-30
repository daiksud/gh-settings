package github

import (
	"encoding/json"

	"github.com/daiksud/gh-settings/internal/settings"
)

func (client *Client) warnRulesetReadbackNormalizations(id int64, raw map[string]json.RawMessage) {
	if client.RulesetNormalizationWarning == nil {
		return
	}
	policy, records, err := decodeObservedRulesetPolicy(raw)
	if err != nil {
		return
	}
	for _, record := range []settings.ObservationNormalization{records.Enforcement, records.QualitySeverity} {
		if record != (settings.ObservationNormalization{}) {
			client.RulesetNormalizationWarning(id, policy.Name, record)
		}
	}
}
