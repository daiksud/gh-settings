package settings

import (
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// LabelDeclaration selects an existing ID or a create-only name and owns the
// complete supported object; omitted description has the fixed empty default.
type LabelDeclaration struct {
	ID          *int64
	Name        string
	Color       *string
	Description *string
}

// LabelDeclarations retains validated, unique declarations and owns their values.
// Its zero value is an explicitly empty collection; omission uses a nil pointer.
type LabelDeclarations struct{ entries []LabelDeclaration }

func NewLabelDeclarations(labels []LabelDeclaration) (LabelDeclarations, error) {
	if err := ValidateLabels(labels); err != nil {
		return LabelDeclarations{}, err
	}
	owned := copyLabelDeclarations(labels)
	for _, label := range owned {
		if label.Color != nil {
			*label.Color = strings.ToLower(*label.Color)
		}
	}
	return LabelDeclarations{entries: owned}, nil
}

func (d LabelDeclarations) Len() int { return len(d.entries) }

// Values returns an independent slice, including independent attribute pointers.
func (d LabelDeclarations) Values() []LabelDeclaration { return copyLabelDeclarations(d.entries) }

func copyLabelDeclarations(labels []LabelDeclaration) []LabelDeclaration {
	result := make([]LabelDeclaration, len(labels))
	for i, label := range labels {
		result[i] = copyLabelDeclaration(label)
	}
	return result
}

func copyLabelDeclaration(label LabelDeclaration) LabelDeclaration {
	if label.ID != nil {
		id := *label.ID
		label.ID = &id
	}
	if label.Color != nil {
		color := *label.Color
		label.Color = &color
	}
	if label.Description != nil {
		description := *label.Description
		label.Description = &description
	}
	return label
}

// LabelState is the complete writable subset observed from GitHub.
type LabelState struct {
	Name, Color, Description string
	ID                       int64
}

// LabelIdentity uses Unicode simple case folding without trimming or renaming.
func LabelIdentity(name string) string {
	return strings.Map(func(r rune) rune {
		smallest := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < smallest {
				smallest = next
			}
		}
		return smallest
	}, name)
}

func ValidateLabelName(name string) error {
	if strings.TrimSpace(name) == "" || !utf8.ValidString(name) {
		return fmt.Errorf("label name must be a nonempty string")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("label name must not contain control characters")
		}
	}
	return nil
}

func ValidateLabelColor(color string) error {
	if len(color) != 6 {
		return fmt.Errorf("label color must be six hexadecimal characters without #")
	}
	if _, err := hex.DecodeString(color); err != nil {
		return fmt.Errorf("label color must be six hexadecimal characters without #")
	}
	return nil
}

func ValidateLabelDescription(description string) error {
	if !utf8.ValidString(description) || utf8.RuneCountInString(description) > 100 {
		return fmt.Errorf("label description must contain at most 100 characters")
	}
	return nil
}

func ValidateLabels(labels []LabelDeclaration) error {
	seen := make(map[string]bool, len(labels))
	ids := make(map[int64]bool, len(labels))
	for _, label := range labels {
		if label.ID != nil {
			if *label.ID <= 0 || ids[*label.ID] {
				return fmt.Errorf("invalid or duplicate label id")
			}
			ids[*label.ID] = true
		}
		if err := ValidateLabelName(label.Name); err != nil {
			return err
		}
		identity := LabelIdentity(label.Name)
		if seen[identity] {
			return fmt.Errorf("ambiguous duplicate label name %q", label.Name)
		}
		seen[identity] = true
		if label.Color == nil {
			return fmt.Errorf("label %q requires color", label.Name)
		}
		if label.Color != nil {
			if err := ValidateLabelColor(*label.Color); err != nil {
				return err
			}
		}
		if label.Description != nil {
			if err := ValidateLabelDescription(*label.Description); err != nil {
				return err
			}
		}
	}
	return nil
}

func SortedLabels(labels []LabelDeclaration) []LabelDeclaration {
	result := append([]LabelDeclaration{}, labels...)
	slices.SortFunc(result, func(a, b LabelDeclaration) int { return strings.Compare(LabelIdentity(a.Name), LabelIdentity(b.Name)) })
	return result
}

// CompareLabels selects only listed identities. Missing IDs are create-only;
// supplied IDs must resolve in the selected repository and never fall back.
func CompareLabels(current LabelObservation, desired *LabelDeclarations) (Comparison, error) {
	if desired == nil || desired.Len() == 0 {
		return Comparison{}, nil
	}
	labels, err := current.Values()
	if err != nil {
		return Comparison{}, err
	}
	observed := make(map[string]LabelState, len(labels))
	byID := make(map[int64]LabelState, len(labels))
	for _, label := range labels {
		observed[LabelIdentity(label.Name)] = label
		byID[label.ID] = label
	}
	result := Comparison{Managed: 1}
	var changes []LabelChange
	for _, declared := range SortedLabels(desired.Values()) {
		key := LabelIdentity(declared.Name)
		after := copyLabelDeclaration(declared)
		if after.Description == nil {
			empty := ""
			after.Description = &empty
		}
		if declared.ID == nil {
			if existing, present := observed[key]; present {
				return Comparison{}, fmt.Errorf("label %q already exists (id %d); specify id: %d to update it", declared.Name, existing.ID, existing.ID)
			}
			changes = append(changes, LabelChange{operation: labelCreate{after: after}})
			continue
		}
		before, present := byID[*declared.ID]
		if !present {
			return Comparison{}, fmt.Errorf("label id %d does not exist in this repository; refusing create fallback", *declared.ID)
		}
		if existing, present := observed[key]; present && existing.ID != before.ID {
			return Comparison{}, fmt.Errorf("label name %q belongs to existing id %d; refusing rename collision", declared.Name, existing.ID)
		}
		if LabelIdentity(before.Name) == key {
			after.Name = before.Name
		}
		if after.Name != before.Name || *after.Color != before.Color || *after.Description != before.Description {
			changes = append(changes, LabelChange{operation: labelUpdate{before: before, after: after}})
		}
	}
	result.Labels = LabelChanges{entries: changes}
	return result, nil
}
