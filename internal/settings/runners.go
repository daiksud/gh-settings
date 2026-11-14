package settings

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"slices"
	"strings"
	"unicode"
)

const RunnersPath = "repository.settings.actions.runners"

type RunnerLabels = uimodel.RunnerLabels

// RunnerLabelIdentity defines client Unicode simple-fold equivalence, without trimming.
func RunnerLabelIdentity(name string) string { return EnvironmentIdentity(name) }
func ValidateRunnerLabelName(name string) error {
	if strings.TrimSpace(name) == "" || strings.ContainsFunc(name, unicode.IsControl) {
		return fmt.Errorf("runner label names must be nonblank and contain no controls")
	}
	return nil
}
func ValidateRunnerLabels(values []RunnerLabels) error {
	ids := map[int64]bool{}
	for _, v := range values {
		if v.ID <= 0 || ids[v.ID] || v.CustomLabels == nil || len(*v.CustomLabels) > 100 {
			return fmt.Errorf("runner labels require unique positive IDs and explicit lists of at most 100 labels")
		}
		ids[v.ID] = true
		names := map[string]bool{}
		for _, name := range *v.CustomLabels {
			if err := ValidateRunnerLabelName(name); err != nil {
				return err
			}
			key := RunnerLabelIdentity(name)
			if names[key] {
				return fmt.Errorf("runner custom labels must be case-insensitively unique")
			}
			names[key] = true
		}
	}
	return nil
}
func CloneRunnerLabels(values []RunnerLabels) []RunnerLabels {
	result := make([]RunnerLabels, 0, len(values))
	for _, v := range values {
		if v.CustomLabels != nil {
			labels := append([]string{}, (*v.CustomLabels)...)
			v.CustomLabels = &labels
		}
		result = append(result, v)
	}
	return result
}
func RunnerLabelsMatch(a, b RunnerLabels) bool {
	if a.ID != b.ID || a.CustomLabels == nil || b.CustomLabels == nil || len(*a.CustomLabels) != len(*b.CustomLabels) {
		return false
	}
	names := map[string]bool{}
	for _, n := range *a.CustomLabels {
		names[RunnerLabelIdentity(n)] = true
	}
	for _, n := range *b.CustomLabels {
		if !names[RunnerLabelIdentity(n)] {
			return false
		}
	}
	return true
}

type RunnerState struct {
	Value          RunnerLabels
	ReadOnlyLabels []string
}
type RunnerObservation struct {
	acquired bool
	entries  []RunnerState
}

func NewRunnerObservation(entries []RunnerState) (RunnerObservation, error) {
	o := RunnerObservation{acquired: true, entries: make([]RunnerState, 0, len(entries))}
	values := []RunnerLabels{}
	for _, v := range entries {
		values = append(values, v.Value)
		seen := map[string]bool{}
		if v.Value.CustomLabels != nil {
			for _, n := range *v.Value.CustomLabels {
				seen[RunnerLabelIdentity(n)] = true
			}
		}
		for _, n := range v.ReadOnlyLabels {
			if ValidateRunnerLabelName(n) != nil || seen[RunnerLabelIdentity(n)] {
				return RunnerObservation{}, fmt.Errorf("runner label classification inconsistent")
			}
			seen[RunnerLabelIdentity(n)] = true
		}
		v.Value = CloneRunnerLabels([]RunnerLabels{v.Value})[0]
		v.ReadOnlyLabels = append([]string{}, v.ReadOnlyLabels...)
		o.entries = append(o.entries, v)
	}
	if err := ValidateRunnerLabels(values); err != nil {
		return RunnerObservation{}, err
	}
	return o, nil
}
func (o RunnerObservation) Values() ([]RunnerLabels, error) {
	if !o.acquired {
		return nil, fmt.Errorf("runner observation unavailable")
	}
	values := []RunnerLabels{}
	for _, v := range o.entries {
		values = append(values, v.Value)
	}
	values = CloneRunnerLabels(values)
	slices.SortFunc(values, func(a, b RunnerLabels) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return values, nil
}

// PrepareRunnerLabels validates every selected runner before the first family write.
func PrepareRunnerLabels(o RunnerObservation, desired []RunnerLabels) error {
	if err := ValidateRunnerLabels(desired); err != nil {
		return err
	}
	if !o.acquired {
		return fmt.Errorf("runner observation unavailable")
	}
	for _, d := range desired {
		found := false
		for _, v := range o.entries {
			if d.ID != v.Value.ID {
				continue
			}
			found = true
			for _, n := range *d.CustomLabels {
				for _, ro := range v.ReadOnlyLabels {
					if RunnerLabelIdentity(n) == RunnerLabelIdentity(ro) {
						return fmt.Errorf("custom label collides with a read-only runner label")
					}
				}
			}
		}
		if !found {
			return fmt.Errorf("selected runner unavailable")
		}
	}
	return nil
}

type RunnerLabelsChange struct{ before, after RunnerLabels }

func NewRunnerLabelsChange(before, after RunnerLabels) (RunnerLabelsChange, error) {
	if ValidateRunnerLabels([]RunnerLabels{before}) != nil || ValidateRunnerLabels([]RunnerLabels{after}) != nil || before.ID != after.ID || RunnerLabelsMatch(before, after) {
		return RunnerLabelsChange{}, fmt.Errorf("invalid runner labels change")
	}
	return RunnerLabelsChange{CloneRunnerLabels([]RunnerLabels{before})[0], CloneRunnerLabels([]RunnerLabels{after})[0]}, nil
}
func (c RunnerLabelsChange) Values() (RunnerLabels, RunnerLabels) {
	return CloneRunnerLabels([]RunnerLabels{c.before})[0], CloneRunnerLabels([]RunnerLabels{c.after})[0]
}
func (c RunnerLabelsChange) Field() string { return RunnersPath }
func CompareRunnerLabels(o RunnerObservation, desired []RunnerLabels) (int, []RunnerLabelsChange, error) {
	if len(desired) == 0 {
		return 0, nil, nil
	}
	if err := PrepareRunnerLabels(o, desired); err != nil {
		return 0, nil, err
	}
	changes := []RunnerLabelsChange{}
	for _, d := range desired {
		for _, v := range o.entries {
			if d.ID == v.Value.ID && !RunnerLabelsMatch(v.Value, d) {
				c, err := NewRunnerLabelsChange(v.Value, d)
				if err != nil {
					return 0, nil, err
				}
				changes = append(changes, c)
			}
		}
	}
	return 1, changes, nil
}
