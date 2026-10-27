package settings

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Deployment patterns have case-sensitive Git ref identities. API IDs are
// observation metadata, not declaration identity or YAML writeback inputs.
type EnvironmentPattern struct {
	Type string `json:"type" yaml:"type"`
	Name string `json:"name" yaml:"name"`
}

type EnvironmentPatternState struct {
	ID      int64
	Pattern EnvironmentPattern
}

type EnvironmentPatternChange struct {
	Operation string
	ID        int64
	Pattern   EnvironmentPattern
}

func normalizeEnvironmentPatterns(patterns []EnvironmentPattern) ([]EnvironmentPattern, error) {
	result := slices.Clone(patterns)
	if result == nil {
		result = []EnvironmentPattern{}
	}
	seen := map[EnvironmentPattern]bool{}
	for _, pattern := range result {
		if pattern.Type != "branch" && pattern.Type != "tag" {
			return nil, fmt.Errorf("environment deployment pattern type must be branch or tag")
		}
		if pattern.Name == "" || !utf8.ValidString(pattern.Name) || strings.ContainsFunc(pattern.Name, unicode.IsControl) {
			return nil, fmt.Errorf("environment deployment pattern name must be nonempty valid text without controls")
		}
		if seen[pattern] {
			return nil, fmt.Errorf("duplicate environment deployment pattern %s %q", pattern.Type, pattern.Name)
		}
		seen[pattern] = true
	}
	slices.SortFunc(result, func(a, b EnvironmentPattern) int {
		if order := cmp.Compare(a.Type, b.Type); order != 0 {
			return order
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return result, nil
}

// A nil declaration sends no pattern CRUD requests. An explicit empty list
// selects the pattern endpoint and removes all observed entries.
func ReconcileEnvironmentPatterns(current []EnvironmentPatternState, desired *[]EnvironmentPattern) ([]EnvironmentPatternChange, error) {
	if desired == nil {
		return nil, nil
	}
	patterns, err := normalizeEnvironmentPatterns(*desired)
	if err != nil {
		return nil, err
	}
	observed := map[EnvironmentPattern]int64{}
	ids := map[int64]bool{}
	for _, entry := range current {
		if _, err := normalizeEnvironmentPatterns([]EnvironmentPattern{entry.Pattern}); err != nil {
			return nil, err
		}
		if entry.ID <= 0 || ids[entry.ID] || observed[entry.Pattern] != 0 {
			return nil, fmt.Errorf("ambiguous environment deployment pattern observation")
		}
		ids[entry.ID], observed[entry.Pattern] = true, entry.ID
	}
	wanted := map[EnvironmentPattern]bool{}
	changes := []EnvironmentPatternChange{}
	for _, pattern := range patterns {
		wanted[pattern] = true
		if observed[pattern] == 0 {
			changes = append(changes, EnvironmentPatternChange{Operation: "create", Pattern: pattern})
		}
	}
	// Create first, then remove obsolete entries. This is ordered, not atomic,
	// and does not promise enforcement during policy-mode transitions.
	removals := []EnvironmentPatternChange{}
	for pattern, id := range observed {
		if !wanted[pattern] {
			removals = append(removals, EnvironmentPatternChange{Operation: "delete", ID: id, Pattern: pattern})
		}
	}
	slices.SortFunc(removals, func(a, b EnvironmentPatternChange) int { return cmp.Compare(a.ID, b.ID) })
	return append(changes, removals...), nil
}
