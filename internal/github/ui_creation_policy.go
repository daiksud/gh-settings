package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareCreationPolicy(choice *uimodel.CreationAllowedBy, control settings.CreationPolicyControl) (*string, error) {
	if choice == nil {
		return nil, nil
	}
	if value, ok := settings.CreationPolicyValueForChoice(*choice); ok {
		return &value, nil
	}
	return nil, fmt.Errorf("unsupported general.features.%s.creation_allowed_by %q", control, *choice)
}

func observeCreationPolicy(value *string, control settings.CreationPolicyControl) (*uimodel.CreationAllowedBy, error) {
	if value == nil {
		return nil, nil
	}
	if choice, ok := settings.CreationPolicyChoiceForValue(*value); ok {
		return &choice, nil
	}
	return nil, fmt.Errorf("general.features.%s.creation_allowed_by is outside UI choices", control)
}
