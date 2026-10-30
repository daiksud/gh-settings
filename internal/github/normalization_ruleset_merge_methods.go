package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// prepareRulesetMergeMethodsRules projects an acquired list; the caller must
// still validate the complete policy before publishing any observation.
func prepareRulesetMergeMethodsRules(raw json.RawMessage) (json.RawMessage, settings.RulesetMergeMethodsNormalization, error) {
	var rules []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, settings.RulesetMergeMethodsNormalization{}, err
	}
	if rules == nil {
		return nil, settings.RulesetMergeMethodsNormalization{}, fmt.Errorf("Rulesets response requires a non-null rules array")
	}
	record := settings.RulesetMergeMethodsNormalization{}
	seenRule := false
	for _, rule := range rules {
		var kind string
		if err := requiredRuleJSON(rule, "type", &kind); err != nil {
			return nil, settings.RulesetMergeMethodsNormalization{}, err
		}
		if kind != "pull_request" {
			continue
		}
		if seenRule {
			return nil, settings.RulesetMergeMethodsNormalization{}, fmt.Errorf("duplicate pull_request rule")
		}
		seenRule = true
		var parameters map[string]json.RawMessage
		if err := requiredRuleJSON(rule, "parameters", &parameters); err != nil {
			return nil, settings.RulesetMergeMethodsNormalization{}, err
		}
		var items []json.RawMessage
		if err := requiredRuleJSON(parameters, "allowed_merge_methods", &items); err != nil {
			return nil, settings.RulesetMergeMethodsNormalization{}, err
		}
		if len(items) == 0 {
			return nil, settings.RulesetMergeMethodsNormalization{}, fmt.Errorf("allowed_merge_methods requires a nonempty string list")
		}
		methods := make([]string, 0, len(items))
		seen := map[string]bool{}
		unsupported := false
		for _, item := range items {
			var method string
			// encoding/json otherwise accepts null as the string zero value.
			if bytes.Equal(bytes.TrimSpace(item), []byte("null")) {
				return nil, settings.RulesetMergeMethodsNormalization{}, fmt.Errorf("allowed_merge_methods contains null")
			}
			if err := json.Unmarshal(item, &method); err != nil {
				return nil, settings.RulesetMergeMethodsNormalization{}, err
			}
			if seen[method] {
				return nil, settings.RulesetMergeMethodsNormalization{}, fmt.Errorf("duplicate allowed_merge_methods value %q", method)
			}
			seen[method] = true
			methods = append(methods, method)
			if _, supported := settings.RulesetMergeMethodChoiceForValue(method); !supported {
				unsupported = true
			}
		}
		if !unsupported {
			continue
		}
		fallback := settings.DefaultRulesetAllowedMergeMethods()
		encoded, err := json.Marshal(fallback)
		if err != nil {
			return nil, settings.RulesetMergeMethodsNormalization{}, err
		}
		parameters["allowed_merge_methods"] = encoded
		encoded, err = json.Marshal(parameters)
		if err != nil {
			return nil, settings.RulesetMergeMethodsNormalization{}, err
		}
		rule["parameters"] = encoded
		record = settings.RulesetMergeMethodsNormalization{Path: settings.RulesetMergeMethodsPath, RawValue: methods, Fallback: fallback}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return nil, settings.RulesetMergeMethodsNormalization{}, err
	}
	return encoded, record, nil
}
