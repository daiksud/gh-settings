package settings

import (
	"fmt"
	"slices"
	"strings"
)

func NewActionsEventPolicyDelete(before ActionsEventPolicy) (ActionsEventPolicyChange, error) {
	if before.ID <= 0 {
		return ActionsEventPolicyChange{}, fmt.Errorf("Actions policy deletion requires positive id")
	}
	if err := ValidateActionsEventPolicyState(before); err != nil {
		return ActionsEventPolicyChange{}, err
	}
	owned := canonicalActionsEventPolicy(before)
	return ActionsEventPolicyChange{before: &owned, deleted: true}, nil
}

func ReconcileActionsPoliciesWithPrune(observed ActionsPolicyObservation, desired *ActionsPolicyDeclarations, prune bool) ([]ActionsEventPolicyChange, error) {
	changes, err := ReconcileActionsPolicies(observed, desired)
	if err != nil || !prune || desired == nil {
		return changes, err
	}
	values, err := observed.Values()
	if err != nil {
		return nil, err
	}
	keep := map[int64]bool{}
	for _, value := range desired.Values() {
		if value.ID > 0 {
			keep[value.ID] = true
		}
	}
	slices.SortFunc(values, func(a, b ActionsEventPolicyState) int {
		if order := strings.Compare(a.Name, b.Name); order != 0 {
			return order
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	names := map[string]bool{}
	for _, value := range values {
		if keep[value.ID] {
			continue
		}
		if names[value.Name] {
			return nil, fmt.Errorf("ambiguous Actions policy deletion name %q", value.Name)
		}
		names[value.Name] = true
		if value.Policy == nil {
			return nil, fmt.Errorf("cannot prune unsupported Actions policy id %d: %s", value.ID, value.Unsupported)
		}
		change, err := NewActionsEventPolicyDelete(*value.Policy)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}

func CompareActionsPoliciesWithPrune(observed ActionsPolicyObservation, desired *ActionsPolicyDeclarations, prune bool) (Comparison, error) {
	changes, err := ReconcileActionsPoliciesWithPrune(observed, desired, prune)
	if err != nil {
		return Comparison{}, err
	}
	result := Comparison{ActionsPolicies: changes}
	if desired != nil && (desired.Len() > 0 || prune) {
		result.Managed = 1
	}
	return result, nil
}
