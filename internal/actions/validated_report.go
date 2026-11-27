package actions

import (
	"fmt"
	"reflect"

	"github.com/daiksud/gh-settings/internal/settings"
)

// validatedPlanReport retains the result of the JSON and report-state boundary.
type validatedPlanReport struct {
	outcome, diagnostic, resource string
	comparison                    settings.Comparison
	rulesets                      []settings.RulesetDisplayChange
}

func validatePlanReport(report PlanReport, target string) (validatedPlanReport, error) {
	if !validFailureResource(report.Outcome, report.Resource) {
		return validatedPlanReport{}, fmt.Errorf("invalid plan report resource")
	}
	execution, executionManaged, err := validateActionsExecutionReport(report.ActionsExecution)
	if err != nil {
		return validatedPlanReport{}, err
	}
	maxManaged := settings.PullRequestLimitsFieldCount + 4 + settings.RepositoryFieldCount() + 16 + settings.ActionsPolicyFieldCount + settings.SecurityFieldCount + settings.CodeQLConfigurationFieldCount + settings.PagesFieldCount // Collections count once; scalar policy controls retain their closed bounds.
	scalarChanges := len(report.OIDCSubject) + len(report.CopilotReviewEffort) + len(report.PullRequestLimits) + len(report.CopilotMCP) + len(report.CopilotPublicSearch) + len(report.CopilotInstructions) + len(report.CodeQLConfiguration) + len(report.Changes) + 2*len(report.CommitMessages) + len(report.Actions) + executionManaged + len(report.Security) + minimumPagesReportFields(report.Pages)
	if report.Changes == nil || report.Managed < scalarChanges || report.Managed > maxManaged {
		return validatedPlanReport{}, fmt.Errorf("invalid plan report comparison")
	}
	if len(report.Labels) > 0 && report.Managed < scalarChanges+1 {
		return validatedPlanReport{}, fmt.Errorf("invalid label report managed count")
	}
	runners, err := validateRunnersReport(report.Runners)
	if err != nil {
		return validatedPlanReport{}, err
	}
	runnerDeletions, err := validateRunnerDeletionsReport(report.RunnerDeletions, runners)
	if err != nil {
		return validatedPlanReport{}, err
	}
	if (len(runners) > 0 || len(runnerDeletions) > 0) && report.Managed < scalarChanges+1 {
		return validatedPlanReport{}, fmt.Errorf("invalid runner managed count")
	}
	deployKeys, err := validateDeployKeyReport(report.DeployKeys)
	if err != nil {
		return validatedPlanReport{}, err
	}
	collaborators, err := validateCollaboratorReport(report.Collaborators)
	if err != nil {
		return validatedPlanReport{}, err
	}
	collaboratorDeletions, err := validateCollaboratorDeletions(report.CollaboratorDeletions, collaborators)
	if err != nil {
		return validatedPlanReport{}, err
	}
	labels, err := validateLabelReport(report.Labels)
	if err != nil {
		return validatedPlanReport{}, err
	}
	actionsPolicies, err := validateActionsEventPolicyReport(report.ActionsPolicies)
	if err != nil {
		return validatedPlanReport{}, err
	}
	rulesets, err := validateRulesetReport(report.Rulesets)
	if err != nil {
		return validatedPlanReport{}, err
	}
	environments, err := validateEnvironmentReport(report.Environments)
	if err != nil {
		return validatedPlanReport{}, err
	}
	variables, err := validateVariableReport(report.Variables)
	if err != nil {
		return validatedPlanReport{}, err
	}
	environmentVariables, err := validateEnvironmentVariableReport(report.EnvironmentVariables)
	if err != nil {
		return validatedPlanReport{}, err
	}
	agentsVariables, err := validateVariableReport(report.AgentsVariables)
	if err != nil {
		return validatedPlanReport{}, err
	}
	copilotInstructions, err := validateCopilotInstructionsReport(report.CopilotInstructions)
	if err != nil {
		return validatedPlanReport{}, err
	}
	pullRequestExemptions, err := validatePullRequestExemptionsReport(report.PullRequestExemptions)
	if err != nil {
		return validatedPlanReport{}, err
	}
	pullRequestLimits, err := validatePullRequestLimitsReport(report.PullRequestLimits)
	if err != nil {
		return validatedPlanReport{}, err
	}
	oidcSubject, err := validateOIDCSubjectReport(report.OIDCSubject)
	if err != nil {
		return validatedPlanReport{}, err
	}
	copilotMCP, err := validateCopilotToggleReport(report.CopilotMCP, settings.CopilotMCPPath)
	if err != nil {
		return validatedPlanReport{}, err
	}
	copilotPublicSearch, err := validateCopilotToggleReport(report.CopilotPublicSearch, settings.CopilotPublicSearchPath)
	if err != nil {
		return validatedPlanReport{}, err
	}
	copilotReviewEffort, err := validateCopilotReviewEffortReport(report.CopilotReviewEffort)
	if err != nil {
		return validatedPlanReport{}, err
	}
	if err := settings.ValidateClassicChanges(report.BranchProtectionRules); err != nil {
		return validatedPlanReport{}, err
	}
	if err := settings.ValidateAutolinkChanges(report.Autolinks); err != nil {
		return validatedPlanReport{}, err
	}
	if err := settings.ValidateWebhookChanges(report.Webhooks); err != nil {
		return validatedPlanReport{}, err
	}
	collections := 0
	if len(report.Webhooks) > 0 {
		collections++
	}
	if len(runners) > 0 || len(runnerDeletions) > 0 {
		collections++
	}
	if len(deployKeys) > 0 {
		collections++
	}
	if len(report.Autolinks) > 0 {
		collections++
	}
	if len(report.BranchProtectionRules) > 0 {
		collections++
	}
	if len(collaborators) > 0 || len(collaboratorDeletions) > 0 {
		collections++
	}
	if len(pullRequestExemptions) > 0 {
		collections++
	}
	if len(actionsPolicies) > 0 {
		collections++
	}
	if len(variables) > 0 {
		collections++
	}
	if len(environmentVariables) > 0 {
		collections++
	}
	if len(agentsVariables) > 0 {
		collections++
	}
	if labels.Len() > 0 {
		collections++
	}
	if len(rulesets) > 0 {
		collections++
	}
	if len(environments) > 0 {
		collections++
	}
	if report.Managed < scalarChanges+collections {
		return validatedPlanReport{}, fmt.Errorf("invalid collection managed count")
	}
	hasChanges := len(report.Webhooks) > 0 || len(runners) > 0 || len(runnerDeletions) > 0 || len(deployKeys) > 0 || len(report.Autolinks) > 0 || len(oidcSubject) > 0 || len(collaboratorDeletions) > 0 || len(report.BranchProtectionRules) > 0 || len(pullRequestExemptions) > 0 || len(environmentVariables) > 0 || len(actionsPolicies) > 0 || len(copilotReviewEffort) > 0 || len(pullRequestLimits) > 0 || len(copilotMCP) > 0 || len(copilotPublicSearch) > 0 || len(copilotInstructions) > 0 || len(variables) > 0 || len(agentsVariables) > 0 || len(report.CodeQLConfiguration) > 0 || len(report.Changes) > 0 || len(report.CommitMessages) > 0 || labels.Len() > 0 || len(rulesets) > 0 || len(environments) > 0 || len(report.Actions) > 0 || len(execution) > 0 || len(report.Security) > 0 || len(report.Pages) > 0
	for _, person := range collaborators {
		hasChanges = hasChanges || person.Changes()
	}
	valid := false
	switch report.Outcome {
	case "changed":
		valid = report.Managed > 0 && hasChanges && report.Diagnostic == "" && report.Target == target
	case "no-difference":
		valid = report.Managed > 0 && !hasChanges && report.Diagnostic == "" && report.Target == target
	case "unmanaged":
		valid = report.Managed == 0 && report.Target == "" && report.Diagnostic == ""
	case "invalid-config":
		valid = report.Managed == 0 && report.Target == "" && report.Diagnostic == invalidConfigDiagnostic
	case "target-failure":
		valid = report.Managed == 0 && report.Target == "" && report.Diagnostic == targetFailureDiagnostic
	case "read-failure":
		valid = report.Managed == 0 && report.Target == target && report.Diagnostic == readFailureDiagnostic
	}
	if !valid {
		return validatedPlanReport{}, fmt.Errorf("invalid plan report state")
	}
	commitMessages, err := validateCommitMessageReport(report.CommitMessages, report.Changes)
	if err != nil {
		return validatedPlanReport{}, err
	}
	if err := validateCommitMessageContexts(report.CommitMessageContexts, report.Changes); err != nil {
		return validatedPlanReport{}, err
	}
	comparison := settings.Comparison{Runners: runners, RunnerDeletions: runnerDeletions, Webhooks: report.Webhooks, DeployKeys: deployKeys, Autolinks: report.Autolinks, OIDCSubject: oidcSubject, CollaboratorDeletions: collaboratorDeletions, Collaborators: collaborators, BranchProtectionRules: report.BranchProtectionRules, ActionsPolicies: actionsPolicies, PullRequestLimits: pullRequestLimits, PullRequestExemptions: pullRequestExemptions, CopilotReviewEffort: copilotReviewEffort, CopilotMCP: copilotMCP, CopilotPublicSearch: copilotPublicSearch, CopilotInstructions: copilotInstructions, Variables: variables, AgentsVariables: agentsVariables, EnvironmentVariables: environmentVariables, Managed: report.Managed, CommitMessages: commitMessages, CommitMessageContexts: report.CommitMessageContexts, Labels: labels, Environments: environments, ActionsExecution: execution}
	for _, raw := range report.Actions {
		if raw.SecurityStatusNormalization != nil {
			return validatedPlanReport{}, fmt.Errorf("security status normalization cannot describe an Actions comparison")
		}
		var change settings.ActionsPolicyChange
		var err error
		if raw.Normalization != nil {
			change, err = settings.NewActionsPolicyChangeWithNormalization(raw.Field, raw.Before, raw.After, *raw.Normalization)
		} else {
			change, err = settings.NewActionsPolicyChange(raw.Field, raw.Before, raw.After)
		}
		if err != nil {
			return validatedPlanReport{}, err
		}
		comparison.Actions = append(comparison.Actions, change)
	}
	if err := validateActionsExecutionContext(report.ActionsExecutionContext, report.Actions, report.ActionsExecution); err != nil {
		return validatedPlanReport{}, err
	}
	comparison.ActionsExecutionContext = report.ActionsExecutionContext
	if err := settings.ValidateActionsPolicyChanges(comparison.Actions, comparison.ActionsExecution...); err != nil {
		return validatedPlanReport{}, err
	}
	for _, raw := range report.Security {
		if raw.Normalization != nil {
			return validatedPlanReport{}, fmt.Errorf("workflow normalization cannot describe a security comparison")
		}
		var change settings.SecurityChange
		var err error
		if raw.SecurityStatusNormalization != nil {
			change, err = settings.NewSecurityChangeWithNormalization(raw.Field, raw.Before, raw.After, *raw.SecurityStatusNormalization)
		} else {
			change, err = settings.NewSecurityChange(raw.Field, raw.Before, raw.After)
		}
		if err != nil {
			return validatedPlanReport{}, err
		}
		comparison.Security = append(comparison.Security, change)
	}
	if err := settings.ValidateSecurityChanges(comparison.Security); err != nil {
		return validatedPlanReport{}, err
	}
	codeql, err := validateCodeQLConfigurationReport(report.CodeQLConfiguration)
	if err != nil {
		return validatedPlanReport{}, err
	}
	comparison.CodeQLConfiguration = codeql
	pages, err := validatePagesReport(report.Pages)
	if err != nil {
		return validatedPlanReport{}, err
	}
	comparison.Pages = pages
	seen := make(map[settings.FieldID]bool, len(report.Changes))
	for _, raw := range report.Changes {
		change, err := parseReportChange(raw)
		if err != nil {
			return validatedPlanReport{}, err
		}
		if seen[change.Field()] {
			return validatedPlanReport{}, fmt.Errorf("invalid plan report change")
		}
		seen[change.Field()] = true
		comparison.Changes = append(comparison.Changes, change)
	}
	return validatedPlanReport{outcome: report.Outcome, diagnostic: report.Diagnostic, resource: report.Resource, comparison: comparison, rulesets: rulesets}, nil
}

