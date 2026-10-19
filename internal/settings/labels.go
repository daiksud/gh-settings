package settings

import (
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// LabelDeclaration owns only declared attributes; Name identifies the label.
type LabelDeclaration struct {
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
		if label.Color != nil {
			color := *label.Color
			label.Color = &color
		}
		if label.Description != nil {
			description := *label.Description
			label.Description = &description
		}
		result[i] = label
	}
	return result
}

// LabelState is the complete writable subset observed from GitHub.
type LabelState struct{ Name, Color, Description string }

// LabelChange describes a fresh collection operation, never a saved apply input.
type LabelChange struct {
	Operation string
	Name      string
	Before    *LabelState
	After     *LabelDeclaration
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
	for _, label := range labels {
		if err := ValidateLabelName(label.Name); err != nil {
			return err
		}
		identity := LabelIdentity(label.Name)
		if seen[identity] {
			return fmt.Errorf("ambiguous duplicate label name %q", label.Name)
		}
		seen[identity] = true
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

// CompareLabels treats any declared list, including [], as authoritative membership.
func CompareLabels(current LabelObservation, desired *LabelDeclarations) (Comparison, error) {
	if desired == nil {
		return Comparison{}, nil
	}
	labels, err := current.Values()
	if err != nil {
		return Comparison{}, err
	}
	observed := make(map[string]LabelState, len(labels))
	for _, label := range labels {
		observed[LabelIdentity(label.Name)] = label
	}
	result := Comparison{Managed: 1}
	for _, declared := range SortedLabels(desired.Values()) {
		key := LabelIdentity(declared.Name)
		before, present := observed[key]
		if !present {
			after := declared
			result.Labels = append(result.Labels, LabelChange{Operation: "create", Name: declared.Name, After: &after})
			continue
		}
		delete(observed, key)
		after := LabelDeclaration{Name: before.Name}
		if declared.Color != nil && *declared.Color != before.Color {
			after.Color = declared.Color
		}
		if declared.Description != nil && *declared.Description != before.Description {
			after.Description = declared.Description
		}
		if after.Color != nil || after.Description != nil {
			result.Labels = append(result.Labels, LabelChange{Operation: "update", Name: before.Name, Before: &before, After: &after})
		}
	}
	remaining := make([]LabelState, 0, len(observed))
	for _, label := range observed {
		remaining = append(remaining, label)
	}
	slices.SortFunc(remaining, func(a, b LabelState) int { return strings.Compare(LabelIdentity(a.Name), LabelIdentity(b.Name)) })
	for _, label := range remaining {
		before := label
		result.Labels = append(result.Labels, LabelChange{Operation: "delete", Name: label.Name, Before: &before})
	}
	return result, nil
}

// ValidateLabelChanges checks typed operation values at write/report boundaries.
func ValidateLabelChanges(changes []LabelChange) error {
	seen := make(map[string]bool, len(changes))
	for _, change := range changes {
		if err := ValidateLabelName(change.Name); err != nil {
			return err
		}
		key := LabelIdentity(change.Name)
		if seen[key] {
			return fmt.Errorf("duplicate label operation %q", change.Name)
		}
		seen[key] = true
		if change.Before != nil {
			before := change.Before
			if before.Name != change.Name || before.Color != strings.ToLower(before.Color) {
				return fmt.Errorf("invalid label before value")
			}
			if err := ValidateLabels([]LabelDeclaration{{Name: before.Name, Color: &before.Color, Description: &before.Description}}); err != nil {
				return err
			}
		}
		if change.After != nil {
			if change.After.Name != change.Name {
				return fmt.Errorf("label operation cannot rename")
			}
			if err := ValidateLabels([]LabelDeclaration{*change.After}); err != nil {
				return err
			}
			if change.After.Color != nil && *change.After.Color != strings.ToLower(*change.After.Color) {
				return fmt.Errorf("invalid label after color")
			}
		}
		valid := false
		switch change.Operation {
		case "create":
			valid = change.Before == nil && change.After != nil
		case "delete":
			valid = change.Before != nil && change.After == nil
		case "update":
			valid = change.Before != nil && change.After != nil
			if valid {
				a, b := change.After, change.Before
				valid = (a.Color != nil || a.Description != nil) && (a.Color == nil || *a.Color != b.Color) && (a.Description == nil || *a.Description != b.Description)
			}
		}
		if !valid {
			return fmt.Errorf("invalid %q label operation", change.Operation)
		}
	}
	return nil
}
