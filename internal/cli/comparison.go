package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

func RenderComparison(comparison settings.Comparison) []byte {
	var output strings.Builder
	if comparison.Managed == 0 {
		output.WriteString("No managed repository settings.\n")
	} else if !comparison.HasChanges() {
		output.WriteString("No differences for managed repository settings.\n")
	} else {
		shownMenus := map[settings.CommitMessageMenu]bool{}
		for _, change := range comparison.Changes {
			if context, ok := presentation.CommitMessageContextForField(change.Field().Path(), comparison.CommitMessageContexts); ok {
				if !shownMenus[context.Menu] {
					fmt.Fprintln(&output, presentation.ExactCommitMessageLine(context))
					shownMenus[context.Menu] = true
				}
				continue
			}
			if change.Normalization() != (settings.ObservationNormalization{}) {
				fmt.Fprintln(&output, presentation.CreationPolicyLine(change))
				continue
			}
			line := func(before, after string) {
				fmt.Fprintf(&output, "%s: %s → %s\n", presentation.SettingPath(change.Field().Path()), before, after)
			}
			change.Visit(
				func(before, after bool) { line(strconv.FormatBool(before), strconv.FormatBool(after)) },
				func(before, after string) {
					before, after = presentation.StringValues(change.Field().Path(), before, after)
					line(strconv.Quote(before), strconv.Quote(after))
				},
				func(before, after []string) { line(topicsValue(before), topicsValue(after)) },
			)
			if note := presentation.ArchiveOrder(change); note != "" {
				fmt.Fprintln(&output, note)
			}
		}
		for _, change := range comparison.CommitMessages {
			fmt.Fprintln(&output, presentation.CommitMessageLine(change))
		}
		for _, change := range comparison.Labels.Values() {
			fmt.Fprintln(&output, presentation.LabelLine(change))
		}
	}
	for _, change := range comparison.Variables {
		fmt.Fprintln(&output, presentation.VariableLine(change))
	}
	for _, change := range comparison.EnvironmentVariables {
		fmt.Fprintln(&output, presentation.EnvironmentVariableLine(change))
	}
	for _, change := range comparison.AgentsVariables {
		fmt.Fprintln(&output, presentation.AgentVariableLine(change))
	}
	for _, change := range comparison.CopilotInstructions {
		fmt.Fprintln(&output, presentation.CopilotInstructionsLine(change))
	}
	for _, change := range comparison.OIDCSubject {
		fmt.Fprintln(&output, presentation.OIDCSubjectLine(change))
	}
	for _, change := range comparison.CopilotMCP {
		fmt.Fprintln(&output, presentation.CopilotMCPLine(change))
	}
	for _, change := range comparison.PullRequestExemptions {
		fmt.Fprintln(&output, presentation.PullRequestExemptionsLine(change))
	}
	for _, change := range comparison.PullRequestLimits {
		fmt.Fprintln(&output, presentation.PullRequestLimitsLine(change))
	}
	for _, change := range comparison.CopilotReviewEffort {
		fmt.Fprintln(&output, presentation.CopilotReviewEffortLine(change))
	}
	for _, change := range comparison.ActionsPolicies {
		fmt.Fprintln(&output, presentation.ActionsEventPolicyLine(change))
	}
	for _, change := range comparison.Rulesets {
		fmt.Fprintln(&output, presentation.RulesetLine(change.Display()))
	}
	for _, change := range comparison.BranchProtectionRules {
		fmt.Fprintln(&output, presentation.ClassicProtectionLine(change))
	}
	for _, change := range comparison.Environments {
		fmt.Fprintln(&output, presentation.EnvironmentLine(change))
	}
	for _, change := range comparison.ActionsExecution {
		fmt.Fprintln(&output, presentation.ActionsExecutionLine(change))
	}
	shownActionsExecution := false
	for _, change := range comparison.Actions {
		if presentation.IsActionsExecutionField(change.Field()) {
			if !shownActionsExecution {
				fmt.Fprintln(&output, presentation.ExactActionsExecutionLine(comparison.ActionsExecutionContext))
				shownActionsExecution = true
			}
			continue
		}
		fmt.Fprintln(&output, presentation.ActionsPolicyLine(change))
	}
	for _, c := range comparison.CodeQLConfiguration {
		fmt.Fprintln(&output, presentation.CodeQLConfigurationLine(c))
	}
	for _, change := range comparison.Security {
		fmt.Fprintln(&output, presentation.SecurityLine(change))
	}
	for _, c := range comparison.Pages {
		fmt.Fprintln(&output, presentation.PagesLine(c))
	}
	for _, deletion := range comparison.CollaboratorDeletions {
		fmt.Fprintln(&output, presentation.CollaboratorDeletionLine(deletion))
	}
	for _, person := range comparison.Collaborators {
		fmt.Fprintln(&output, presentation.CollaboratorLine(person))
	}
	return []byte(output.String())
}

func topicsValue(names []string) string {
	data, _ := json.Marshal(names)
	return string(data)
}
