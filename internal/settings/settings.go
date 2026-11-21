package settings

import (
	"fmt"
	"slices"
)

// Declaration holds only explicitly managed values for each supported resource.
type Declaration struct {
	Webhooks              *[]Webhook
	Runners               *[]RunnerLabels
	DeployKeys            *DeployKeyDeclarations
	Autolinks             *[]Autolink
	BranchProtectionRules *[]ClassicRule
	OIDCSubject           *OIDCSubjectTemplate
	Collaborators         *CollaboratorDeclarations
	CopilotInstructions   *bool
	CopilotMCP            *bool
	PullRequestLimits     PullRequestLimitsDeclaration
	PullRequestExemptions *PullRequestExemptionsDeclaration
	CopilotReviewEffort   *string
	Repository            RepositoryDeclaration
	Topics                TopicsDeclaration
	Variables             *VariableDeclarations
	AgentsVariables       *VariableDeclarations
	Labels                *LabelDeclarations
	Rulesets              *RulesetDeclarations
	Environments          *EnvironmentDeclarations
	EnvironmentVariables  []EnvironmentVariableDeclaration
	Actions               ActionsPolicyDeclaration
	ActionsPolicies       *ActionsPolicyDeclarations
	Security              SecurityDeclaration
	Pages                 PagesDeclaration
}

// TopicsDeclaration distinguishes omitted names from an explicitly empty list.
type TopicsDeclaration struct{ Names *[]string }

// CollectionSelected reports whether entries in a declared collection need
// reconciliation. A present empty collection selects work only with prune.
func CollectionSelected[T interface{ Len() int }](entries *T, prune bool) bool {
	return entries != nil && (prune || (*entries).Len() > 0)
}

// SliceSelected applies the same selection to slice-backed declarations.
func SliceSelected[T any](entries *[]T, prune bool) bool {
	return entries != nil && (prune || len(*entries) > 0)
}

// Empty reports whether the declaration selects no work without pruning.
// Topics and pull request exemptions own their full list, even when empty.
func (d Declaration) Empty() bool {
	return d.Repository.Empty() &&
		d.Topics.Names == nil && d.PullRequestExemptions == nil &&
		d.Actions.Empty() && d.Security.Empty() && d.Pages.Empty() &&
		d.OIDCSubject == nil && d.PullRequestLimits.Empty() &&
		d.CopilotReviewEffort == nil && d.CopilotMCP == nil && d.CopilotInstructions == nil &&
		!CollectionSelected(d.Collaborators, false) &&
		!CollectionSelected(d.DeployKeys, false) &&
		!CollectionSelected(d.Labels, false) &&
		!CollectionSelected(d.Rulesets, false) &&
		!CollectionSelected(d.ActionsPolicies, false) &&
		!CollectionSelected(d.Variables, false) &&
		!CollectionSelected(d.AgentsVariables, false) &&
		!CollectionSelected(d.Environments, false) &&
		EnvironmentVariablesEmpty(d.EnvironmentVariables) &&
		!SliceSelected(d.Runners, false) &&
		!SliceSelected(d.Webhooks, false) &&
		!SliceSelected(d.Autolinks, false) &&
		!SliceSelected(d.BranchProtectionRules, false)
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
		result.Changes = []Change{{field: topicsField, values: topicsPair{before: before, after: after}}}
	}
	return result
}
