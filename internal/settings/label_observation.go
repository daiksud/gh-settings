package settings

import (
	"fmt"
	"slices"
	"strings"
)

// LabelObservation owns the complete writable values of a finished labels read.
// The zero value is missing; construction distinguishes a completed empty read.
type LabelObservation struct{ entries []LabelState }

func NewLabelObservation(labels []LabelState) (LabelObservation, error) {
	owned := make([]LabelState, len(labels))
	seen := make(map[string]bool, len(labels))
	ids := make(map[int64]bool, len(labels))
	for i, label := range labels {
		if label.ID <= 0 || ids[label.ID] {
			return LabelObservation{}, fmt.Errorf("invalid or duplicate label id")
		}
		ids[label.ID] = true
		if err := ValidateLabelName(label.Name); err != nil {
			return LabelObservation{}, err
		}
		identity := LabelIdentity(label.Name)
		if seen[identity] {
			return LabelObservation{}, fmt.Errorf("ambiguous duplicate label name %q", label.Name)
		}
		seen[identity] = true
		if err := ValidateLabelColor(label.Color); err != nil {
			return LabelObservation{}, err
		}
		if err := ValidateLabelDescription(label.Description); err != nil {
			return LabelObservation{}, err
		}
		label.Color = strings.ToLower(label.Color)
		owned[i] = label
	}
	return LabelObservation{entries: owned}, nil
}

// Values returns independent values, or an error when no complete read exists.
func (o LabelObservation) Values() ([]LabelState, error) {
	if o.entries == nil {
		return nil, fmt.Errorf("missing labels observation")
	}
	return slices.Clone(o.entries), nil
}
