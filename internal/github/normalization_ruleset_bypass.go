package github

import (
	"encoding/json"
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

// prepareRulesetBypassActors projects acquired modes, not a complete policy.
// The caller must still use strict policy validation before publishing evidence.
func prepareRulesetBypassActors(raw json.RawMessage) (json.RawMessage, []settings.RulesetBypassModeNormalization, error) {
	var actors []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &actors); err != nil {
		return nil, nil, err
	}
	if actors == nil {
		return nil, nil, fmt.Errorf("Rulesets response requires a non-null bypass_actors array")
	}
	type identity struct {
		kind  string
		id    int64
		hasID bool
	}
	seen := map[identity]bool{}
	var records []settings.RulesetBypassModeNormalization
	for _, actor := range actors {
		var kind, mode string
		if err := requiredRuleJSON(actor, "actor_type", &kind); err != nil {
			return nil, nil, err
		}
		if err := requiredRuleJSON(actor, "bypass_mode", &mode); err != nil {
			return nil, nil, err
		}
		var id *int64
		if rawID, present := actor["actor_id"]; present {
			if err := json.Unmarshal(rawID, &id); err != nil {
				return nil, nil, err
			}
		}
		switch kind {
		case "Integration", "RepositoryRole", "Team", "User":
			if id == nil || *id <= 0 {
				return nil, nil, fmt.Errorf("bypass actor %q requires a positive actor_id", kind)
			}
		case "OrganizationAdmin":
			// GitHub's typed numeric value is ignored for this actor identity.
			id = nil
		case "DeployKey":
			if id != nil {
				return nil, nil, fmt.Errorf("DeployKey actor_id must be null")
			}
		default:
			return nil, nil, fmt.Errorf("unsupported bypass actor %q", kind)
		}
		key := identity{kind: kind, hasID: id != nil}
		if id != nil {
			key.id = *id
		}
		if seen[key] {
			return nil, nil, fmt.Errorf("duplicate bypass actor identity")
		}
		seen[key] = true
		if _, supported := settings.RulesetBypassModeChoiceForValue(mode); supported {
			continue
		}
		fallback, _ := settings.RulesetBypassModeValueForChoice(settings.DefaultRulesetBypassModeChoice)
		encoded, err := json.Marshal(fallback)
		if err != nil {
			return nil, nil, err
		}
		actor["bypass_mode"] = encoded
		records = append(records, settings.RulesetBypassModeNormalization{
			ActorType: kind, ActorID: id,
			Normalization: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: mode, Fallback: string(settings.DefaultRulesetBypassModeChoice)},
		})
	}
	encoded, err := json.Marshal(actors)
	if err != nil {
		return nil, nil, err
	}
	return encoded, records, nil
}
