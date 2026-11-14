package app

import (
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// TargetResolver supplies the repository belonging to the selected checkout.
type TargetResolver func() (repository.Repository, error)

// PlanResult identifies the selected repository and its declared-field comparison.
// A local no-op has no target and no managed fields.
type PlanResult struct {
	Target     repository.Repository
	Comparison settings.Comparison
}

// PlanError identifies a failed planning stage without changing its diagnostic.
type PlanError struct {
	Stage  string
	Path   string
	Target repository.Repository
	Err    error
}

func (e *PlanError) Error() string {
	if e.Path != "" {
		return e.Path + ": " + e.Err.Error()
	}
	return e.Err.Error()
}
func (e *PlanError) Unwrap() error { return e.Err }

// Plan validates locally, then reads and compares only when fields are declared.
func Plan(load DeclarationLoader, resolve TargetResolver, read Reader, options ...settings.PruneScopes) (PlanResult, error) {
	prune, optionErr := operationPrune(options)
	if optionErr != nil {
		return PlanResult{}, optionErr
	}
	desired, err := Validate(load)
	if err != nil {
		return PlanResult{}, &PlanError{Stage: "invalid-config", Err: err}
	}
	if desired.Empty() && !prune.Manages(desired) {
		return PlanResult{}, nil
	}
	repo, err := resolve()
	if err != nil {
		return PlanResult{}, &PlanError{Stage: "target-failure", Err: err}
	}
	var comparison settings.Comparison
	createdEnvironments := map[string]bool{}
	if desired.Repository.Archived != nil {
		state, err := read.Archive(repo)
		if err == nil {
			err = state.Validate(repo)
		}
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.general.danger_zone.archive_this_repository", Target: repo, Err: err}
		}
		comparison.Managed++
		if state.Archived != *desired.Repository.Archived {
			field, _ := settings.ParseFieldID("repository.archived")
			change, _ := settings.NewBoolChange(field, state.Archived, *desired.Repository.Archived)
			comparison.Changes = append(comparison.Changes, change)
		}
	}
	other := desired.Repository
	other.Archived = nil
	if fields := settings.DeclaredRepositoryFields(other); len(fields) > 0 {
		observed, err := read.ObserveRepository(repo, fields)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository", Target: repo, Err: err}
		}
		otherComparison, err := settings.CompareRepositoryObservation(observed, other)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository", Target: repo, Err: err}
		}
		comparison.Managed += otherComparison.Managed
		comparison.CommitMessages = append(comparison.CommitMessages, otherComparison.CommitMessages...)
		comparison.CommitMessageContexts = append(comparison.CommitMessageContexts, otherComparison.CommitMessageContexts...)
		// An explicit unarchive precedes normal writes; archive is always last.
		if desired.Repository.Archived != nil && !*desired.Repository.Archived {
			comparison.Changes = append(comparison.Changes, otherComparison.Changes...)
		} else {
			comparison.Changes = append(otherComparison.Changes, comparison.Changes...)
		}
	}
	if desired.Topics.Names != nil {
		current, err := read.Topics(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.about.topics", Target: repo, Err: err}
		}
		topics := settings.CompareTopics(current, desired.Topics)
		comparison.Managed += topics.Managed
		comparison.Changes = append(comparison.Changes, topics.Changes...)
	}
	if desired.Collaborators != nil && (desired.Collaborators.Len() > 0 || prune.Has("collaborators")) {
		var selected []settings.CollaboratorSelection
		var err error
		if prune.Has("collaborators") {
			var prepared settings.CollaboratorPruneObservation
			if read.CollaboratorPrune == nil {
				err = fmt.Errorf("collaborator prune reader unavailable")
			} else {
				prepared, err = read.CollaboratorPrune(repo, *desired.Collaborators)
			}
			if err == nil {
				selected = prepared.Selections()
				comparison.CollaboratorDeletions = prepared.Deletions()
			}
		} else {
			selected, err = prepareCollaborators(repo, *desired.Collaborators, read.Collaborators, read.ResolveCollaborator)
		}
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.collaborators", Target: repo, Err: err}
		}
		comparison.Managed++
		comparison.Collaborators = selected
	}
	if desired.Labels != nil && (desired.Labels.Len() > 0 || prune.Has("labels")) {
		current, err := read.Labels(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.labels", Target: repo, Err: err}
		}
		labels, err := settings.CompareLabelsWithPrune(current, desired.Labels, prune.Has("labels"))
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.labels", Target: repo, Err: err}
		}
		comparison.Managed += labels.Managed
		comparison.Labels = labels.Labels
	}
	if desired.Variables != nil && (desired.Variables.Len() > 0 || prune.Has("actions-variables")) {
		var changes []settings.VariableChange
		var err error
		if prune.Has("actions-variables") {
			inventory, readErr := read.Variables(repo)
			if readErr != nil {
				return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.variables", Target: repo, Err: readErr}
			}
			changes, err = planVariablePrune(repo, *desired.Variables, read.Variable, inventory)
		} else {
			changes, err = planVariables(repo, *desired.Variables, read.Variable)
		}
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.variables", Target: repo, Err: err}
		}
		comparison.Managed++
		comparison.Variables = changes
	}
	if desired.AgentsVariables != nil && (desired.AgentsVariables.Len() > 0 || prune.Has("agents-variables")) {
		var changes []settings.VariableChange
		var err error
		if prune.Has("agents-variables") {
			inventory, readErr := read.AgentVariables(repo)
			if readErr != nil {
				return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.agents.variables", Target: repo, Err: readErr}
			}
			changes, err = planVariablePrune(repo, *desired.AgentsVariables, read.AgentVariable, inventory)
		} else {
			changes, err = planVariables(repo, *desired.AgentsVariables, read.AgentVariable)
		}
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.agents.variables", Target: repo, Err: err}
		}
		comparison.Managed++
		comparison.AgentsVariables = changes
	}
	if desired.Webhooks != nil && (len(*desired.Webhooks) > 0 || prune.Has("webhooks")) {
		var observed settings.WebhookObservation
		var prepared settings.WebhookPruneObservation
		var err error
		if prune.Has("webhooks") {
			if read.WebhookPrune == nil {
				err = fmt.Errorf("webhook prune reader unavailable")
			} else {
				prepared, err = read.WebhookPrune(repo)
			}
			if err == nil {
				prepared, err = settings.PrepareWebhookPrune(prepared, *desired.Webhooks)
			}
			if err == nil {
				observed = prepared.Inventory()
			}
		} else if read.Webhooks == nil {
			err = fmt.Errorf("webhook reader unavailable")
		} else {
			observed, err = read.Webhooks(repo, webhookIDs(*desired.Webhooks))
		}
		var changes []settings.WebhookChange
		if err == nil {
			_, changes, err = settings.PrepareWebhooks(observed, *desired.Webhooks)
		}
		if err == nil && prune.Has("webhooks") {
			var candidates []int64
			candidates, err = prepared.Candidates()
			if err == nil {
				for _, id := range candidates {
					state, found := prepared.Find(id)
					if !found {
						err = fmt.Errorf("webhook candidate unavailable")
						break
					}
					url, mime, active, events := state.State.Config.PayloadURL, state.State.Config.ContentType, state.State.Active, append([]string{}, state.State.Events...)
					before := settings.Webhook{ID: id, PayloadURL: &url, ContentType: &mime, Active: &active, Events: &events}
					changes = append(changes, settings.WebhookChange{Before: &before})
				}
			}
		}
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.WebhooksPath, Target: repo, Err: err}
		}
		comparison.Managed++
		comparison.Webhooks = changes
	}
	if desired.DeployKeys != nil && (desired.DeployKeys.Len() > 0 || prune.Has("deploy-keys")) {
		observed, err := read.DeployKeys(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.DeployKeysPath, Target: repo, Err: err}
		}
		_, changes, err := settings.PrepareDeployKeys(observed, desired.DeployKeys, prune.Has("deploy-keys"))
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.DeployKeysPath, Target: repo, Err: err}
		}
		comparison.Managed++
		comparison.DeployKeys = changes
	}
	if desired.CopilotInstructions != nil {
		observed, err := read.CopilotInstructions(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.CopilotInstructionsPath, Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareCopilotInstructions(observed, desired.CopilotInstructions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.CopilotInstructionsPath, Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.CopilotInstructions = changes
	}
	if desired.Runners != nil && len(*desired.Runners) > 0 {
		observed, err := read.Runners(repo, *desired.Runners)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.RunnersPath, Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareRunnerLabels(observed, *desired.Runners)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.RunnersPath, Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Runners = changes
	}
	if desired.OIDCSubject != nil {
		observed, err := read.OIDCSubject(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.OIDCSubjectPath, Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareOIDCSubject(observed, desired.OIDCSubject)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.OIDCSubjectPath, Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.OIDCSubject = changes
	}
	if desired.CopilotMCP != nil {
		observed, err := read.CopilotMCP(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.CopilotMCPPath, Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareCopilotMCP(observed, desired.CopilotMCP)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.CopilotMCPPath, Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.CopilotMCP = changes
	}
	if desired.PullRequestExemptions != nil {
		observed, err := read.PullRequestExemptions(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.PullRequestExemptionsPath, Target: repo, Err: err}
		}
		managed, changes, err := settings.ComparePullRequestExemptions(observed, desired.PullRequestExemptions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.PullRequestExemptionsPath, Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.PullRequestExemptions = changes
	}
	if !desired.PullRequestLimits.Empty() {
		observed, err := read.PullRequestLimits(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.PullRequestLimitsPath, Target: repo, Err: err}
		}
		managed, changes, err := settings.ComparePullRequestLimits(observed, desired.PullRequestLimits)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.PullRequestLimitsPath, Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.PullRequestLimits = changes
	}
	if desired.CopilotReviewEffort != nil {
		observed, err := read.CopilotReviewEffort(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.CopilotReviewEffortPath, Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareCopilotReviewEffort(observed, desired.CopilotReviewEffort)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.CopilotReviewEffortPath, Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.CopilotReviewEffort = changes
	}
	if desired.ActionsPolicies != nil && (desired.ActionsPolicies.Len() > 0 || prune.Has("actions-policies")) {
		observed, err := read.ActionsPolicies(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.policies", Target: repo, Err: err}
		}
		policies, err := settings.CompareActionsPoliciesWithPrune(observed, desired.ActionsPolicies, prune.Has("actions-policies"))
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.policies", Target: repo, Err: err}
		}
		comparison.Managed += policies.Managed
		comparison.ActionsPolicies = policies.ActionsPolicies
	}
	if desired.Autolinks != nil && (len(*desired.Autolinks) > 0 || prune.Has("autolinks")) {
		observed, err := read.Autolinks(repo)
		var changes []settings.AutolinkChange
		if err == nil {
			changes, err = settings.CompareAutolinks(observed, *desired.Autolinks, prune.Has("autolinks"))
		}
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.AutolinksPath, Target: repo, Err: err}
		}
		comparison.Managed++
		comparison.Autolinks = changes
	}
	if desired.BranchProtectionRules != nil && (len(*desired.BranchProtectionRules) > 0 || prune.Has("branch-protection-rules")) {
		observed, err := read.BranchProtectionRules(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.ClassicProtectionPath, Target: repo, Err: err}
		}
		changes, err := settings.CompareClassicRulesWithPrune(observed, *desired.BranchProtectionRules, prune.Has("branch-protection-rules"))
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.ClassicProtectionPath, Target: repo, Err: err}
		}
		comparison.Managed++
		comparison.BranchProtectionRules = changes
	}
	if desired.Rulesets != nil && (desired.Rulesets.Len() > 0 || prune.Has("rulesets")) {
		observed, err := read.Rulesets(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.rulesets", Target: repo, Err: err}
		}
		changes, err := settings.ReconcileRulesetsWithPrune(observed, desired.Rulesets, prune.Has("rulesets"))
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.rulesets", Target: repo, Err: err}
		}
		comparison.Managed++
		comparison.Rulesets = changes
	}
	if desired.Environments != nil && desired.Environments.Len() > 0 {
		observed, err := read.Environments(repo, environmentNames(desired.Environments))
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.environments", Target: repo, Err: err}
		}
		changes, err := settings.ReconcileEnvironments(observed, desired.Environments)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.environments", Target: repo, Err: err}
		}
		comparison.Managed++
		comparison.Environments = changes
		for _, change := range changes {
			if change.Operation() == "create" {
				createdEnvironments[settings.EnvironmentIdentity(change.Name())] = true
			}
		}
	}
	if prune.Has("environments") && desired.Environments != nil {
		if err := planEnvironmentPrune(repo, desired, read, &comparison); err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.environments", Target: repo, Err: err}
		}
	}
	if !settings.EnvironmentVariablesEmpty(desired.EnvironmentVariables) || prune.Has("environment-variables") && len(desired.EnvironmentVariables) > 0 {
		comparison.Managed++
	}
	for _, selected := range desired.EnvironmentVariables {
		changes, err := planVariables(repo, selected.Variables, func(repo repository.Repository, name string) (settings.VariableObservation, error) {
			if createdEnvironments[settings.EnvironmentIdentity(selected.Environment)] {
				return settings.NewAbsentVariableObservation(), nil
			}
			return read.EnvironmentVariable(repo, selected.Environment, name)
		})
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.environments.variables", Target: repo, Err: err}
		}

		if prune.Has("environment-variables") && !createdEnvironments[settings.EnvironmentIdentity(selected.Environment)] {
			inventory, readErr := read.EnvironmentVariables(repo, selected.Environment)
			if readErr != nil {
				return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.environments.variables", Target: repo, Err: readErr}
			}
			if _, readErr = inventory.EnvironmentID(selected.Environment); readErr != nil {
				return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.environments.variables", Target: repo, Err: readErr}
			}
			deletes, readErr := settings.PrunedVariableChanges(inventory, &selected.Variables)
			if readErr != nil {
				return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.environments.variables", Target: repo, Err: readErr}
			}
			changes = append(changes, deletes...)
		}

		for _, change := range changes {
			comparison.EnvironmentVariables = append(comparison.EnvironmentVariables, settings.EnvironmentVariableChange{Environment: selected.Environment, Change: change})
		}
	}
	if !desired.Actions.ExecutionEmpty() {
		observed, err := read.Actions(repo, !desired.Actions.SelectedActions.Empty())
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general", Target: repo, Err: err}
		}
		actionsComparison, err := settings.CompareActionsPolicy(observed, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general", Target: repo, Err: err}
		}
		comparison.Managed += actionsComparison.Managed
		comparison.Actions = actionsComparison.Changes
		comparison.ActionsExecution = actionsComparison.Execution
		comparison.ActionsExecutionContext = actionsComparison.ExecutionContext
	}
	if !desired.Actions.WorkflowPermissionsEmpty() {
		observed, err := read.WorkflowPermissions(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareWorkflowPermissions(observed, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if !desired.Actions.RetentionEmpty() {
		observed, err := read.Retention(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareRetention(observed, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if desired.Actions.CacheRetentionDays != nil {
		observed, err := read.CacheRetention(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general.cache.cache_retention", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareCacheRetention(observed, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general.cache.cache_retention", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if desired.Actions.CacheSizeGB != nil {
		observed, err := read.CacheStorage(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general.cache.cache_size_eviction_limit", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareCacheStorage(observed, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general.cache.cache_size_eviction_limit", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if desired.Actions.AccessLevel != nil {
		observed, err := read.ActionsAccess(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general.access", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareActionsAccess(observed, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general.access", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if desired.Actions.ForkApprovalPolicy != nil {
		o, err := read.ForkApproval(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareForkApproval(o, desired.Actions)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if !desired.Actions.PrivateFork.Empty() {
		o, err := read.PrivateFork(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general", Target: repo, Err: err}
		}
		managed, changes, err := settings.ComparePrivateFork(o, desired.Actions.PrivateFork)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.actions.general", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Actions = append(comparison.Actions, changes...)
	}
	if !desired.Security.SecretScanningEmpty() {
		observed, err := read.SecretScanning(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security", Target: repo, Err: err}
		}
		if _, err := settings.SecretScanningTransitions(observed, desired.Security); err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareSecretScanning(observed, desired.Security)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Security = changes
	}

	if !desired.Security.DependabotEmpty() {
		o, err := read.Dependabot(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security.dependabot", Target: repo, Err: err}
		}
		managed, changes, err := settings.DependabotTransitions(o, desired.Security)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security.dependabot", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Security = append(comparison.Security, changes...)
	}
	if !desired.Security.CodeQLConfiguration.Empty() {
		o, err := read.CodeQLConfiguration(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.CodeQLConfigurationPath, Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareCodeQLConfiguration(o, desired.Security.CodeQLConfiguration)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: settings.CodeQLConfigurationPath, Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.CodeQLConfiguration = changes
	}
	if desired.Security.CodeQLDefaultSetup != nil {
		observed, err := read.CodeQLDefaultSetup(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security.code_security.codeql_analysis.default_setup", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareCodeQLDefaultSetup(observed, desired.Security)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security.code_security.codeql_analysis.default_setup", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Security = append(comparison.Security, changes...)
	}

	if desired.Security.AIScanPullRequests != nil {
		observed, err := read.AIScanPullRequests(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_pull_requests", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareAIScanPullRequests(observed, desired.Security)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_pull_requests", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Security = append(comparison.Security, changes...)
	}

	if desired.Security.AIScanRepositories != nil {
		observed, err := read.AIScanRepositories(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_repositories", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareAIScanRepositories(observed, desired.Security)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_repositories", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Security = append(comparison.Security, changes...)
	}

	if desired.Security.PrivateVulnerabilityReporting != nil {
		observed, err := read.PrivateReporting(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security.private_vulnerability_reporting", Target: repo, Err: err}
		}
		managed, changes, err := settings.ComparePrivateReporting(observed, desired.Security)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.advanced_security.private_vulnerability_reporting", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Security = append(comparison.Security, changes...)
	}

	if desired.Security.ImmutableReleases != nil {
		observed, err := read.ImmutableReleases(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.general.releases.immutable_release", Target: repo, Err: err}
		}
		managed, changes, err := settings.CompareImmutableReleases(observed, desired.Security)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.general.releases.immutable_release", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Security = append(comparison.Security, changes...)
	}

	if !desired.Pages.Empty() {
		o, err := read.Pages(repo)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.pages", Target: repo, Err: err}
		}
		managed, changes, err := settings.ComparePages(o, desired.Pages)
		if err != nil {
			return PlanResult{}, &PlanError{Stage: "read-failure", Path: "repository.settings.pages", Target: repo, Err: err}
		}
		comparison.Managed += managed
		comparison.Pages = changes
	}
	return PlanResult{Target: repo, Comparison: comparison}, nil
}
