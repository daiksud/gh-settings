package actions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

type EnvironmentPlanChange struct {
	Operation                   string                                       `json:"operation"`
	Before                      *settings.EnvironmentDeclaration             `json:"before,omitempty"`
	After                       *settings.EnvironmentDeclaration             `json:"after"`
	DeploymentModeNormalization *settings.EnvironmentDeploymentNormalization `json:"deployment_mode_normalization,omitempty"`
	PatternObservations         *[]EnvironmentPatternObservation             `json:"pattern_observations,omitempty"`
}

// EnvironmentPatternObservation is private rendering evidence, never apply
// input or independent proof of complete API acquisition.
type EnvironmentPatternObservation struct {
	ID                   int64                              `json:"id"`
	Pattern              settings.EnvironmentPattern        `json:"pattern"`
	RefTypeNormalization *settings.ObservationNormalization `json:"ref_type_normalization,omitempty"`
}

func (c *EnvironmentPlanChange) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	var invalid func(any, string) bool
	invalid = func(value any, key string) bool {
		if value == nil {
			return key != "deployment_branch_policy"
		}
		switch value := value.(type) {
		case map[string]any:
			for key, value := range value {
				if key != strings.ToLower(key) || invalid(value, key) {
					return true
				}
			}
		case []any:
			for _, value := range value {
				if invalid(value, "") {
					return true
				}
			}
		}
		return false
	}
	if invalid(raw, "") {
		return fmt.Errorf("null or noncanonical environment report value")
	}
	type plain EnvironmentPlanChange
	var out plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return err
	}
	// Branch-policy null must be explicit, unlike defaulted YAML omission.
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		return err
	}
	for _, side := range []string{"before", "after"} {
		if value, ok := top[side]; ok {
			var policy map[string]json.RawMessage
			if err := json.Unmarshal(value, &policy); err != nil {
				return err
			}
			if _, ok := policy["deployment_branch_policy"]; !ok && side == "before" {
				return fmt.Errorf("environment report missing deployment_branch_policy")
			}
			if value := policy["deployment_branch_policy"]; value != nil && !bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				var branch map[string]json.RawMessage
				if err := json.Unmarshal(value, &branch); err != nil {
					return err
				}
				if branch["protected_branches"] == nil || branch["custom_branch_policies"] == nil {
					return fmt.Errorf("environment report missing branch-policy boolean")
				}
			}
		}
	}
	*c = EnvironmentPlanChange(out)
	return nil
}

func validateEnvironmentReport(entries []EnvironmentPlanChange) ([]settings.EnvironmentChange, error) {
	changes := []settings.EnvironmentChange{}
	last := ""
	for _, entry := range entries {
		var change settings.EnvironmentChange
		var err error
		if entry.PatternObservations != nil && entry.DeploymentModeNormalization != nil {
			return nil, fmt.Errorf("environment mode and pattern normalization metadata cannot be combined")
		}
		if entry.PatternObservations != nil {
			patterns := make([]settings.EnvironmentPatternState, 0, len(*entry.PatternObservations))
			for _, value := range *entry.PatternObservations {
				state := settings.EnvironmentPatternState{ID: value.ID, Pattern: value.Pattern}
				if value.RefTypeNormalization != nil {
					if *value.RefTypeNormalization == (settings.ObservationNormalization{}) {
						return nil, fmt.Errorf("present environment ref type normalization cannot be empty")
					}
					state.RefTypeNormalization = *value.RefTypeNormalization
				}
				patterns = append(patterns, state)
			}
			change, err = settings.NewEnvironmentChangeWithPatternNormalizations(entry.Operation, entry.Before, entry.After, patterns)
		} else if record := entry.DeploymentModeNormalization; record != nil {
			if *record == (settings.EnvironmentDeploymentNormalization{}) {
				return nil, fmt.Errorf("present environment deployment normalization cannot be empty")
			}
			change, err = settings.NewEnvironmentChangeWithDeploymentNormalization(entry.Operation, entry.Before, entry.After, *record)
		} else {
			change, err = settings.NewEnvironmentChange(entry.Operation, entry.Before, entry.After)
		}
		if err != nil {
			return nil, err
		}
		key := settings.EnvironmentIdentity(change.Name())
		if last != "" && key <= last {
			return nil, fmt.Errorf("duplicate or unordered environment changes")
		}
		last = key
		changes = append(changes, change)
	}
	return changes, nil
}
