package cli

import (
	"fmt"
	"strings"

	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

func RenderComparison(comparison settings.Comparison) []byte {
	var output strings.Builder
	writeLine := func(line string) { fmt.Fprintln(&output, line) }
	if comparison.Managed == 0 {
		output.WriteString("No managed repository settings.\n")
	} else if !comparison.HasChanges() {
		output.WriteString("No differences for managed repository settings.\n")
	} else {
		presentation.RenderRepositoryChanges(&comparison, writeLine)
		presentation.RenderChanges(comparison.Labels.Values(), presentation.LabelLine, writeLine)
	}
	presentation.RenderChanges(comparison.Variables, presentation.VariableLine, writeLine)
	presentation.RenderChanges(comparison.EnvironmentVariables, presentation.EnvironmentVariableLine, writeLine)
	presentation.RenderChanges(comparison.AgentsVariables, presentation.AgentVariableLine, writeLine)
	presentation.RenderChanges(comparison.CopilotInstructions, presentation.CopilotInstructionsLine, writeLine)
	presentation.RenderChanges(comparison.Runners, presentation.RunnerLabelsLine, writeLine)
	presentation.RenderChanges(comparison.RunnerDeletions, presentation.RunnerDeletionLine, writeLine)
	presentation.RenderChanges(comparison.OIDCSubject, presentation.OIDCSubjectLine, writeLine)
	presentation.RenderChanges(comparison.CopilotMCP, presentation.CopilotToggleLine, writeLine)
	presentation.RenderChanges(comparison.CopilotPublicSearch, presentation.CopilotToggleLine, writeLine)
	presentation.RenderChanges(comparison.PullRequestExemptions, presentation.PullRequestExemptionsLine, writeLine)
	presentation.RenderChanges(comparison.PullRequestLimits, presentation.PullRequestLimitsLine, writeLine)
	presentation.RenderChanges(comparison.CopilotReviewEffort, presentation.CopilotReviewEffortLine, writeLine)
	presentation.RenderChanges(comparison.ActionsPolicies, presentation.ActionsEventPolicyLine, writeLine)
	for _, change := range comparison.Rulesets {
		writeLine(presentation.RulesetLine(change.Display()))
	}
	presentation.RenderChanges(comparison.Autolinks, presentation.AutolinkLine, writeLine)
	presentation.RenderChanges(comparison.BranchProtectionRules, presentation.ClassicProtectionLine, writeLine)
	presentation.RenderChanges(comparison.Environments, presentation.EnvironmentLine, writeLine)
	presentation.RenderActionsPolicyChanges(&comparison, writeLine)
	presentation.RenderCodeQLConfiguration(comparison.CodeQLConfiguration, writeLine)
	presentation.RenderChanges(comparison.Security, presentation.SecurityLine, writeLine)
	presentation.RenderChanges(comparison.Pages, presentation.PagesLine, writeLine)
	presentation.RenderChanges(comparison.Webhooks, presentation.WebhookLine, writeLine)
	presentation.RenderChanges(comparison.DeployKeys, presentation.DeployKeyLine, writeLine)
	presentation.RenderChanges(comparison.CollaboratorDeletions, presentation.CollaboratorDeletionLine, writeLine)
	presentation.RenderChanges(comparison.Collaborators, presentation.CollaboratorLine, writeLine)
	return []byte(output.String())
}
