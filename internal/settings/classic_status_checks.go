package settings

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

type ClassicStatusChecks = uimodel.ClassicStatusChecks
type ClassicStatusCheck = uimodel.ClassicStatusCheck

func ClassicStatusChecksManaged(g *ClassicStatusChecks) bool { return g != nil && g.Enabled != nil }

func ValidateClassicStatusChecks(g *ClassicStatusChecks) error {
	if g == nil {
		return nil
	}
	if g.Enabled == nil {
		if g.RequireBranchesToBeUpToDateBeforeMerging != nil || g.StatusChecks != nil {
			return fmt.Errorf("classic status checks require enabled")
		}
		return nil
	}
	if !*g.Enabled {
		if g.RequireBranchesToBeUpToDateBeforeMerging != nil || g.StatusChecks != nil {
			return fmt.Errorf("disabled classic status checks forbid child controls")
		}
		return nil
	}
	if g.RequireBranchesToBeUpToDateBeforeMerging == nil || g.StatusChecks == nil {
		return fmt.Errorf("enabled classic status checks require strictness and the complete status_checks list")
	}
	seen := map[string]bool{}
	for _, check := range *g.StatusChecks {
		if !classicCheckText(check.Name) || !classicCheckText(check.AcceptUpdatesFrom) || seen[check.Name] {
			return fmt.Errorf("classic status checks require unique exact names and explicit valid App sources")
		}
		seen[check.Name] = true
	}
	return nil
}

func classicCheckText(v string) bool {
	return strings.TrimSpace(v) != "" && !strings.ContainsFunc(v, unicode.IsControl)
}

func CloneClassicStatusChecks(g *ClassicStatusChecks) *ClassicStatusChecks {
	if g == nil {
		return nil
	}
	v := *g
	v.Enabled = cloneClassicValue(g.Enabled)
	v.RequireBranchesToBeUpToDateBeforeMerging = cloneClassicValue(g.RequireBranchesToBeUpToDateBeforeMerging)
	if g.StatusChecks != nil {
		checks := append([]ClassicStatusCheck{}, (*g.StatusChecks)...)
		v.StatusChecks = &checks
	}
	return &v
}

func orderedClassicChecks(checks []ClassicStatusCheck) []ClassicStatusCheck {
	result := append([]ClassicStatusCheck{}, checks...)
	slices.SortFunc(result, func(a, b ClassicStatusCheck) int { return strings.Compare(a.Name, b.Name) })
	return result
}

func ClassicStatusChecksMatch(observed, desired *ClassicStatusChecks) bool {
	if !ClassicStatusChecksManaged(desired) {
		return true
	}
	if ValidateClassicStatusChecks(desired) != nil || !ClassicStatusChecksManaged(observed) || ValidateClassicStatusChecks(observed) != nil || *observed.Enabled != *desired.Enabled {
		return false
	}
	if !*desired.Enabled {
		return true
	}
	return *observed.RequireBranchesToBeUpToDateBeforeMerging == *desired.RequireBranchesToBeUpToDateBeforeMerging && reflect.DeepEqual(orderedClassicChecks(*observed.StatusChecks), orderedClassicChecks(*desired.StatusChecks))
}

// Projection never changes the private nullable, ordered snapshot. Coherence and
// unique names are conservative supported-observation boundaries, not API promises.
func ProjectClassicStatusChecks(raw ClassicCompanions) (*ClassicStatusChecks, error) {
	if !raw.Contexts.Present || !raw.Checks.Present {
		return nil, fmt.Errorf("missing classic status-check acquisition")
	}
	if raw.Checks.Value != nil {
		for _, check := range *raw.Checks.Value {
			if check.AppID != nil && (!classicCheckText(*check.AppID) || *check.AppID == "any") {
				return nil, fmt.Errorf("invalid acquired classic status-check App identity")
			}
		}
	}
	enabled := raw.RequiresStatusChecks
	result := &ClassicStatusChecks{Enabled: &enabled}
	if !enabled {
		return result, nil
	}
	if raw.Contexts.Value == nil || raw.Checks.Value == nil {
		return nil, fmt.Errorf("enabled classic status checks have unavailable lists")
	}
	checks := []ClassicStatusCheck{}
	names := map[string]bool{}
	for _, check := range *raw.Checks.Value {
		source := "any"
		if check.AppID != nil {
			source = *check.AppID
		}
		checks = append(checks, ClassicStatusCheck{Name: check.Context, AcceptUpdatesFrom: source})
		names[check.Context] = true
	}
	result.StatusChecks = &checks
	strict := raw.RequiresStrictStatusChecks
	result.RequireBranchesToBeUpToDateBeforeMerging = &strict
	if err := ValidateClassicStatusChecks(result); err != nil {
		return nil, err
	}
	legacy := map[string]bool{}
	for _, name := range *raw.Contexts.Value {
		if !classicCheckText(name) || legacy[name] || !names[name] {
			return nil, fmt.Errorf("classic status-check aliases are incoherent")
		}
		legacy[name] = true
	}
	if len(legacy) != len(names) {
		return nil, fmt.Errorf("classic status-check aliases are incoherent")
	}
	checks = orderedClassicChecks(checks)
	return result, nil
}

// Input owns only the selected active union. Explicit any prevents GitHub from
// inferring a recently observed provider when appId is omitted.
func ClassicStatusChecksInput(before, desired *ClassicStatusChecks) map[string]any {
	input := map[string]any{}
	if !ClassicStatusChecksManaged(desired) {
		return input
	}
	active := *desired.Enabled
	initial := !ClassicStatusChecksManaged(before)
	if initial || *before.Enabled != active {
		input["requiresStatusChecks"] = active
	}
	if !active {
		return input
	}
	enabling := initial || !*before.Enabled
	if enabling || *before.RequireBranchesToBeUpToDateBeforeMerging != *desired.RequireBranchesToBeUpToDateBeforeMerging {
		input["requiresStrictStatusChecks"] = *desired.RequireBranchesToBeUpToDateBeforeMerging
	}
	if enabling || !reflect.DeepEqual(orderedClassicChecks(*before.StatusChecks), orderedClassicChecks(*desired.StatusChecks)) {
		checks := []map[string]string{}
		for _, check := range orderedClassicChecks(*desired.StatusChecks) {
			checks = append(checks, map[string]string{"context": check.Name, "appId": check.AcceptUpdatesFrom})
		}
		input["requiredStatusChecks"] = checks
	}
	return input
}

func ClassicStatusCheckAppIDs(g *ClassicStatusChecks) []string {
	ids := []string{}
	if !ClassicStatusChecksManaged(g) || !*g.Enabled || g.StatusChecks == nil {
		return ids
	}
	for _, check := range *g.StatusChecks {
		if check.AcceptUpdatesFrom != "any" && !slices.Contains(ids, check.AcceptUpdatesFrom) {
			ids = append(ids, check.AcceptUpdatesFrom)
		}
	}
	slices.Sort(ids)
	return ids
}
