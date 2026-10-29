package settings

import "fmt"

// RawPattern returns the acquired identity, never the UI fallback identity.
// The enclosing state binds the evidence to its positive ID and exact name.
func (state EnvironmentPatternState) RawPattern() (EnvironmentPattern, error) {
	if state.ID <= 0 {
		return EnvironmentPattern{}, fmt.Errorf("environment deployment pattern requires a positive ID")
	}
	if _, err := normalizeEnvironmentPatterns([]EnvironmentPattern{state.Pattern}); err != nil {
		return EnvironmentPattern{}, err
	}
	raw := state.Pattern
	if record := state.RefTypeNormalization; record != (ObservationNormalization{}) {
		_, supported := EnvironmentRefTypeChoiceForValue(record.RawValue)
		if record.Path != EnvironmentRefTypePath || record.Fallback != string(DefaultEnvironmentRefType) || state.Pattern.Type != record.Fallback || supported {
			return EnvironmentPattern{}, fmt.Errorf("invalid environment deployment ref type observation provenance")
		}
		raw.Type = record.RawValue
	}
	return raw, nil
}
