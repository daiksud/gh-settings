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
		if kind != "pull_request" {
			continue
		}
		var parameters map[string]json.RawMessage
		if err := requiredRuleJSON(rule, "parameters", &parameters); err != nil {
			return err
		}
		var additionalApproval bool
		if err := requiredRuleJSON(parameters, "require_extra_approval_for_unattributed_changes", &additionalApproval); err != nil {
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

func (client *Client) preflightRulesetUpdate(repo repository.Repository, change settings.RulesetChange) error {
	var raw map[string]json.RawMessage
	if err := client.rest.Get(rulesetsPath(repo)+"/"+strconv.FormatInt(change.ID(), 10)+"?includes_parents=false", &raw); err != nil {
		return fmt.Errorf("read Ruleset policy before update: %w", err)
	}
	owned, err := rulesetOwner(raw, repo)
	if err != nil || !owned {
		return fmt.Errorf("cannot establish repository ownership before Ruleset update: %v", err)
	}
	var id int64
	if err := requiredRuleJSON(raw, "id", &id); err != nil || id != change.ID() {
		return fmt.Errorf("Ruleset preflight ID mismatch")
	}
	current, records, err := decodeObservedRulesetPolicy(raw)
	if err != nil {
		return fmt.Errorf("unsupported or incomplete Ruleset preflight; no update sent: %w", err)
	}
	// Code Quality is managed again, but whole-object writes must not overwrite
	// a rule added or changed since acquisition, including normalized raw values.
	if !reflect.DeepEqual(rulesetQualitySnapshot(&current, records.QualitySeverity), rulesetQualitySnapshot(change.Before(), change.Normalizations().QualitySeverity)) {
		return fmt.Errorf("Code Quality rule changed before update; no update sent")
	}
	if !reflect.DeepEqual(rulesetCoverageSnapshot(&current), rulesetCoverageSnapshot(change.Before())) {
		return fmt.Errorf("Code Coverage rule changed before update; no update sent")
	}
	actual, present := settings.RulesetAdditionalApproval(&current)
	expected, wasPresent := settings.RulesetAdditionalApproval(change.Before())
	if present != wasPresent || !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("PR additional-approval value changed before update; no update sent")
	}
	return nil
}

// A newly acquired rule must not erase coverage added or edited since selection.
// Nil is absent; a present zero pair is a selected rule with disabled thresholds.
func rulesetCoverageSnapshot(policy *settings.RulesetDeclaration) *[2]float64 {
	if policy != nil && policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Type == "code_coverage" {
				return &[2]float64{*rule.Parameters.MinimumCoverage, *rule.Parameters.MaxCoverageDrop}
			}
		}
	}
	return nil
}

func rulesetQualitySnapshot(policy *settings.RulesetDeclaration, record settings.ObservationNormalization) *string {
	if policy != nil && policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Type == "code_quality" && rule.Parameters != nil && rule.Parameters.Severity != nil {
				value := *rule.Parameters.Severity
				if record != (settings.ObservationNormalization{}) {
					value = record.RawValue
				}
				return &value
			}
		}
	}
	return nil
}
