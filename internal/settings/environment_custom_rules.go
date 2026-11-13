package settings

import (
	"fmt"
	"slices"
)

// EnvironmentCustomRule is acquired binding identity, never a YAML declaration.
// App metadata is immutable comparison evidence within one reconciliation.
type EnvironmentCustomRule struct {
	ID, AppID                               int64
	NodeID, AppNodeID, Slug, IntegrationURL string
}

func normalizeEnvironmentCustomRules(ids []int64) ([]int64, error) {
	if len(ids) > 6 {
		return nil, fmt.Errorf("environment custom_rules supports at most six App IDs")
	}
	values := slices.Clone(ids)
	if values == nil {
		values = []int64{}
	}
	slices.Sort(values)
	for i, id := range values {
		if id <= 0 || (i > 0 && values[i-1] == id) {
			return nil, fmt.Errorf("environment custom_rules requires unique positive App IDs")
		}
	}
	return values, nil
}

func EnvironmentCustomRuleIDs(rules []EnvironmentCustomRule) ([]int64, error) {
	if rules == nil {
		return nil, fmt.Errorf("environment custom protection rules were not acquired")
	}
	ids, bindings, nodes, appNodes := []int64{}, map[int64]bool{}, map[string]bool{}, map[string]bool{}
	for _, rule := range rules {
		if rule.ID <= 0 || rule.AppID <= 0 || rule.NodeID == "" || rule.AppNodeID == "" || rule.Slug == "" || rule.IntegrationURL == "" || bindings[rule.ID] || nodes[rule.NodeID] || appNodes[rule.AppNodeID] {
			return nil, fmt.Errorf("invalid or ambiguous environment custom protection rule identity")
		}
		bindings[rule.ID], nodes[rule.NodeID], appNodes[rule.AppNodeID] = true, true, true
		ids = append(ids, rule.AppID)
	}
	return normalizeEnvironmentCustomRules(ids)
}

// EnvironmentCustomRuleChanges freezes exact deletion bindings and sorted App
// additions. Capacity is a necessary custom-App bound, not an invented formula
// for the server's combined built-in/custom rule limit.
func EnvironmentCustomRuleChanges(current []EnvironmentCustomRule, desired *[]int64) ([]int64, []EnvironmentCustomRule, error) {
	ids, err := EnvironmentCustomRuleIDs(current)
	if err != nil {
		return nil, nil, err
	}
	if desired == nil {
		return nil, nil, nil
	}
	want, err := normalizeEnvironmentCustomRules(*desired)
	if err != nil {
		return nil, nil, err
	}
	add, remove := []int64{}, []EnvironmentCustomRule{}
	for _, id := range want {
		if !slices.Contains(ids, id) {
			add = append(add, id)
		}
	}
	if len(ids)+len(add) > 6 {
		return nil, nil, fmt.Errorf("environment custom_rules replacement exceeds six temporary App bindings; first apply an explicit reduced custom_rules set, then add the replacements (protection changes between applies)")
	}
	for _, rule := range current {
		if !slices.Contains(want, rule.AppID) {
			remove = append(remove, rule)
		}
	}
	slices.SortFunc(remove, func(a, b EnvironmentCustomRule) int {
		if a.AppID < b.AppID {
			return -1
		}
		if a.AppID > b.AppID {
			return 1
		}
		return 0
	})
	return add, remove, nil
}

func SameEnvironmentCustomRules(a, b []EnvironmentCustomRule) bool {
	if _, err := EnvironmentCustomRuleIDs(a); err != nil {
		return false
	}
	if _, err := EnvironmentCustomRuleIDs(b); err != nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	byID := map[int64]EnvironmentCustomRule{}
	for _, rule := range a {
		byID[rule.ID] = rule
	}
	for _, rule := range b {
		if byID[rule.ID] != rule {
			return false
		}
	}
	return true
}
