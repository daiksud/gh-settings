package settings

// RawEnforcement returns acquired policy only after validating complete evidence.
func (state RulesetState) RawEnforcement() (string, error) {
	if err := state.validateNormalizationEvidence(); err != nil {
		return "", err
	}
	record := state.EnforcementNormalization
	if record == (ObservationNormalization{}) {
		return *state.Policy.Enforcement, nil
	}
	return record.RawValue, nil
}