func parseReportChange(raw PlanChange) (settings.Change, error) {
	if raw.SecurityStatusNormalization != nil {
		return settings.Change{}, fmt.Errorf("security status normalization cannot describe a repository comparison")
	}
	field, err := settings.ParseFieldID(raw.Field)
	if err != nil {
		return settings.Change{}, fmt.Errorf("invalid plan report change")
	}
	if raw.Normalization != nil {
		before, beforeOK := raw.Before.(string)
		after, afterOK := raw.After.(string)
		if !beforeOK || !afterOK {
			return settings.Change{}, fmt.Errorf("creation policy normalization requires string values")
		}
		return settings.NewCreationPolicyChangeWithNormalization(field, before, after, *raw.Normalization)
	}
	var change settings.Change
	if field == settings.TopicsField() {
		before, beforeOK := summaryTopics(raw.Before)
		after, afterOK := summaryTopics(raw.After)
		if !beforeOK || !afterOK || reflect.DeepEqual(before, after) {
			return settings.Change{}, fmt.Errorf("invalid plan report change")
		}
		change, err = settings.NewTopicsChange(before, after)
	} else {
		switch before := raw.Before.(type) {
		case bool:
			after, ok := raw.After.(bool)
			if !ok || before == after {
				return settings.Change{}, fmt.Errorf("invalid plan report change")
			}
			change, err = settings.NewBoolChange(field, before, after)
		case string:
			after, ok := raw.After.(string)
			if !ok || before == after {
				return settings.Change{}, fmt.Errorf("invalid plan report change")
			}
			change, err = settings.NewStringChange(field, before, after)
		default:
			return settings.Change{}, fmt.Errorf("invalid plan report change")
		}
	}
	if err != nil {
		return settings.Change{}, fmt.Errorf("invalid plan report change")
	}
	return change, nil
}

func summaryTopics(value any) ([]string, bool) {
	var names []string
	switch value := value.(type) {
	case []string:
		names = value
	case []any:
		names = make([]string, 0, len(value))
		for _, item := range value {
			name, ok := item.(string)
			if !ok {
				return nil, false
			}
			names = append(names, name)
		}
	default:
		return nil, false
	}
	if names == nil || settings.ValidateTopics(names) != nil {
		return nil, false
	}
	sorted := settings.SortedTopics(names)
	return sorted, reflect.DeepEqual(sorted, names)
}
