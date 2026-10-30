package settings

import "fmt"

// RulesetNormalizations owns the two scalar observation records independently.
// It is validation/presentation evidence and never a writable policy payload.
type RulesetNormalizations struct {
	Enforcement     ObservationNormalization
	QualitySeverity ObservationNormalization
}

func (c RulesetChange) Normalizations() RulesetNormalizations        { return c.normalizations }
func (c RulesetDisplayChange) Normalizations() RulesetNormalizations { return c.normalizations }

func validateRulesetNormalizationChange(operation string, id int64, before, after *RulesetDeclaration, records RulesetNormalizations) error {
	if operation != "update" || id <= 0 || before == nil || after == nil || before.ID == nil || after.ID == nil || *before.ID != id || *after.ID != id || records == (RulesetNormalizations{}) {
		return fmt.Errorf("normalized Ruleset change requires one selected update ID")
	}
	state := RulesetState{ID: id, Policy: *before, EnforcementNormalization: records.Enforcement, QualitySeverityNormalization: records.QualitySeverity}
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
	// Complete validation requires canonical severity for a present quality rule.
	// An absent desired rule is removal, not a synthesized severity choice.
	return nil
}

func NewRulesetChangeWithNormalizations(operation string, id int64, before, after *RulesetDeclaration, records RulesetNormalizations) (RulesetChange, error) {
	if records == (RulesetNormalizations{}) {
		return RulesetChange{}, fmt.Errorf("normalized Ruleset change requires provenance")
	}
	return newRulesetChange(operation, id, before, after, records)
}

func NewRulesetDisplayChangeWithNormalizations(operation string, before, after *RulesetDeclaration, records RulesetNormalizations) (RulesetDisplayChange, error) {
	if records == (RulesetNormalizations{}) {
		return RulesetDisplayChange{}, fmt.Errorf("normalized Ruleset display requires provenance")
	}
	return newRulesetDisplayChange(operation, before, after, records)
}
