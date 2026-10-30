package settings

import "fmt"

func NewPagesObservationWithFolderNormalization(state PagesState, record ObservationNormalization) (PagesObservation, error) {
	if state.BuildType != "legacy" || state.Source == nil {
		return PagesObservation{}, fmt.Errorf("invalid Pages folder normalization provenance")
	}
	if err := validatePagesFolderNormalization(state.Source.Path, record); err != nil {
		return PagesObservation{}, err
	}
	observation, err := NewPagesObservation(state)
	if err != nil {
		return PagesObservation{}, err
	}
	observation.folderNormalization = record
	return observation, nil
}

func (observation PagesObservation) FolderNormalization() ObservationNormalization {
	return observation.folderNormalization
}

// RawSource restores only the acquired folder token. Other configured-state
// semantics, including domain canonicalization, remain owned by Values.
func (observation PagesObservation) RawSource() (*PagesSource, error) {
	state, err := observation.Values()
	if err != nil {
		return nil, err
	}
	if observation.folderNormalization != (ObservationNormalization{}) {
		if state.Source == nil {
			return nil, fmt.Errorf("normalized Pages folder requires an acquired source")
		}
		state.Source.Path = observation.folderNormalization.RawValue
	}
	return state.Source, nil
}

func NewPagesChangeWithNormalization(field string, before, after any, inactive bool, record ObservationNormalization) (PagesChange, error) {
	b, bvalid := before.(string)
	a, avalid := after.(string)
	_, supportedDesired := PagesFolderChoiceForValue(a)
	if field != "pages.source.path" || !bvalid || !avalid || inactive || !supportedDesired {
		return PagesChange{}, fmt.Errorf("invalid Pages folder comparison normalization")
	}
	if err := validatePagesFolderNormalization(b, record); err != nil {
		return PagesChange{}, err
	}
	return PagesChange{field: field, before: b, after: a, folderNormalization: record}, nil
}

func validatePagesFolderNormalization(effective string, record ObservationNormalization) error {
	fallback, _ := PagesFolderValueForChoice(DefaultPagesFolderChoice)
	_, supported := PagesFolderChoiceForValue(record.RawValue)
	if effective != fallback || record.Path != PagesFolderPath || record.Fallback != string(DefaultPagesFolderChoice) || supported {
		return fmt.Errorf("invalid Pages folder normalization provenance")
	}
	return nil
}
