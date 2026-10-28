package github

import (
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

// LabelsDeclaration retains omission and the existing selected-ID ownership.
func LabelsDeclaration(labels *[]uimodel.Label) (*settings.LabelDeclarations, error) {
	if labels == nil {
		return nil, nil
	}
	entries := make([]settings.LabelDeclaration, len(*labels))
	for i, label := range *labels {
		if label.Color == nil || !strings.HasPrefix(*label.Color, "#") {
			return nil, fmt.Errorf("labels color requires # followed by six hexadecimal characters")
		}
		color := strings.TrimPrefix(*label.Color, "#")
		entries[i] = settings.LabelDeclaration{ID: label.ID, Name: label.Name, Color: &color, Description: label.Description}
	}
	result, err := settings.NewLabelDeclarations(entries)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// LabelsObservation requires a completely acquired, validated collection.
func LabelsObservation(observed settings.LabelObservation) ([]uimodel.Label, error) {
	entries, err := observed.Values()
	if err != nil {
		return nil, err
	}
	result := make([]uimodel.Label, len(entries))
	for i, label := range entries {
		color := "#" + label.Color
		result[i] = uimodel.Label{ID: &label.ID, Name: label.Name, Color: &color, Description: &label.Description}
	}
	return result, nil
}
