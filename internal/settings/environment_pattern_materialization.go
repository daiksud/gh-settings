package settings

import "fmt"

// ValidateEnvironmentPatternObservation checks complete raw identities without
// assuming that their UI projections remain distinct.
func ValidateEnvironmentPatternObservation(states []EnvironmentPatternState) error {
	if states == nil {
		return fmt.Errorf("environment deployment patterns were not acquired")
	}
	ids := map[int64]bool{}
	identities := map[EnvironmentPattern]bool{}
	for _, state := range states {
		raw, err := state.RawPattern()
		if err != nil {
			return err
		}
		if ids[state.ID] || identities[raw] {
			return fmt.Errorf("ambiguous environment deployment pattern observation")
		}
		ids[state.ID], identities[raw] = true, true
	}
	return nil
}

// EffectiveEnvironmentPatterns materializes a uniquely representable UI list.
// A projection collision is an error; acquired raw entries are never discarded.
func EffectiveEnvironmentPatterns(states []EnvironmentPatternState) ([]EnvironmentPattern, error) {
	if err := ValidateEnvironmentPatternObservation(states); err != nil {
		return nil, err
	}
	patterns := make([]EnvironmentPattern, 0, len(states))
	for _, state := range states {
		patterns = append(patterns, state.Pattern)
	}
	return normalizeEnvironmentPatterns(patterns)
}
