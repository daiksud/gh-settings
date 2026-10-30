package presentation

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

// RulesetLine renders selected whole objects, including IDs and fixed defaults.
func RulesetLine(change settings.RulesetDisplayChange) string {
	prior, desired := change.Before(), change.After()
	before, _ := json.Marshal(prior)
	after, _ := json.Marshal(desired)
	line := fmt.Sprintf("settings.rulesets: %s %q: %s → %s", change.Operation(), change.Name(), before, after)
	records := change.Normalizations()
	if records.IsZero() {
		return line
	}
	id := *prior.ID
	var details []string
	if record := records.Enforcement; record != (settings.ObservationNormalization{}) {
		decision := fmt.Sprintf("%s (Ruleset %d %q): %s (normalized observation) → %s (write required)", record.Path, id, change.Name(), strconv.Quote(record.Fallback), strconv.Quote(*desired.Enforcement))
		details = append(details, RulesetNormalizationLine(id, change.Name(), record), decision)
	}
	if record := records.RequiredApprovals; record != (settings.RulesetRequiredApprovalsNormalization{}) {
		afterChoice := "inactive (rule removed; write required)"
		for _, rule := range *desired.Rules {
			if rule.Type == "pull_request" {
				afterChoice = strconv.Itoa(*rule.Parameters.RequiredApprovingReviewCount) + " (write required)"
				break
			}
		}
		decision := fmt.Sprintf("%s (Ruleset %d %q): %d (normalized observation) → %s", record.Path, id, change.Name(), record.Fallback, afterChoice)
		details = append(details, RulesetRequiredApprovalsNormalizationLine(id, change.Name(), record), decision)
	}
	if record := records.MergeMethods; !record.IsZero() {
		afterChoice := "inactive (rule removed; write required)"
		for _, rule := range *desired.Rules {
			if rule.Type == "pull_request" {
				var ordered []string
				for _, method := range settings.DefaultRulesetAllowedMergeMethods() {
					if slices.Contains(*rule.Parameters.AllowedMergeMethods, method) {
						ordered = append(ordered, method)
					}
				}
				encoded, _ := json.Marshal(ordered)
				afterChoice = string(encoded) + " (write required)"
				break
			}
		}
		fallback, _ := json.Marshal(record.Fallback)
		decision := fmt.Sprintf("%s (Ruleset %d %q): %s (normalized observation) → %s", record.Path, id, change.Name(), fallback, afterChoice)
		details = append(details, RulesetMergeMethodsNormalizationLine(id, change.Name(), record), decision)
	}
	if record := records.QualitySeverity; record != (settings.ObservationNormalization{}) {
		afterChoice := "inactive (rule removed; write required)"
		for _, rule := range *desired.Rules {
			if rule.Type == "code_quality" {
				choice, _ := settings.RulesetQualitySeverityChoiceForValue(*rule.Parameters.Severity)
				afterChoice = strconv.Quote(string(choice)) + " (write required)"
				break
			}
		}
		decision := fmt.Sprintf("%s (Ruleset %d %q): %s (normalized observation) → %s", record.Path, id, change.Name(), strconv.Quote(record.Fallback), afterChoice)
		details = append(details, RulesetNormalizationLine(id, change.Name(), record), decision)
	}
	for _, record := range settings.OrderedRulesetBypassModeNormalizations(records.BypassModes) {
		afterChoice := "inactive (actor removed; write required)"
		for _, actor := range *desired.BypassActors {
			sameID := actor.ActorID == nil && record.ActorID == nil || actor.ActorID != nil && record.ActorID != nil && *actor.ActorID == *record.ActorID
			if actor.ActorType == record.ActorType && sameID {
				choice, _ := settings.RulesetBypassModeChoiceForValue(*actor.BypassMode)
				afterChoice = strconv.Quote(string(choice)) + " (write required)"
				break
			}
		}
		decision := fmt.Sprintf("%s (Ruleset %d %q, %s): %s (normalized observation) → %s", record.Normalization.Path, id, change.Name(), rulesetBypassActorLabel(record), strconv.Quote(record.Normalization.Fallback), afterChoice)
		details = append(details, RulesetBypassNormalizationLine(id, change.Name(), record), decision)
	}
	return strings.Join(details, "\n") + "\nEffective policy: " + line
}

func rulesetBypassActorLabel(record settings.RulesetBypassModeNormalization) string {
	label := fmt.Sprintf("actor_type=%q", record.ActorType)
	if record.ActorID != nil {
		label += fmt.Sprintf(", actor_id=%d", *record.ActorID)
	}
	return label
}

func RulesetBypassNormalizationLine(id int64, name string, record settings.RulesetBypassModeNormalization) string {
	return fmt.Sprintf("%s Ruleset %d %q, %s.", NormalizationLine(record.Normalization), id, name, rulesetBypassActorLabel(record))
}

func RulesetNormalizationLine(id int64, name string, record settings.ObservationNormalization) string {
	return fmt.Sprintf("%s Ruleset %d %q.", NormalizationLine(record), id, name)
}

func RulesetRequiredApprovalsNormalizationLine(id int64, name string, record settings.RulesetRequiredApprovalsNormalization) string {
	return fmt.Sprintf("Warning: %s: observed %d is outside UI choices; using %d (projection only). Ruleset %d %q.", record.Path, record.RawValue, record.Fallback, id, name)
}

func RulesetMergeMethodsNormalizationLine(id int64, name string, record settings.RulesetMergeMethodsNormalization) string {
	raw, _ := json.Marshal(record.RawValue)
	fallback, _ := json.Marshal(record.Fallback)
	return fmt.Sprintf("Warning: %s: observed %s is outside UI choices; using %s (projection only). Ruleset %d %q.", record.Path, raw, fallback, id, name)
}
