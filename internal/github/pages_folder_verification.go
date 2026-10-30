package github

import (
	"reflect"

	"github.com/daiksud/gh-settings/internal/settings"
)

func preparePagesVerificationState(o settings.PagesObservation, d settings.PagesDeclaration) (settings.PagesState, error) {
	expected, err := settings.PreparePages(o, d)
	if err != nil {
		return settings.PagesState{}, err
	}
	if d.Source == nil && expected.BuildType == "legacy" {
		expected.Source, err = o.RawSource()
		if err != nil {
			return settings.PagesState{}, err
		}
	}
	if d.BuildType == nil {
		expected.BuildType, err = o.RawBuildType()
		if err != nil {
			return settings.PagesState{}, err
		}
	}
	return expected, nil
}

func samePagesVerificationState(expected settings.PagesState, actual settings.PagesObservation) (bool, error) {
	state, err := actual.Values()
	if err != nil {
		return false, err
	}
	state.Source, err = actual.RawSource()
	if err != nil {
		return false, err
	}
	state.BuildType, err = actual.RawBuildType()
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(state, expected), nil
}
