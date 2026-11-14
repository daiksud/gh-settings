package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

const (
	invalidConfigDiagnostic = "configuration validation failed"
	targetFailureDiagnostic = "repository target resolution failed"
	readFailureDiagnostic   = "repository read failed"
)

// PlanReport is same-version, ephemeral presentation data, never apply input.
type PlanReport struct {
	Webhooks                []settings.WebhookChange          `json:"webhooks,omitempty"`
	Runners                 RunnerPlanChanges                 `json:"runners,omitempty"`
	DeployKeys              []DeployKeyPlanEntry              `json:"deploy_keys,omitempty"`
	Autolinks               []settings.AutolinkChange         `json:"autolinks,omitempty"`
	CollaboratorDeletions   []CollaboratorDeletionEntry       `json:"collaborator_deletions,omitempty"`
	BranchProtectionRules   []settings.ClassicChange          `json:"branch_protection_rules,omitempty"`
	OIDCSubject             []PlanChange                      `json:"oidc_subject,omitempty"`
	Collaborators           []CollaboratorPlanEntry           `json:"collaborators,omitempty"`
	PullRequestExemptions   []PullRequestExemptionsPlanChange `json:"pull_request_exemptions,omitempty"`
	EnvironmentVariables    []EnvironmentVariablePlanChange   `json:"environment_variables,omitempty"`
	ActionsPolicies         ActionsEventPolicyPlanChanges     `json:"actions_policies,omitempty"`
	CopilotReviewEffort     []PlanChange                      `json:"copilot_review_effort,omitempty"`
	CopilotInstructions     []PlanChange                      `json:"copilot_custom_instructions,omitempty"`
	CopilotMCP              []PlanChange                      `json:"copilot_mcp,omitempty"`
	PullRequestLimits       []PlanChange                      `json:"pull_request_limits,omitempty"`
	Variables               []VariablePlanChange              `json:"variables,omitempty"`
	AgentsVariables         []VariablePlanChange              `json:"agents_variables,omitempty"`
	CodeQLConfiguration     []PlanChange                      `json:"codeql_configuration,omitempty"`
	Outcome                 string                            `json:"outcome"`
	Target                  string                            `json:"target,omitempty"`
	Managed                 int                               `json:"managed"`
	Changes                 []PlanChange                      `json:"changes"`
	CommitMessageContexts   []settings.CommitMessageContext   `json:"commit_message_contexts,omitempty"`
	CommitMessages          []CommitMessagePlanChange         `json:"commit_messages,omitempty"`
	Resource                string                            `json:"resource,omitempty"`
	Diagnostic              string                            `json:"diagnostic,omitempty"`
	Labels                  []LabelPlanChange                 `json:"labels,omitempty"`
	Actions                 []PlanChange                      `json:"actions,omitempty"`
	ActionsExecutionContext *settings.ActionsExecutionContext `json:"actions_execution_context,omitempty"`
	ActionsExecution        []ActionsExecutionPlanChange      `json:"actions_execution,omitempty"`
	Security                []PlanChange                      `json:"security,omitempty"`
	Rulesets                []RulesetPlanChange               `json:"rulesets,omitempty"`
	Environments            []EnvironmentPlanChange           `json:"environments,omitempty"`
	Pages                   []PagesPlanChange                 `json:"pages,omitempty"`
}

// PlanChange retains the native string, boolean, integer or string-list comparison values.
type PlanChange struct {
	Field                       string                                `json:"field"`
	Before                      any                                   `json:"before"`
	After                       any                                   `json:"after"`
	Normalization               *settings.ObservationNormalization    `json:"normalization,omitempty"`
	SecurityStatusNormalization *settings.SecurityStatusNormalization `json:"security_status_normalization,omitempty"`
}

