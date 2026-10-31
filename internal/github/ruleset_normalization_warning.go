package github

import (
	"encoding/json"
	"slices"

	"github.com/daiksud/gh-settings/internal/settings"
)

func (client *Client) warnRulesetReadbackNormalizations(id int64, raw map[string]json.RawMessage) {
	if client.RulesetNormalizationWarning == nil && client.RulesetRequiredApprovalsNormalizationWarning == nil && client.RulesetBypassModeNormalizationWarning == nil && client.RulesetMergeMethodsNormalizationWarning == nil && client.RulesetTeamApprovalsNormalizationWarning == nil && client.RulesetScanningAlertsNormalizationWarning == nil && client.RulesetScanningSecurityAlertsNormalizationWarning == nil {
		return
	}
	policy, records, err := decodeObservedRulesetPolicy(raw)
	if err != nil {
		return
	}
	client.warnRulesetNormalizations(id, policy.Name, records)
}

func (client *Client) warnRulesetNormalizations(id int64, name string, records settings.RulesetNormalizations) {
	if record := records.Enforcement; record != (settings.ObservationNormalization{}) && client.RulesetNormalizationWarning != nil {
		client.RulesetNormalizationWarning(id, name, record)
	}
	if record := records.RequiredApprovals; record != (settings.RulesetRequiredApprovalsNormalization{}) && client.RulesetRequiredApprovalsNormalizationWarning != nil {
		client.RulesetRequiredApprovalsNormalizationWarning(id, name, record)
	}
	if record := records.MergeMethods; !record.IsZero() && client.RulesetMergeMethodsNormalizationWarning != nil {
		record.RawValue, record.Fallback = slices.Clone(record.RawValue), slices.Clone(record.Fallback)
		client.RulesetMergeMethodsNormalizationWarning(id, name, record)
	}
	if client.RulesetTeamApprovalsNormalizationWarning != nil {
		for _, record := range settings.OrderedRulesetTeamApprovalsNormalizations(records.TeamApprovals) {
			client.RulesetTeamApprovalsNormalizationWarning(id, name, record)
		}
	}
	if client.RulesetScanningAlertsNormalizationWarning != nil {
		for _, record := range settings.OrderedRulesetScanningAlertsNormalizations(records.ScanningAlerts) {
			client.RulesetScanningAlertsNormalizationWarning(id, name, record)
		}
	}
	if client.RulesetScanningSecurityAlertsNormalizationWarning != nil {
		for _, record := range settings.OrderedRulesetScanningSecurityAlertsNormalizations(records.ScanningSecurityAlerts) {
			client.RulesetScanningSecurityAlertsNormalizationWarning(id, name, record)
		}
	}
	if record := records.QualitySeverity; record != (settings.ObservationNormalization{}) && client.RulesetNormalizationWarning != nil {
		client.RulesetNormalizationWarning(id, name, record)
	}
	if client.RulesetBypassModeNormalizationWarning != nil {
		for _, record := range settings.OrderedRulesetBypassModeNormalizations(records.BypassModes) {
			client.RulesetBypassModeNormalizationWarning(id, name, record)
		}
	}
}
