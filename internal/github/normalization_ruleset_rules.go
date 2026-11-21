package github

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

type ruleProjection struct {
	kind      string
	normalize func(map[string]json.RawMessage, *settings.RulesetNormalizations) (bool, error)
}

var (
	qualitySeverityProjection        = ruleProjection{"code_quality", normalizeQualitySeverity}
	requiredApprovalsProjection      = ruleProjection{"pull_request", normalizeRequiredApprovals}
	mergeMethodsProjection           = ruleProjection{"pull_request", normalizeMergeMethods}
	scanningAlertsProjection         = ruleProjection{"code_scanning", normalizeScanningAlerts}
	scanningSecurityAlertsProjection = ruleProjection{"code_scanning", normalizeScanningSecurityAlerts}
)

// Decode once into a private copy. Each projection checks its field across all
// rules before the next field is inspected. Return evidence only on success.
func projectObservedRules(raw json.RawMessage, projections ...ruleProjection) (json.RawMessage, settings.RulesetNormalizations, error) {
	var rules []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, settings.RulesetNormalizations{}, err
	}
	if rules == nil {
		return nil, settings.RulesetNormalizations{}, fmt.Errorf("Rulesets response requires a non-null rules array")
	}
	var records settings.RulesetNormalizations
	for _, projection := range projections {
		if err := projection.apply(rules, &records); err != nil {
			return nil, settings.RulesetNormalizations{}, err
		}
	}
	encoded, err := json.Marshal(rules)
	if err != nil {
		return nil, settings.RulesetNormalizations{}, err
	}
	return encoded, records, nil
}

func (projection ruleProjection) apply(rules []map[string]json.RawMessage, records *settings.RulesetNormalizations) error {
	seen := false
	for _, rule := range rules {
		var kind string
		if err := requiredRuleJSON(rule, "type", &kind); err != nil {
			return err
		}
		if kind != projection.kind {
			continue
		}
		if seen {
			return fmt.Errorf("duplicate %s rule", kind)
		}
		seen = true
		var parameters map[string]json.RawMessage
		if err := requiredRuleJSON(rule, "parameters", &parameters); err != nil {
			return err
		}
		changed, err := projection.normalize(parameters, records)
		if err != nil {
			return err
		}
		if changed {
			if err := setRuleJSON(rule, "parameters", parameters); err != nil {
				return err
			}
		}
	}
	return nil
}

func setRuleJSON(fields map[string]json.RawMessage, key string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	fields[key] = encoded
	return nil
}

func normalizeQualitySeverity(parameters map[string]json.RawMessage, records *settings.RulesetNormalizations) (bool, error) {
	var severity string
	if err := requiredRuleJSON(parameters, "severity", &severity); err != nil {
		return false, err
	}
	if _, supported := settings.RulesetQualitySeverityChoiceForValue(severity); supported {
		return false, nil
	}
	fallback, _ := settings.RulesetQualitySeverityValueForChoice(settings.DefaultRulesetQualitySeverityChoice)
	records.QualitySeverity = settings.ObservationNormalization{
		Path: settings.RulesetQualitySeverityPath, RawValue: severity, Fallback: string(settings.DefaultRulesetQualitySeverityChoice),
	}
	return true, setRuleJSON(parameters, "severity", fallback)
}

func normalizeRequiredApprovals(parameters map[string]json.RawMessage, records *settings.RulesetNormalizations) (bool, error) {
	var count int
	if err := requiredRuleJSON(parameters, "required_approving_review_count", &count); err != nil {
		return false, err
	}
	if settings.RulesetRequiredApprovalsSupported(count) {
		return false, nil
	}
	fallback := settings.DefaultRulesetRequiredApprovals
	records.RequiredApprovals = settings.RulesetRequiredApprovalsNormalization{
		Path: settings.RulesetRequiredApprovalsPath, RawValue: count, Fallback: fallback,
	}
	return true, setRuleJSON(parameters, "required_approving_review_count", fallback)
}

func normalizeMergeMethods(parameters map[string]json.RawMessage, records *settings.RulesetNormalizations) (bool, error) {
	var items []json.RawMessage
	if err := requiredRuleJSON(parameters, "allowed_merge_methods", &items); err != nil {
		return false, err
	}
	if len(items) == 0 {
		return false, fmt.Errorf("allowed_merge_methods requires a nonempty string list")
	}
	methods := make([]string, 0, len(items))
	seen := map[string]bool{}
	unsupported := false
	for _, item := range items {
		var method string
		// encoding/json otherwise accepts null as the string zero value.
		if bytes.Equal(bytes.TrimSpace(item), []byte("null")) {
			return false, fmt.Errorf("allowed_merge_methods contains null")
		}
		if err := json.Unmarshal(item, &method); err != nil {
			return false, err
		}
		if seen[method] {
			return false, fmt.Errorf("duplicate allowed_merge_methods value %q", method)
		}
		seen[method] = true
		methods = append(methods, method)
		if _, supported := settings.RulesetMergeMethodChoiceForValue(method); !supported {
			unsupported = true
		}
	}
	if !unsupported {
		return false, nil
	}
	fallback := settings.DefaultRulesetAllowedMergeMethods()
	records.MergeMethods = settings.RulesetMergeMethodsNormalization{
		Path: settings.RulesetMergeMethodsPath, RawValue: methods, Fallback: fallback,
	}
	return true, setRuleJSON(parameters, "allowed_merge_methods", fallback)
}
