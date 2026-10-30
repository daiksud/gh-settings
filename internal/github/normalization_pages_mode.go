package github

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
)

type preparedPagesMode struct {
	BuildType     string
	Normalization settings.ObservationNormalization
}

func preparePagesModeValue(raw *string) (preparedPagesMode, error) {
	if raw == nil {
		return preparedPagesMode{}, fmt.Errorf("Pages mode requires an acquired string")
	}
	if _, valid := settings.PagesModeChoiceForValue(*raw); valid {
		return preparedPagesMode{BuildType: *raw}, nil
	}
	fallback, _ := settings.PagesModeValueForChoice(settings.DefaultPagesModeChoice)
	return preparedPagesMode{BuildType: fallback, Normalization: settings.ObservationNormalization{
		Path: settings.PagesModePath, RawValue: *raw, Fallback: string(settings.DefaultPagesModeChoice),
	}}, nil
}
