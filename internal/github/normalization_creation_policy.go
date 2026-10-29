package github

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

type preparedCreationPolicy struct {
	Value         *string
	Normalization settings.ObservationNormalization
}

func prepareCreationPolicyObservation(value *string, control settings.CreationPolicyControl) (preparedCreationPolicy, error) {
	fallback, valid := control.DefaultChoice()
	if !valid {
		return preparedCreationPolicy{}, fmt.Errorf("unsupported creation policy control %q", control)
	}
	if value == nil {
		return preparedCreationPolicy{}, nil
	}
	if _, valid := settings.CreationPolicyChoiceForValue(*value); valid {
		acquired := *value
		return preparedCreationPolicy{Value: &acquired}, nil
	}
	effective, valid := settings.CreationPolicyValueForChoice(fallback)
	if !valid {
		return preparedCreationPolicy{}, fmt.Errorf("missing adopted %s creation policy fallback", control)
	}
	return preparedCreationPolicy{Value: &effective, Normalization: settings.ObservationNormalization{Path: control.Path(), RawValue: *value, Fallback: string(fallback)}}, nil
}
