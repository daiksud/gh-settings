package cli

import (
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

func labelLine(change settings.LabelChange) string {
	prefix := fmt.Sprintf("settings.labels: %s %q", change.Operation, change.Name)
	switch change.Operation {
	case "delete":
		return prefix + " (removes issue/PR associations)"
	case "create":
		var attributes []string
		if change.After.Color != nil {
			attributes = append(attributes, fmt.Sprintf("color=%q", *change.After.Color))
		}
		if change.After.Description != nil {
			attributes = append(attributes, fmt.Sprintf("description=%q", *change.After.Description))
		}
		if len(attributes) > 0 {
			prefix += " " + strings.Join(attributes, " ")
		}
	case "update":
		var attributes []string
		if change.After.Color != nil {
			attributes = append(attributes, fmt.Sprintf("color: %q → %q", change.Before.Color, *change.After.Color))
		}
		if change.After.Description != nil {
			attributes = append(attributes, fmt.Sprintf("description: %q → %q", change.Before.Description, *change.After.Description))
		}
		prefix += " " + strings.Join(attributes, ", ")
	}
	return prefix
}
