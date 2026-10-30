package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// SecurityStatusNormalization retains an acquired API status separately from
// its effective boolean. It never supplies omitted desired intent.
type SecurityStatusNormalization struct {
	Path     string `json:"path"`
	RawValue string `json:"raw_value"`
	Fallback bool   `json:"fallback"`
}

func (record *SecurityStatusNormalization) UnmarshalJSON(data []byte) error {
	var fields struct {
		Path     *string `json:"path"`
		RawValue *string `json:"raw_value"`
		Fallback *bool   `json:"fallback"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return fmt.Errorf("decode security status normalization: %w", err)
	}
	if fields.Path == nil || fields.RawValue == nil || fields.Fallback == nil {
		return fmt.Errorf("security status normalization requires string path/raw_value and boolean fallback")
	}
	*record = SecurityStatusNormalization{Path: *fields.Path, RawValue: *fields.RawValue, Fallback: *fields.Fallback}
	return nil
}

func NewSecretScanningObservationWithNormalizations(scanning, push bool, records []SecurityStatusNormalization) (SecretScanningObservation, error) {
	observation := NewSecretScanningObservation(scanning, push)
	controls := [2]SecurityStatusControl{SecretProtectionControl, PushProtectionControl}
	for _, record := range records {
		index := -1
		for i, control := range controls {
			if record.Path == control.Path() {
				index = i
				break
			}
		}
		if index < 0 || observation.normalizations[index] != (SecurityStatusNormalization{}) {
			return SecretScanningObservation{}, fmt.Errorf("invalid or duplicate security status normalization")
		}
		control := controls[index]
		if err := validateSecurityStatusNormalization(control, [2]bool{scanning, push}[index], record); err != nil {
			return SecretScanningObservation{}, err
		}
		observation.normalizations[index] = record
	}
	return observation, nil
}

func (observation SecretScanningObservation) Normalizations() []SecurityStatusNormalization {
	var records []SecurityStatusNormalization
	for _, record := range observation.normalizations {
		if record.Path != "" {
			records = append(records, record)
		}
	}
	return records
}

func (observation SecretScanningObservation) RawStatuses() (string, string, error) {
	scanning, push, err := observation.Values()
	if err != nil {
		return "", "", err
	}
	statuses := [2]string{SecurityStatusForValue(scanning), SecurityStatusForValue(push)}
	for i, record := range observation.normalizations {
		if record.Path != "" {
			statuses[i] = record.RawValue
		}
	}
	return statuses[0], statuses[1], nil
}

func NewSecurityChangeWithNormalization(field string, before, after any, record SecurityStatusNormalization) (SecurityChange, error) {
	var control SecurityStatusControl
	for _, candidate := range []SecurityStatusControl{SecretProtectionControl, PushProtectionControl} {
		if candidate.Field() == field {
			control = candidate
			break
		}
	}
	b, bvalid := before.(bool)
	a, avalid := after.(bool)
	if !bvalid || !avalid {
		return SecurityChange{}, fmt.Errorf("invalid security status comparison normalization")
	}
	if err := validateSecurityStatusNormalization(control, b, record); err != nil {
		return SecurityChange{}, err
	}
	return SecurityChange{field: field, before: b, after: a, normalization: record}, nil
}

func validateSecurityStatusNormalization(control SecurityStatusControl, effective bool, record SecurityStatusNormalization) error {
	fallback, known := control.DefaultValue()
	_, supported := SecurityValueForStatus(record.RawValue)
	if !known || effective != fallback || record.Path != control.Path() || record.Fallback != fallback || supported {
		return fmt.Errorf("invalid security status normalization provenance")
	}
	return nil
}
