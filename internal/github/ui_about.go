package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// AboutDeclaration translates UI intent to the existing adapter inputs.
// It does not apply UI observation defaults to omitted desired values.
func AboutDeclaration(about uimodel.About) (settings.Declaration, error) {
	result := settings.Declaration{Repository: settings.RepositoryDeclaration{
		Description: copyUIValue(about.Description), Homepage: copyUIValue(about.Website),
	}}
	if about.Topics != nil {
		if err := settings.ValidateTopics(*about.Topics); err != nil {
			return settings.Declaration{}, err
		}
		names := settings.SortedTopics(*about.Topics)
		result.Topics.Names = &names
	}
	return result, nil
}

// AboutObservation projects values from the existing complete API readers.
// Call it only after both repository and topics acquisition succeeded.
func AboutObservation(repository settings.RepositoryState, topics []string) uimodel.About {
	names := settings.SortedTopics(topics)
	return uimodel.About{Description: copyUIValue(&repository.Description), Website: copyUIValue(&repository.Homepage), Topics: &names}
}

func copyUIValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
