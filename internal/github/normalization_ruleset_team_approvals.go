package github

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// prepareRulesetTeamApprovalsRules projects typed counts for existing Team IDs.
// Complete policy validation must still gate publication of these records.
func prepareRulesetTeamApprovalsRules(raw json.RawMessage) (json.RawMessage, []settings.RulesetTeamApprovalsNormalization, error) {
	var rules []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, nil, err
	}
	if rules == nil {
		return nil, nil, fmt.Errorf("Rulesets response requires a non-null rules array")
	}
	var records []settings.RulesetTeamApprovalsNormalization
	seenRule := false
	for _, rule := range rules {
		var kind string
		if err := requiredRuleJSON(rule, "type", &kind); err != nil {
			return nil, nil, err
		}
		if kind != "pull_request" {
			continue
		}
		if seenRule {
			return nil, nil, fmt.Errorf("duplicate pull_request rule")
		}
		seenRule = true
		var parameters map[string]json.RawMessage
		if err := requiredRuleJSON(rule, "parameters", &parameters); err != nil {
			return nil, nil, err
		}
		var reviewers []map[string]json.RawMessage
		if err := requiredRuleJSON(parameters, "required_reviewers", &reviewers); err != nil {
			return nil, nil, err
		}
		if len(reviewers) > 15 {
			return nil, nil, fmt.Errorf("at most 15 required reviewers are supported")
		}
		seen := map[int64]bool{}
		changed := false
		for _, reviewer := range reviewers {
			var identity map[string]json.RawMessage
			if err := requiredRuleJSON(reviewer, "reviewer", &identity); err != nil {
				return nil, nil, err
			}
			var actorType string
			var id int64
			if err := requiredRuleJSON(identity, "type", &actorType); err != nil {
				return nil, nil, err
			}
			if err := requiredRuleJSON(identity, "id", &id); err != nil {
				return nil, nil, err
			}
			if actorType != "Team" || id <= 0 || seen[id] {
				return nil, nil, fmt.Errorf("invalid or duplicate required Team reviewer")
			}
			seen[id] = true
			var count int
			if err := requiredRuleJSON(reviewer, "minimum_approvals", &count); err != nil {
				return nil, nil, err
			}
			if settings.RulesetTeamApprovalsSupported(count) {
				continue
			}
			fallback := settings.DefaultRulesetTeamApprovals
			encoded, err := json.Marshal(fallback)
			if err != nil {
				return nil, nil, err
			}
			reviewer["minimum_approvals"] = encoded
			records = append(records, settings.RulesetTeamApprovalsNormalization{ReviewerID: id, Path: settings.RulesetTeamApprovalsPath, RawValue: count, Fallback: fallback})
			changed = true
		}
		if changed {
			encoded, err := json.Marshal(reviewers)
			if err != nil {
				return nil, nil, err
			}
			parameters["required_reviewers"] = encoded
			encoded, err = json.Marshal(parameters)
			if err != nil {
				return nil, nil, err
			}
			rule["parameters"] = encoded
		}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return nil, nil, err
	}
	return encoded, records, nil
}
