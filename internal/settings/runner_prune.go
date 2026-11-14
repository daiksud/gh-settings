package settings

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// RunnerPruneObservation is private operational evidence. It must never be reported.
type RunnerPruneTarget struct {
	ID, OwnerID int64
	Archived    bool
}
type RunnerPruneLabel struct {
	ID         *int64
	Name, Kind string
}
type RunnerPruneState struct {
	ID                                       int64
	Name, OS, Status                         string
	Busy                                     bool
	Ephemeral                                *bool
	Version                                  *string
	Labels                                   []RunnerPruneLabel
	ListLabels, DetailLabels, EndpointLabels []RunnerPruneLabel
	HasOperationalReads                      bool
	ListStatus, LastStatus                   string
	ListBusy, LastBusy                       bool
	ListVersion, LastVersion                 *string
}
type RunnerPruneObservation struct {
	Target  RunnerPruneTarget
	Entries []RunnerPruneState
}

func (o RunnerPruneObservation) Validate(desired []RunnerLabels) error {
	if o.Target.ID <= 0 || o.Target.OwnerID <= 0 || o.Entries == nil {
		return fmt.Errorf("incomplete runner inventory")
	}
	if err := ValidateRunnerLabels(desired); err != nil {
		return err
	}
	ids := map[int64]bool{}
	for _, v := range o.Entries {
		if v.ID <= 0 || ids[v.ID] || strings.TrimSpace(v.Name) == "" || strings.TrimSpace(v.OS) == "" || (v.Status != "online" && v.Status != "offline") || v.Labels == nil || (v.Version != nil && strings.TrimSpace(*v.Version) == "") {
			return fmt.Errorf("invalid runner inventory evidence")
		}
		ids[v.ID] = true
		names := map[string]bool{}
		labelIDs := map[int64]bool{}
		for _, l := range v.Labels {
			k := RunnerLabelIdentity(l.Name)
			if ValidateRunnerLabelName(l.Name) != nil || names[k] || (l.Kind != "custom" && l.Kind != "read-only") || (l.ID != nil && (*l.ID <= 0 || labelIDs[*l.ID])) {
				return fmt.Errorf("invalid runner label evidence")
			}
			names[k] = true
			if l.ID != nil {
				labelIDs[*l.ID] = true
			}
		}
	}
	for _, d := range desired {
		if !ids[d.ID] {
			return fmt.Errorf("selected runner unavailable")
		}
		for _, v := range o.Entries {
			if v.ID == d.ID {
				for _, n := range *d.CustomLabels {
					for _, l := range v.Labels {
						if l.Kind == "read-only" && RunnerLabelIdentity(l.Name) == RunnerLabelIdentity(n) {
							return fmt.Errorf("custom label collides with read-only runner label")
						}
					}
				}
			}
		}
	}
	return nil
}
func (o RunnerPruneObservation) Candidates(desired []RunnerLabels) ([]RunnerPruneState, error) {
	if err := o.Validate(desired); err != nil {
		return nil, err
	}
	keep := map[int64]bool{}
	for _, d := range desired {
		keep[d.ID] = true
	}
	result := []RunnerPruneState{}
	for _, v := range o.Entries {
		if !keep[v.ID] {
			if !runnerCandidateCoherent(v) {
				return nil, fmt.Errorf("runner prune candidate is busy or operational evidence changed")
			}
			result = append(result, v)
		}
	}
	slices.SortFunc(result, func(a, b RunnerPruneState) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return result, nil
}
func RunnerPruneCustomLabels(v RunnerPruneState) []string {
	out := []string{}
	for _, l := range v.Labels {
		if l.Kind == "custom" {
			out = append(out, l.Name)
		}
	}
	slices.Sort(out)
	return out
}
func (o RunnerPruneObservation) AfterUnarchive() RunnerPruneObservation {
	o.Target.Archived = false
	return o
}
func (o RunnerPruneObservation) Without(id int64) (RunnerPruneObservation, error) {
	out := RunnerPruneObservation{Target: o.Target, Entries: []RunnerPruneState{}}
	found := false
	for _, v := range o.Entries {
		if v.ID == id {
			found = true
			continue
		}
		out.Entries = append(out.Entries, v)
	}
	if !found {
		return out, fmt.Errorf("runner candidate missing")
	}
	return out, nil
}

// Verify preserves all configuration; retained operational values may change. A
// confirmed own update may replace only that retained runner's custom labels.
func VerifyRunnerPrune(expected, actual RunnerPruneObservation, candidates []int64, own *RunnerLabels) error {
	if err := actual.Validate(nil); err != nil {
		return err
	}
	if expected.Target != actual.Target || len(expected.Entries) != len(actual.Entries) {
		return fmt.Errorf("runner inventory changed")
	}
	candidate := map[int64]bool{}
	for _, id := range candidates {
		candidate[id] = true
	}
	for _, a := range expected.Entries {
		var b RunnerPruneState
		found := false
		for _, v := range actual.Entries {
			if v.ID == a.ID {
				b = v
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("runner inventory identity changed")
		}
		if a.Name != b.Name || a.OS != b.OS || !reflect.DeepEqual(a.Ephemeral, b.Ephemeral) {
			return fmt.Errorf("runner configuration changed")
		}
		if candidate[a.ID] && (a.Status != b.Status || !runnerCandidateCoherent(b) || !reflect.DeepEqual(a.Version, b.Version)) {
			return fmt.Errorf("runner candidate operational state changed or busy")
		}
		if own != nil && own.ID == a.ID {
			if candidate[a.ID] || !RunnerLabelsMatch(RunnerLabels{ID: b.ID, CustomLabels: ptrRunnerStrings(RunnerPruneCustomLabels(b))}, *own) {
				return fmt.Errorf("runner label update not confirmed")
			}
			a.Labels = runnerReadonly(a.Labels)
			b.Labels = runnerReadonly(b.Labels)
			a.ListLabels = runnerReadonly(a.ListLabels)
			b.ListLabels = runnerReadonly(b.ListLabels)
			a.DetailLabels = runnerReadonly(a.DetailLabels)
			b.DetailLabels = runnerReadonly(b.DetailLabels)
			a.EndpointLabels = runnerReadonly(a.EndpointLabels)
			b.EndpointLabels = runnerReadonly(b.EndpointLabels)
		}
		if !reflect.DeepEqual(a.Labels, b.Labels) || !reflect.DeepEqual(a.ListLabels, b.ListLabels) || !reflect.DeepEqual(a.DetailLabels, b.DetailLabels) || !reflect.DeepEqual(a.EndpointLabels, b.EndpointLabels) {
			return fmt.Errorf("runner label evidence changed")
		}
	}
	return nil
}
func runnerReadonly(v []RunnerPruneLabel) []RunnerPruneLabel {
	out := []RunnerPruneLabel{}
	for _, l := range v {
		if l.Kind == "read-only" {
			out = append(out, l)
		}
	}
	return out
}
func ptrRunnerStrings(v []string) *[]string { return &v }

func (o RunnerPruneObservation) LabelsObservation() (RunnerObservation, error) {
	states := []RunnerState{}
	for _, v := range o.Entries {
		custom := RunnerPruneCustomLabels(v)
		readonly := []string{}
		for _, l := range v.Labels {
			if l.Kind == "read-only" {
				readonly = append(readonly, l.Name)
			}
		}
		states = append(states, RunnerState{Value: RunnerLabels{ID: v.ID, CustomLabels: &custom}, ReadOnlyLabels: readonly})
	}
	return NewRunnerObservation(states)
}

func runnerCandidateCoherent(v RunnerPruneState) bool {
	if v.Busy {
		return false
	}
	if !v.HasOperationalReads {
		return true
	}
	return !v.ListBusy && !v.LastBusy && v.ListStatus == v.Status && v.LastStatus == v.Status && reflect.DeepEqual(v.ListVersion, v.Version) && reflect.DeepEqual(v.LastVersion, v.Version)
}
