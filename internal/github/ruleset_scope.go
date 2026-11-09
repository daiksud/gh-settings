package github

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Check raw presence/types before observation normalization can project values.
func rejectUnsupportedRulesetScope(raw map[string]json.RawMessage) error {
	var rules []map[string]json.RawMessage
	if err := requiredRuleJSON(raw, "rules", &rules); err != nil {
		return err
	}
	for _, rule := range rules {
		var kind string
		if err := requiredRuleJSON(rule, "type", &kind); err != nil {
			return err
		}
		if kind == "code_coverage" {
			return fmt.Errorf("unsupported Ruleset rule %q outside Free/public non-Preview scope", kind)
		}
		if kind != "pull_request" {
			continue
		}
		var parameters map[string]json.RawMessage
		if err := requiredRuleJSON(rule, "parameters", &parameters); err != nil {
			return err
		}
		var preview bool
		if err := requiredRuleJSON(parameters, "require_extra_approval_for_unattributed_changes", &preview); err != nil {
			return err
		}
		var teams []json.RawMessage
		if err := requiredRuleJSON(parameters, "required_reviewers", &teams); err != nil {
			return err
		}
		if teams == nil || len(teams) != 0 {
			return fmt.Errorf("unsupported team required-reviewer policy")
		}
	}
	return nil
}

func (client *Client) preflightRulesetUnmanaged(repo repository.Repository, change settings.RulesetChange) error {
	var raw map[string]json.RawMessage
	if err := client.rest.Get(rulesetsPath(repo)+"/"+strconv.FormatInt(change.ID(), 10)+"?includes_parents=false", &raw); err != nil {
		return fmt.Errorf("read unowned Ruleset companions before update: %w", err)
	}
	owned, err := rulesetOwner(raw, repo)
	if err != nil || !owned {
		return fmt.Errorf("cannot establish repository ownership before Ruleset update: %v", err)
	}
	var id int64
	if err := requiredRuleJSON(raw, "id", &id); err != nil || id != change.ID() {
		return fmt.Errorf("Ruleset preflight ID mismatch")
	}
	current, _, err := decodeObservedRulesetPolicy(raw)
	if err != nil {
		return fmt.Errorf("unsupported or incomplete Ruleset preflight; no update sent: %w", err)
	}
	if !reflect.DeepEqual(settings.RulesetUnownedCodeQuality(&current), settings.RulesetUnownedCodeQuality(change.Before())) {
		return fmt.Errorf("unowned Code Quality rule changed before update; no update sent")
	}
	actual, present := settings.RulesetPreviewApproval(&current)
	expected, wasPresent := settings.RulesetPreviewApproval(change.Before())
	if present != wasPresent || !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("unowned PR additional-approval value changed before update; no update sent")
	}
	return nil
}

// A closed, known API shape can be retained without supporting public intent.
// Unknown members, duplicate rules and malformed values cannot be replayed safely.
func splitUnownedCodeQuality(rules []map[string]json.RawMessage) ([]map[string]json.RawMessage, *string, error) {
	owned := make([]map[string]json.RawMessage, 0, len(rules))
	var severity *string
	for _, rule := range rules {
		var kind string
		if err := requiredRuleJSON(rule, "type", &kind); err != nil {
			return nil, nil, err
		}
		if kind != "code_quality" {
			owned = append(owned, rule)
			continue
		}
		if severity != nil || len(rule) != 2 {
			return nil, nil, fmt.Errorf("duplicate or unknown unowned Code Quality rule members")
		}
		var params map[string]json.RawMessage
		if err := requiredRuleJSON(rule, "parameters", &params); err != nil {
			return nil, nil, err
		}
		if len(params) != 1 {
			return nil, nil, fmt.Errorf("unknown or missing unowned Code Quality parameters")
		}
		var value string
		if err := requiredRuleJSON(params, "severity", &value); err != nil {
			return nil, nil, err
		}
		severity = &value
	}
	return owned, severity, nil
}
