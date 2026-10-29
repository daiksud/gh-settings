package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
)

type preparedPagesFolder struct {
	Source        settings.PagesSource
	Normalization settings.ObservationNormalization
}

func prepareLegacyPagesFolderObservation(branch, path *string) (preparedPagesFolder, error) {
	if branch == nil || path == nil {
		return preparedPagesFolder{}, fmt.Errorf("legacy Pages source requires complete branch and folder")
	}
	if !settings.ValidDefaultBranch(*branch) {
		return preparedPagesFolder{}, fmt.Errorf("pages.source.branch must be a nonempty literal branch name")
	}
	source := settings.PagesSource{Branch: *branch, Path: *path}
	var record settings.ObservationNormalization
	if _, valid := settings.PagesFolderChoiceForValue(*path); !valid {
		fallback, _ := settings.PagesFolderValueForChoice(settings.DefaultPagesFolderChoice)
		source.Path = fallback
		record = settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: *path, Fallback: string(settings.DefaultPagesFolderChoice)}
	}
	if err := source.Validate(); err != nil {
		return preparedPagesFolder{}, err
	}
	return preparedPagesFolder{Source: source, Normalization: record}, nil
}
