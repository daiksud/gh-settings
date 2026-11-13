package presentation

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
)

func ClassicProtectionLine(c settings.ClassicChange) string {
	operation := "create"
	identity := ""
	if c.Before != nil {
		operation = "update"
		identity = fmt.Sprintf(" (id %d)", *c.Before.ID)
	}
	parts := []string{fmt.Sprintf("branch_name_pattern: %q", c.After.BranchNamePattern)}
	if c.Before != nil && c.Before.BranchNamePattern != c.After.BranchNamePattern {
		parts[0] = fmt.Sprintf("branch_name_pattern: %q → %q", c.Before.BranchNamePattern, c.After.BranchNamePattern)
	}
	names := []string{"require_conversation_resolution_before_merging", "require_signed_commits", "require_linear_history", "allow_deletions", "lock_branch"}
	var before []*bool
	if c.Before != nil {
		before = settings.ClassicFlags(*c.Before)
	}
	for i, v := range settings.ClassicFlags(c.After) {
		if v == nil {
			continue
		}
		if c.Before == nil {
			parts = append(parts, fmt.Sprintf("%s: %t", names[i], *v))
		} else {
			parts = append(parts, fmt.Sprintf("%s: %t → %t", names[i], *before[i], *v))
		}
	}
	line := fmt.Sprintf("%s: %s%s %s", settings.ClassicProtectionPath, operation, identity, strings.Join(parts, ", "))
	if c.Before == nil {
		line += " (omitted controls are API-controlled on creation)"
	}
	return line
}
