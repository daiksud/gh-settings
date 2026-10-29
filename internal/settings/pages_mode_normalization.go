package settings

import "fmt"

func NewPagesObservationWithModeNormalization(state PagesState, modeRecord, folderRecord ObservationNormalization) (PagesObservation, error) {
	if state.BuildType != "legacy" || state.Source == nil {
		return PagesObservation{}, fmt.Errorf("normalized Pages mode requires a complete legacy source")
	}
	if err := validatePagesModeNormalization(state.BuildType, modeRecord); err != nil {
		return PagesObservation{}, err
	}
	var observed PagesObservation
	var err error
	if folderRecord != (ObservationNormalization{}) {
		observed, err = NewPagesObservationWithFolderNormalization(state, folderRecord)
	} else {
		observed, err = NewPagesObservation(state)
	}
	if err != nil {
		return PagesObservation{}, err
	}
	observed.modeNormalization = modeRecord
	return observed, nil
}

func (o PagesObservation) ModeNormalization() ObservationNormalization { return o.modeNormalization }

func (o PagesObservation) RawBuildType() (string, error) {
	state, err := o.Values()
	if err != nil {
		return "", err
	}
	if o.modeNormalization != (ObservationNormalization{}) {
		return o.modeNormalization.RawValue, nil
	}
	return state.BuildType, nil
}

func validatePagesModeNormalization(effective string, record ObservationNormalization) error {
	fallback, _ := PagesModeValueForChoice(DefaultPagesModeChoice)
	_, supported := PagesModeChoiceForValue(record.RawValue)
	if effective != fallback || record.Path != PagesModePath || record.Fallback != string(DefaultPagesModeChoice) || supported {
		return fmt.Errorf("invalid Pages mode normalization provenance")
	}
	return nil
}
