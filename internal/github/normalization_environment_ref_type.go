package github

import (
	"encoding/json"

	"github.com/daiksud/gh-settings/internal/settings"
)

// decodeObservedEnvironmentPattern projects only an acquired string type.
// Mutation acknowledgments continue using the strict native decoder.
func decodeObservedEnvironmentPattern(raw map[string]json.RawMessage) (settings.EnvironmentPatternState, error) {
	var kind string
	if err := requiredEnvironmentJSON(raw, "type", &kind); err != nil {
		return settings.EnvironmentPatternState{}, err
	}
	record := settings.ObservationNormalization{}
	projected := raw
	if _, supported := settings.EnvironmentRefTypeChoiceForValue(kind); !supported {
		projected = make(map[string]json.RawMessage, len(raw))
		for key, value := range raw {
			projected[key] = value
		}
		fallback := string(settings.DefaultEnvironmentRefType)
		projected["type"], _ = json.Marshal(fallback)
		record = settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: kind, Fallback: fallback}
	}
	state, err := decodeEnvironmentPattern(projected)
	if err != nil {
		return settings.EnvironmentPatternState{}, err
	}
	state.RefTypeNormalization = record
	if _, err := state.RawPattern(); err != nil {
		return settings.EnvironmentPatternState{}, err
	}
	return state, nil
}
