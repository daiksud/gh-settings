package settings

import (
	"fmt"
	"slices"
)

// Declaration holds only explicitly managed values for each supported resource.
type Declaration struct {
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

func (d Declaration) Empty() bool {
	return d.OIDCSubject == nil && (d.Collaborators == nil || d.Collaborators.Len() == 0) && (d.BranchProtectionRules == nil || len(*d.BranchProtectionRules) == 0) && d.PullRequestExemptions == nil && EnvironmentVariablesEmpty(d.EnvironmentVariables) && d.CopilotReviewEffort == nil && d.PullRequestLimits.Empty() && d.CopilotMCP == nil && d.CopilotInstructions == nil && (d.Variables == nil || d.Variables.Len() == 0) && (d.AgentsVariables == nil || d.AgentsVariables.Len() == 0) && d.Repository.Empty() && d.Topics.Names == nil && (d.Labels == nil || d.Labels.Len() == 0) && (d.Rulesets == nil || d.Rulesets.Len() == 0) && (d.Environments == nil || d.Environments.Len() == 0) && (d.ActionsPolicies == nil || d.ActionsPolicies.Len() == 0) && d.Actions.Empty() && d.Security.Empty() && d.Pages.Empty()
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
