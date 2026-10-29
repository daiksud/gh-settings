package github

import (
	"encoding/json"

	"github.com/daiksud/gh-settings/internal/settings"
)

func (client *Client) warnRulesetReadbackNormalizations(id int64, raw map[string]json.RawMessage) {
	if client.RulesetNormalizationWarning == nil && client.RulesetRequiredApprovalsNormalizationWarning == nil {
		return
	}
	rules, approvals, err := prepareRulesetRequiredApprovalsRules(raw["rules"])
	if err != nil {
		return
	}
	projected := make(map[string]json.RawMessage, len(raw))
	for key, value := range raw {
		projected[key] = value
	}
	projected["rules"] = rules
	policy, records, err := decodeObservedRulesetPolicy(projected)
	if err != nil {
		return
	}
	records.RequiredApprovals = approvals
	client.warnRulesetNormalizations(id, policy.Name, records)
}

func (client *Client) warnRulesetNormalizations(id int64, name string, records settings.RulesetNormalizations) {
	if record := records.Enforcement; record != (settings.ObservationNormalization{}) && client.RulesetNormalizationWarning != nil {
		client.RulesetNormalizationWarning(id, name, record)
	}
	if record := records.RequiredApprovals; record != (settings.RulesetRequiredApprovalsNormalization{}) && client.RulesetRequiredApprovalsNormalizationWarning != nil {
		client.RulesetRequiredApprovalsNormalizationWarning(id, name, record)
	}
	if record := records.QualitySeverity; record != (settings.ObservationNormalization{}) && client.RulesetNormalizationWarning != nil {
		client.RulesetNormalizationWarning(id, name, record)
	}
}
