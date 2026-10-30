package settings

// RawQualitySeverity distinguishes an absent rule from acquired empty raw text.
// A present value is owned by the caller and cannot mutate observation evidence.
func (state RulesetState) RawQualitySeverity() (*string, error) {
	if err := state.validateNormalizationEvidence(); err != nil {
		return nil, err
	}
	severity := rulesetQualitySeverity(state.Policy)
	if severity == nil {
		return nil, nil
	}
	value := *severity
	if record := state.QualitySeverityNormalization; record != (ObservationNormalization{}) {
		value = record.RawValue
	}
	return &value, nil
}
