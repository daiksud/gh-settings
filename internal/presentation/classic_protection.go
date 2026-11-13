package presentation

import (
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
)

func ClassicProtectionLine(c settings.ClassicChange) string {
	names := []string{"require_conversation_resolution_before_merging", "require_signed_commits", "require_linear_history", "allow_deletions", "lock_branch", "do_not_allow_bypassing_the_above_settings"}
	if c.After == nil {
		parts := []string{fmt.Sprintf("branch_name_pattern: %q", c.Before.BranchNamePattern)}
		for i, value := range settings.ClassicFlags(*c.Before) {
			parts = append(parts, fmt.Sprintf("%s: %t", names[i], *value))
		}
		if c.Before.AllowForkSyncing != nil {
			parts = append(parts, fmt.Sprintf("allow_fork_syncing: %t", *c.Before.AllowForkSyncing))
		}
		parts = append(parts, classicStatusChecksLine(c.Before.RequireStatusChecksToPassBeforeMerging))
		parts = append(parts, classicReviewsLine(c.Before.RequireAPullRequestBeforeMerging))
		parts = append(parts, classicDeploymentLine(c.Before.RequireDeploymentsToSucceedBeforeMerging))
		return fmt.Sprintf("%s: delete (id %d) before {%s} → absent (removes the whole rule, including other configured protections; an overlapping rule may take effect)", settings.ClassicProtectionPath, *c.Before.ID, strings.Join(parts, ", "))
	}
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
	var before []*bool
	if c.Before != nil {
		before = settings.ClassicFlags(*c.Before)
	}
	for i, v := range settings.ClassicFlags(*c.After) {
		if v == nil {
			continue
		}
		if c.Before == nil {
			parts = append(parts, fmt.Sprintf("%s: %t", names[i], *v))
		} else {
			parts = append(parts, fmt.Sprintf("%s: %t → %t", names[i], *before[i], *v))
		}
	}
	if c.After.AllowForkSyncing != nil {
		value := fmt.Sprintf("allow_fork_syncing: %t", *c.After.AllowForkSyncing)
		if c.Before != nil {
			previous := "inactive"
			if c.Before.AllowForkSyncing != nil {
				previous = fmt.Sprintf("%t", *c.Before.AllowForkSyncing)
			}
			value = fmt.Sprintf("allow_fork_syncing: %s → %t", previous, *c.After.AllowForkSyncing)
		}
		parts = append(parts, value)
	}
	if settings.ClassicReviewsManaged(c.After.RequireAPullRequestBeforeMerging) {
		value := classicReviewsLine(c.After.RequireAPullRequestBeforeMerging)
		if c.Before != nil {
			value = classicReviewsLine(c.Before.RequireAPullRequestBeforeMerging) + " → " + value
		}
		parts = append(parts, value)
	}
	if settings.ClassicDeploymentsSelected(c.After.RequireDeploymentsToSucceedBeforeMerging) {
		value := classicDeploymentLine(c.After.RequireDeploymentsToSucceedBeforeMerging)
		if c.Before != nil {
			value = classicDeploymentLine(c.Before.RequireDeploymentsToSucceedBeforeMerging) + " → " + value
		}
		parts = append(parts, value)
	}
	line := fmt.Sprintf("%s: %s%s %s", settings.ClassicProtectionPath, operation, identity, strings.Join(parts, ", "))
	if settings.ClassicStatusChecksManaged(c.After.RequireStatusChecksToPassBeforeMerging) {
		line += ", " + classicStatusChecksLine(c.After.RequireStatusChecksToPassBeforeMerging)
	}
	if c.Before == nil {
		line += " (omitted controls are API-controlled on creation)"
	}
	return line
}

func classicStatusChecksLine(group *settings.ClassicStatusChecks) string {
	data, _ := json.Marshal(group)
	return "require_status_checks_to_pass_before_merging: " + string(data)
}
func classicReviewsLine(group *settings.ClassicPullRequestReviews) string {
	data, _ := json.Marshal(group)
	return "require_a_pull_request_before_merging: " + string(data)
}

func classicDeploymentLine(group *settings.ClassicDeployments) string {
	data, _ := json.Marshal(group)
	return "require_deployments_to_succeed_before_merging: " + string(data)
}
