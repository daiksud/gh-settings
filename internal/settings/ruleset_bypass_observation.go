package settings

import (
	"fmt"
	"sort"
)

// RulesetBypassModeNormalization binds one acquired token to its exact actor.
// It is observation evidence, never a desired actor or writable policy value.
type RulesetBypassModeNormalization struct {
	ActorType     string
	ActorID       *int64
	Normalization ObservationNormalization
}

type rulesetBypassIdentity struct {
	actorType string
	actorID   int64
	hasID     bool
}

func bypassIdentity(actorType string, actorID *int64) rulesetBypassIdentity {
	identity := rulesetBypassIdentity{actorType: actorType}
	if actorID != nil {
		identity.actorID, identity.hasID = *actorID, true
	}
	return identity
}

// RawBypassActors returns owned acquired actors after validating all evidence.
// A complete empty list is distinct from missing acquisition or invalid evidence.
func (state RulesetState) RawBypassActors() ([]BypassActor, error) {
	if err := state.validateNormalizationEvidence(); err != nil {
		return nil, err
	}
	actors := *cloneRuleset(state.Policy).BypassActors
	for _, record := range state.BypassModeNormalizations {
		key := bypassIdentity(record.ActorType, record.ActorID)
		for i := range actors {
			if bypassIdentity(actors[i].ActorType, actors[i].ActorID) == key {
				value := record.Normalization.RawValue
				actors[i].BypassMode = &value
			}
		}
	}
	return actors, nil
}

func (state RulesetState) validateBypassModeNormalizations() error {
	actors := make(map[rulesetBypassIdentity]BypassActor, len(*state.Policy.BypassActors))
	for _, actor := range *state.Policy.BypassActors {
		actors[bypassIdentity(actor.ActorType, actor.ActorID)] = actor
	}
	seen := make(map[rulesetBypassIdentity]bool, len(state.BypassModeNormalizations))
	fallback, _ := RulesetBypassModeValueForChoice(DefaultRulesetBypassModeChoice)
	for _, record := range state.BypassModeNormalizations {
		key := bypassIdentity(record.ActorType, record.ActorID)
		actor, found := actors[key]
		normalization := record.Normalization
		_, supported := RulesetBypassModeChoiceForValue(normalization.RawValue)
		if !found || seen[key] || *actor.BypassMode != fallback || normalization.Path != RulesetBypassModePath || normalization.Fallback != string(DefaultRulesetBypassModeChoice) || supported {
			return fmt.Errorf("invalid Ruleset bypass mode observation provenance")
		}
		seen[key] = true
	}
	return nil
}

func cloneRulesetBypassModeNormalizations(records []RulesetBypassModeNormalization) []RulesetBypassModeNormalization {
	if records == nil {
		return nil
	}
	result := append([]RulesetBypassModeNormalization{}, records...)
	for i := range result {
		if result[i].ActorID != nil {
			value := *result[i].ActorID
			result[i].ActorID = &value
		}
	}
	return result
}

// OrderedRulesetBypassModeNormalizations returns owned evidence in actor order.
// It preserves nil/empty collections and does not validate or normalize identity.
func OrderedRulesetBypassModeNormalizations(records []RulesetBypassModeNormalization) []RulesetBypassModeNormalization {
	result := cloneRulesetBypassModeNormalizations(records)
	sort.Slice(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.ActorType != b.ActorType {
			return a.ActorType < b.ActorType
		}
		if a.ActorID == nil || b.ActorID == nil {
			return a.ActorID == nil && b.ActorID != nil
		}
		return *a.ActorID < *b.ActorID
	})
	return result
}

func cloneRulesetState(state RulesetState) RulesetState {
	state.Policy = cloneRuleset(state.Policy)
	state.BypassModeNormalizations = cloneRulesetBypassModeNormalizations(state.BypassModeNormalizations)
	return state
}
