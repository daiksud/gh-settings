package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareCreationPolicy(choice *uimodel.CreationAllowedBy, path string) (*string, error) {
	if choice == nil {
		return nil, nil
	}
	switch *choice {
	case uimodel.AllUsers:
		v := "ALL"
		return &v, nil
	case uimodel.CollaboratorsOnly:
		v := "COLLABORATORS_ONLY"
		return &v, nil
	default:
		return nil, fmt.Errorf("unsupported general.features.%s.creation_allowed_by %q", path, *choice)
	}
}

func observeCreationPolicy(value *string, path string) (*uimodel.CreationAllowedBy, error) {
	if value == nil {
		return nil, nil
	}
	var choice uimodel.CreationAllowedBy
	switch *value {
	case "ALL":
		choice = uimodel.AllUsers
	case "COLLABORATORS_ONLY":
		choice = uimodel.CollaboratorsOnly
	default:
		return nil, fmt.Errorf("general.features.%s.creation_allowed_by is outside UI choices", path)
	}
	return &choice, nil
}