func writePlanReport(path string, plan func() (app.PlanResult, error)) error {
	result, planErr := plan()
	if planErr == nil {
		if err := settings.ValidateWebhookChanges(result.Comparison.Webhooks); err != nil {
			planErr = &app.PlanError{Stage: "read-failure", Path: settings.WebhooksPath, Target: result.Target, Err: err}
			result.Comparison = settings.Comparison{}
		}
	}
	if planErr == nil {
		if err := settings.ValidateAutolinkChanges(result.Comparison.Autolinks); err != nil {
			planErr = &app.PlanError{Stage: "read-failure", Path: settings.AutolinksPath, Target: result.Target, Err: err}
			result.Comparison = settings.Comparison{}
		}
	}
	if planErr == nil {
		if err := settings.ValidateClassicChanges(result.Comparison.BranchProtectionRules); err != nil {
			planErr = &app.PlanError{Stage: "read-failure", Path: settings.ClassicProtectionPath, Target: result.Target, Err: err}
			result.Comparison = settings.Comparison{}
		}
	}
	if planErr == nil {
		if err := presentation.ValidateRulesetChanges(result.Comparison.Rulesets); err != nil {
			planErr = &app.PlanError{Stage: "read-failure", Path: "repository.settings.rulesets", Target: result.Target, Err: err}
			result.Comparison = settings.Comparison{}
		}
	}

	if planErr == nil {
		if err := presentation.ValidateActionsEventPolicyChanges(result.Comparison.ActionsPolicies); err != nil {
			planErr = &app.PlanError{Stage: "read-failure", Path: "repository.settings.actions.policies", Target: result.Target, Err: err}
			result.Comparison = settings.Comparison{}
		}
	}
	report := PlanReport{Outcome: "unmanaged", Managed: result.Comparison.Managed, Changes: make([]PlanChange, 0, len(result.Comparison.Changes))}
	if planErr != nil {
		var failure *app.PlanError
		if !errors.As(planErr, &failure) {
			return planErr
		}
		report.Resource = reportFailureResource(failure)
		switch failure.Stage {
		case "invalid-config":
			report.Outcome = "invalid-config"
			report.Diagnostic = invalidConfigDiagnostic
		case "read-failure":
			report.Outcome = "read-failure"
			report.Target = failure.Target.Owner + "/" + failure.Target.Name
			report.Diagnostic = readFailureDiagnostic
		case "target-failure":
			report.Outcome = "target-failure"
			report.Diagnostic = targetFailureDiagnostic
		default:
			return planErr
		}
	}
	if result.Comparison.Managed > 0 {
		report.Target = result.Target.Owner + "/" + result.Target.Name
		report.Outcome = "changed"
		if !result.Comparison.HasChanges() {
			report.Outcome = "no-difference"
		}
	}
	for _, change := range result.Comparison.Changes {
		entry := PlanChange{Field: change.Field().Path()}
		if normalization := change.Normalization(); normalization != (settings.ObservationNormalization{}) {
			entry.Normalization = &normalization
		}
		change.Visit(
			func(before, after bool) { entry.Before, entry.After = before, after },
			func(before, after string) { entry.Before, entry.After = before, after },
			func(before, after []string) { entry.Before, entry.After = before, after },
		)
		report.Changes = append(report.Changes, entry)
	}
	report.Webhooks = result.Comparison.Webhooks
	report.Autolinks = result.Comparison.Autolinks
	report.BranchProtectionRules = result.Comparison.BranchProtectionRules
	report.CommitMessageContexts = result.Comparison.CommitMessageContexts
	report.ActionsExecutionContext = result.Comparison.ActionsExecutionContext
	for _, change := range result.Comparison.CommitMessages {
		_, after := change.Values()
		report.CommitMessages = append(report.CommitMessages, CommitMessagePlanChange{Menu: change.Menu(), After: after, Normalization: change.Normalization()})
	}
	for _, change := range result.Comparison.DeployKeys {
		before, after := change.Values()
		entry := DeployKeyPlanEntry{Operation: change.Operation()}
		if before != nil {
			entry.Before = before
		}
		if after != nil {
			entry.After = after
		}
		report.DeployKeys = append(report.DeployKeys, entry)
	}
	for _, d := range result.Comparison.CollaboratorDeletions {
		report.CollaboratorDeletions = append(report.CollaboratorDeletions, CollaboratorDeletionEntry{Operation: d.Operation(), Username: d.Person.Username, ID: d.Person.ID, InvitationID: d.InvitationID})
	}
	for _, person := range result.Comparison.Collaborators {
		bind := person.Bind
		report.Collaborators = append(report.Collaborators, CollaboratorPlanEntry{Username: person.DeclaredUsername, ID: person.Person.ID, State: person.Status, InvitationID: person.InvitationID, BindIdentity: &bind})
	}
	for _, change := range result.Comparison.Labels.Values() {
		report.Labels = append(report.Labels, labelPlanChange(change))
	}
	for _, change := range result.Comparison.Variables {
		before, after := change.Values()
		record := VariablePlanChange{Name: change.Name(), Before: before, After: &after}
		if change.Operation() == "delete" {
			record.Operation = "delete"
			record.After = nil
		}
		report.Variables = append(report.Variables, record)
	}
	for _, change := range result.Comparison.EnvironmentVariables {
		before, after := change.Change.Values()
		record := EnvironmentVariablePlanChange{Environment: change.Environment, Name: change.Change.Name(), Before: before, After: &after}
		if change.Change.Operation() == "delete" {
			record.Operation = "delete"
			record.After = nil
		}
		report.EnvironmentVariables = append(report.EnvironmentVariables, record)
	}
	for _, change := range result.Comparison.AgentsVariables {
		before, after := change.Values()
		record := VariablePlanChange{Name: change.Name(), Before: before, After: &after}
		if change.Operation() == "delete" {
			record.Operation = "delete"
			record.After = nil
		}
		report.AgentsVariables = append(report.AgentsVariables, record)
	}
	for _, change := range result.Comparison.CopilotInstructions {
		before, after := change.Values()
		report.CopilotInstructions = append(report.CopilotInstructions, PlanChange{Field: change.Field(), Before: before, After: after})
	}
	for _, change := range result.Comparison.Runners {
		before, after := change.Values()
		report.Runners = append(report.Runners, RunnerPlanChange{Field: change.Field(), Before: before, After: after})
	}
	for _, change := range result.Comparison.OIDCSubject {
		before, after := change.Values()
		report.OIDCSubject = append(report.OIDCSubject, PlanChange{Field: change.Field(), Before: before, After: after})
	}
	for _, change := range result.Comparison.CopilotMCP {
		before, after := change.Values()
		report.CopilotMCP = append(report.CopilotMCP, PlanChange{Field: change.Field(), Before: before, After: after})
	}
	for _, change := range result.Comparison.PullRequestExemptions {
		before, after := change.Values()
		report.PullRequestExemptions = append(report.PullRequestExemptions, PullRequestExemptionsPlanChange{Before: before, After: after})
	}
	for _, change := range result.Comparison.PullRequestLimits {
		before, after := change.Values()
		report.PullRequestLimits = append(report.PullRequestLimits, PlanChange{Field: change.Field(), Before: before, After: after})
	}
	for _, change := range result.Comparison.CopilotReviewEffort {
		before, after := change.Values()
		report.CopilotReviewEffort = append(report.CopilotReviewEffort, PlanChange{Field: change.Field(), Before: before, After: after})
	}
	for _, change := range result.Comparison.ActionsPolicies {
		entry := ActionsEventPolicyPlanChange{Operation: change.Operation()}
		if change.Operation() != "delete" {
			entry.After = actionsEventPolicyReport(change.After())
		}
		if before := change.Before(); before != nil {
			entry.Before = actionsEventPolicyReport(*before)
		}
		report.ActionsPolicies = append(report.ActionsPolicies, entry)
	}
	for _, change := range result.Comparison.Rulesets {
		display := change.Display()
		entry := RulesetPlanChange{Operation: display.Operation(), Before: display.Before(), After: display.After()}
		if record := display.Normalizations().Enforcement; record != (settings.ObservationNormalization{}) {
			entry.EnforcementNormalization = &record
		}
		if record := display.Normalizations().QualitySeverity; record != (settings.ObservationNormalization{}) {
			entry.QualitySeverityNormalization = &record
		}
		if record := display.Normalizations().RequiredApprovals; record != (settings.RulesetRequiredApprovalsNormalization{}) {
			entry.RequiredApprovalsNormalization = &record
		}
		if record := display.Normalizations().MergeMethods; !record.IsZero() {
			entry.MergeMethodsNormalization = &RulesetMergeMethodsNormalization{Path: record.Path, RawValue: record.RawValue, Fallback: record.Fallback}
		}
		if records := display.Normalizations().TeamApprovals; len(records) > 0 {
			values := make([]RulesetTeamApprovalsNormalization, 0, len(records))
			for _, record := range records {
				values = append(values, RulesetTeamApprovalsNormalization{ReviewerID: record.ReviewerID, Path: record.Path, RawValue: record.RawValue, Fallback: record.Fallback})
			}
			entry.TeamApprovalsNormalizations = &values
		}
		if records := display.Normalizations().ScanningAlerts; len(records) > 0 {
			values := make([]RulesetScanningAlertsNormalization, 0, len(records))
			for _, record := range records {
				values = append(values, RulesetScanningAlertsNormalization{Tool: record.Tool, Path: record.Path, RawValue: record.RawValue, Fallback: record.Fallback})
			}
			entry.ScanningAlertsNormalizations = &values
		}
		if records := display.Normalizations().ScanningSecurityAlerts; len(records) > 0 {
			values := make([]RulesetScanningSecurityAlertsNormalization, 0, len(records))
			for _, record := range records {
				values = append(values, RulesetScanningSecurityAlertsNormalization{Tool: record.Tool, Path: record.Path, RawValue: record.RawValue, Fallback: record.Fallback})
			}
			entry.ScanningSecurityAlertsNormalizations = &values
		}
		if records := display.Normalizations().BypassModes; len(records) > 0 {
			values := make([]RulesetBypassModeNormalization, 0, len(records))
			for _, record := range records {
				values = append(values, RulesetBypassModeNormalization{ActorType: record.ActorType, ActorID: record.ActorID, Normalization: record.Normalization})
			}
			entry.BypassModeNormalizations = &values
		}
		report.Rulesets = append(report.Rulesets, entry)
	}
	for _, change := range result.Comparison.Environments {
		entry := EnvironmentPlanChange{Operation: change.Operation(), Before: change.Before(), After: change.After()}
		if record := change.DeploymentModeNormalization(); record != (settings.EnvironmentDeploymentNormalization{}) {
			entry.DeploymentModeNormalization = &record
		}
		if states := change.PatternObservations(); states != nil {
			patterns := make([]EnvironmentPatternObservation, 0, len(states))
			for _, state := range states {
				value := EnvironmentPatternObservation{ID: state.ID, Pattern: state.Pattern}
				if state.RefTypeNormalization != (settings.ObservationNormalization{}) {
					record := state.RefTypeNormalization
					value.RefTypeNormalization = &record
				}
				patterns = append(patterns, value)
			}
			entry.PatternObservations = &patterns
		}
		report.Environments = append(report.Environments, entry)
	}
	for _, change := range result.Comparison.Actions {
		before, after := change.Values()
		entry := PlanChange{Field: change.Field(), Before: before, After: after}
		if normalization := change.Normalization(); normalization != (settings.ObservationNormalization{}) {
			entry.Normalization = &normalization
		}
		report.Actions = append(report.Actions, entry)
	}
	for _, change := range result.Comparison.ActionsExecution {
		report.ActionsExecution = append(report.ActionsExecution, ActionsExecutionPlanChange{Before: change.Before(), After: change.After(), Normalization: change.Normalization()})
	}
	for _, change := range result.Comparison.Security {
		before, after := change.Values()
		entry := PlanChange{Field: change.Field(), Before: before, After: after}
		if normalization := change.Normalization(); normalization != (settings.SecurityStatusNormalization{}) {
			entry.SecurityStatusNormalization = &normalization
		}
		report.Security = append(report.Security, entry)
	}
	for _, c := range result.Comparison.CodeQLConfiguration {
		b, a := c.Values()
		report.CodeQLConfiguration = append(report.CodeQLConfiguration, PlanChange{Field: c.Field(), Before: b, After: a})
	}
	for _, c := range result.Comparison.Pages {
		b, a := c.Values()
		entry := PagesPlanChange{Field: c.Field(), Before: b, After: a, BeforeInactive: c.BeforeInactive(), BeforeAbsent: c.BeforeAbsent()}
		if record := c.Normalization(); record != (settings.ObservationNormalization{}) {
			entry.Normalization = &record
		}
		report.Pages = append(report.Pages, entry)
	}
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("encode plan report: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		return fmt.Errorf("write plan report: %w", err)
	}
	return planErr
}
