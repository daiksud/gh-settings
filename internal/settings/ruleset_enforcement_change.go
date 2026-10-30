package settings

import "fmt"

func (c RulesetChange) EnforcementNormalization() ObservationNormalization {
	return c.enforcementNormalization
}
func (c RulesetDisplayChange) EnforcementNormalization() ObservationNormalization {
	return c.enforcementNormalization
}

func validateRulesetEnforcementChange(operation string, id int64, before, after *RulesetDeclaration, record ObservationNormalization) error {
	if operation != "update" || id <= 0 || before == nil || after == nil || before.ID == nil || after.ID == nil || *before.ID != id || *after.ID != id || record == (ObservationNormalization{}) {
		return fmt.Errorf("normalized Ruleset enforcement requires one selected update ID")
	}
	state := RulesetState{ID: id, Policy: *before, EnforcementNormalization: record}
	if _, err := state.RawEnforcement(); err != nil {
		return err
	}
	if err := ValidateRuleset(*after, true); err != nil {
		return err
	}
	if _, supported := RulesetEnforcementChoiceForValue(*after.Enforcement); !supported {
		return fmt.Errorf("normalized Ruleset enforcement requires a supported desired UI choice")
	}
	return nil
}

func NewRulesetChangeWithEnforcementNormalization(operation string, id int64, before, after *RulesetDeclaration, record ObservationNormalization) (RulesetChange, error) {
	if record == (ObservationNormalization{}) {
		return RulesetChange{}, fmt.Errorf("normalized Ruleset enforcement requires provenance")
	}
	return newRulesetChange(operation, id, before, after, record)
}

func NewRulesetDisplayChangeWithEnforcementNormalization(operation string, before, after *RulesetDeclaration, record ObservationNormalization) (RulesetDisplayChange, error) {
	if record == (ObservationNormalization{}) {
		return RulesetDisplayChange{}, fmt.Errorf("normalized Ruleset enforcement requires provenance")
	}
	return newRulesetDisplayChange(operation, before, after, record)
}
