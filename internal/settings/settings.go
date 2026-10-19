package settings

import (
	"fmt"
	"slices"
)

// Declaration holds only explicitly managed values for each supported resource.
type Declaration struct {
	Repository RepositoryDeclaration
	Topics     TopicsDeclaration
	Labels     *[]LabelDeclaration
}

// TopicsDeclaration distinguishes omitted names from an explicitly empty list.
type TopicsDeclaration struct{ Names *[]string }

func (d Declaration) Empty() bool {
	return d.Repository.Empty() && d.Topics.Names == nil && d.Labels == nil
}

// ValidateTopics follows GitHub's canonical topic naming rules.
func ValidateTopics(names []string) error {
	if len(names) > 20 {
		return fmt.Errorf("topics.names must contain at most 20 topics")
	}
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if len(name) == 0 || len(name) > 50 {
			return fmt.Errorf("topic names must contain 1 to 50 lowercase letters, numbers or hyphens")
		}
		for _, c := range name {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return fmt.Errorf("topic names must contain only lowercase letters, numbers or hyphens")
			}
		}
		if seen[name] {
			return fmt.Errorf("duplicate topic %q", name)
		}
		seen[name] = true
	}
	return nil
}

// SortedTopics returns an independent, non-nil slice for stable output and payloads.
func SortedTopics(names []string) []string {
	result := append([]string{}, names...)
	slices.Sort(result)
	return result
}

func CompareTopics(current []string, desired TopicsDeclaration) Comparison {
	if desired.Names == nil {
		return Comparison{}
	}
	before, after := SortedTopics(current), SortedTopics(*desired.Names)
	result := Comparison{Managed: 1}
	if !slices.Equal(before, after) {
		result.Changes = []Change{{field: topicsField, Before: before, After: after}}
	}
	return result
}
