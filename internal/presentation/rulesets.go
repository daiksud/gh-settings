package presentation

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// RulesetLine renders selected whole objects, including IDs and fixed defaults.
func RulesetLine(change settings.RulesetDisplayChange) string {
	prior, desired := change.Before(), change.After()
	before := rulesetPolicy(prior)
	after := rulesetPolicy(desired)
	line := fmt.Sprintf("repository.settings.rulesets: %s %q: %s → %s", change.Operation(), change.Name(), before, after)
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
	for _, record := range settings.OrderedRulesetTeamApprovalsNormalizations(records.TeamApprovals) {
		afterChoice := "inactive (rule removed; write required)"
		for _, rule := range *desired.Rules {
			if rule.Type != "pull_request" {
				continue
			}
			afterChoice = "inactive (reviewer removed; write required)"
			for _, reviewer := range *rule.Parameters.RequiredReviewers {
				if reviewer.Reviewer.ID == record.ReviewerID {
					afterChoice = strconv.Itoa(*reviewer.MinimumApprovals) + " (write required)"
					break
				}
			}
			break
		}
		decision := fmt.Sprintf("%s (Ruleset %d %q, reviewer=%d): %d (normalized observation) → %s", record.Path, id, change.Name(), record.ReviewerID, record.Fallback, afterChoice)
		details = append(details, RulesetTeamApprovalsNormalizationLine(id, change.Name(), record), decision)
	}
	for _, record := range settings.OrderedRulesetScanningAlertsNormalizations(records.ScanningAlerts) {
		afterChoice := "inactive (rule removed; write required)"
		for _, rule := range *desired.Rules {
			if rule.Type != "code_scanning" {
				continue
			}
			afterChoice = "inactive (tool removed; write required)"
			for _, tool := range *rule.Parameters.CodeScanningTools {
				if tool.Tool == record.Tool {
					choice, _ := settings.RulesetScanningAlertsChoiceForValue(tool.AlertsThreshold)
					afterChoice = strconv.Quote(string(choice)) + " (write required)"
					break
				}
			}
			break
		}
		decision := fmt.Sprintf("%s (Ruleset %d %q, tool=%q): %q (normalized observation) → %s", record.Path, id, change.Name(), record.Tool, record.Fallback, afterChoice)
		details = append(details, RulesetScanningAlertsNormalizationLine(id, change.Name(), record), decision)
	}
	for _, record := range settings.OrderedRulesetScanningSecurityAlertsNormalizations(records.ScanningSecurityAlerts) {
		afterChoice := "inactive (rule removed; write required)"
		for _, rule := range *desired.Rules {
			if rule.Type != "code_scanning" {
				continue
			}
			afterChoice = "inactive (tool removed; write required)"
			for _, tool := range *rule.Parameters.CodeScanningTools {
				if tool.Tool == record.Tool {
					choice, _ := settings.RulesetScanningSecurityAlertsChoiceForValue(tool.SecurityAlertsThreshold)
					afterChoice = strconv.Quote(string(choice)) + " (write required)"
					break
				}
			}
			break
		}
		decision := fmt.Sprintf("%s (Ruleset %d %q, tool=%q): %q (normalized observation) → %s", record.Path, id, change.Name(), record.Tool, record.Fallback, afterChoice)
		details = append(details, RulesetScanningSecurityAlertsNormalizationLine(id, change.Name(), record), decision)
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
	actor, err := github.RulesetBypassIdentityUI(record.ActorType, record.ActorID)
	if err != nil {
		return "actor unavailable in supported UI choices"
	}
	encoded, err := workspace.MarshalUIReport(actor)
	if err != nil {
		return "actor unavailable in supported UI choices"
	}
	return string(encoded)
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

func RulesetTeamApprovalsNormalizationLine(id int64, name string, record settings.RulesetTeamApprovalsNormalization) string {
	return fmt.Sprintf("Warning: %s: observed %d is outside UI choices; using %d (projection only). Ruleset %d %q, reviewer=%d.", record.Path, record.RawValue, record.Fallback, id, name, record.ReviewerID)
}

func RulesetScanningAlertsNormalizationLine(id int64, name string, record settings.RulesetScanningAlertsNormalization) string {
	return fmt.Sprintf("Warning: %s: observed %q is outside UI choices; using %q (projection only). Ruleset %d %q, tool=%q.", record.Path, record.RawValue, record.Fallback, id, name, record.Tool)
}

func RulesetScanningSecurityAlertsNormalizationLine(id int64, name string, record settings.RulesetScanningSecurityAlertsNormalization) string {
	return fmt.Sprintf("Warning: %s: observed %q is outside UI choices; using %q (projection only). Ruleset %d %q, tool=%q.", record.Path, record.RawValue, record.Fallback, id, name, record.Tool)
}

// rulesetPolicy never substitutes a partial or unrepresentable API object for
// a UI policy. A nil value denotes the absent side of a create operation.
func rulesetPolicy(policy *settings.RulesetDeclaration) string {
	if policy == nil {
		return "absent"
	}
	projected, err := github.RulesetObservation(*policy)
	if err != nil {
		return "unavailable (policy is outside supported UI choices)"
	}
	encoded, err := workspace.MarshalUIReport(projected.Ruleset)
	if err != nil {
		return "unavailable (policy cannot be rendered)"
	}
	return string(encoded)
}

// ValidateRulesetDisplay rejects a selected policy that cannot be projected to
// supported UI controls. Never replace an unsupported observation with an
// apparently complete or equal policy.
func ValidateRulesetDisplay(change settings.RulesetDisplayChange) error {
	for _, policy := range []*settings.RulesetDeclaration{change.Before(), change.After()} {
		if policy == nil {
			continue
		}
		if _, err := github.RulesetObservation(*policy); err != nil {
			return fmt.Errorf("selected Ruleset policy cannot be represented using supported UI choices")
		}
	}
	return nil
}

func ValidateRulesetChanges(changes []settings.RulesetChange) error {
	for _, change := range changes {
		if err := ValidateRulesetDisplay(change.Display()); err != nil {
			return err
		}
	}
	return nil
}
