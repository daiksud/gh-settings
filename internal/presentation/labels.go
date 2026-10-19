package presentation

import (
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

// LabelLine formats a native label operation without HTML or I/O.
func LabelLine(change settings.LabelChange) string {
	prefix := fmt.Sprintf("settings.labels: %s %q", change.Operation(), change.Name())
	change.Visit(func(after settings.LabelDeclaration) {
		var attributes []string
		if after.Color != nil {
			attributes = append(attributes, fmt.Sprintf("color=%q", *after.Color))
		}
		if after.Description != nil {
			attributes = append(attributes, fmt.Sprintf("description=%q", *after.Description))
		}
		if len(attributes) > 0 {
			prefix += " " + strings.Join(attributes, " ")
		}
	}, func(before settings.LabelState, after settings.LabelDeclaration) {
		var attributes []string
		if after.Color != nil {
			attributes = append(attributes, fmt.Sprintf("color: %q → %q", before.Color, *after.Color))
		}
		if after.Description != nil {
			attributes = append(attributes, fmt.Sprintf("description: %q → %q", before.Description, *after.Description))
		}
		prefix += " " + strings.Join(attributes, ", ")
	}, func(settings.LabelState) { prefix += " (removes issue/PR associations)" })
	return prefix
}
