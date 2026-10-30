package settings

import (
	"fmt"
	"slices"
)

// RulesetNormalizations owns the concrete observation records independently.
// It is validation/presentation evidence and never a writable policy payload.
type RulesetNormalizations struct {
	Enforcement       ObservationNormalization
	QualitySeverity   ObservationNormalization
	RequiredApprovals RulesetRequiredApprovalsNormalization
	MergeMethods      RulesetMergeMethodsNormalization
	TeamApprovals     []RulesetTeamApprovalsNormalization
	BypassModes       []RulesetBypassModeNormalization
}

// IsZero reports whether the aggregate contains no observation evidence.
func (records RulesetNormalizations) IsZero() bool {
	return records.Enforcement == (ObservationNormalization{}) && records.QualitySeverity == (ObservationNormalization{}) && records.RequiredApprovals == (RulesetRequiredApprovalsNormalization{}) && records.MergeMethods.IsZero() && len(records.TeamApprovals) == 0 && len(records.BypassModes) == 0
}

func (c RulesetChange) Normalizations() RulesetNormalizations {
	return cloneRulesetNormalizations(c.normalizations)
}
func (c RulesetDisplayChange) Normalizations() RulesetNormalizations {
	return cloneRulesetNormalizations(c.normalizations)
}

func cloneRulesetNormalizations(records RulesetNormalizations) RulesetNormalizations {
	records.MergeMethods = cloneRulesetMergeMethodsNormalization(records.MergeMethods)
	records.TeamApprovals = slices.Clone(records.TeamApprovals)
	records.BypassModes = cloneRulesetBypassModeNormalizations(records.BypassModes)
	return records
}

func validateRulesetNormalizationChange(operation string, id int64, before, after *RulesetDeclaration, records RulesetNormalizations) error {
	if operation != "update" || id <= 0 || before == nil || after == nil || before.ID == nil || after.ID == nil || *before.ID != id || *after.ID != id || records.IsZero() {
		return fmt.Errorf("normalized Ruleset change requires one selected update ID")
	}
	state := RulesetState{ID: id, Policy: *before, EnforcementNormalization: records.Enforcement, QualitySeverityNormalization: records.QualitySeverity, RequiredApprovalsNormalization: records.RequiredApprovals, MergeMethodsNormalization: records.MergeMethods, TeamApprovalsNormalizations: records.TeamApprovals, BypassModeNormalizations: records.BypassModes}
	if err := state.validateNormalizationEvidence(); err != nil {
		return err
	}
	if err := ValidateRuleset(*after, true); err != nil {
		return err
	}
	if records.Enforcement != (ObservationNormalization{}) {
		if _, supported := RulesetEnforcementChoiceForValue(*after.Enforcement); !supported {
			return fmt.Errorf("normalized Ruleset enforcement requires a supported desired UI choice")
		}
	}
	if records.RequiredApprovals != (RulesetRequiredApprovalsNormalization{}) {
		if count := rulesetRequiredApprovals(*after); count != nil && !RulesetRequiredApprovalsSupported(*count) {
			return fmt.Errorf("normalized Ruleset required approvals requires a supported desired UI count")
		}
	}
	// Complete validation requires canonical severity for a present quality rule.
	// An absent desired rule is removal, not a synthesized severity choice.
	return nil
}

func NewRulesetChangeWithNormalizations(operation string, id int64, before, after *RulesetDeclaration, records RulesetNormalizations) (RulesetChange, error) {
	if records.IsZero() {
		return RulesetChange{}, fmt.Errorf("normalized Ruleset change requires provenance")
	}
	return newRulesetChange(operation, id, before, after, records)
}

func NewRulesetDisplayChangeWithNormalizations(operation string, before, after *RulesetDeclaration, records RulesetNormalizations) (RulesetDisplayChange, error) {
	if records.IsZero() {
		return RulesetDisplayChange{}, fmt.Errorf("normalized Ruleset display requires provenance")
	}
	return newRulesetDisplayChange(operation, before, after, records)
}
