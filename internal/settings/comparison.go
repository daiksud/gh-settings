package settings

// Comparison counts declared fields and retains changes plus collaborator
// acceptance observations that remain relevant when no invitation is needed.
type Comparison struct {
	Webhooks                []WebhookChange
	Runners                 []RunnerLabelsChange
	RunnerDeletions         []RunnerDeletion
	DeployKeys              []DeployKeyChange
	Autolinks               []AutolinkChange
	CollaboratorDeletions   []CollaboratorDeletion
	BranchProtectionRules   []ClassicChange
	OIDCSubject             []OIDCSubjectChange
	Collaborators           []CollaboratorSelection
	CopilotInstructions     []CopilotInstructionsChange
	CopilotMCP              []CopilotToggleChange
	CopilotPublicSearch     []CopilotToggleChange
	PullRequestLimits       []PullRequestLimitsChange
	PullRequestExemptions   []PullRequestExemptionsChange
	CopilotReviewEffort     []CopilotReviewEffortChange
	Variables               []VariableChange
	AgentsVariables         []VariableChange
	CodeQLConfiguration     []CodeQLConfigurationChange
	Managed                 int
	Changes                 []Change
	CommitMessages          []NormalizedCommitMessageChange
	CommitMessageContexts   []CommitMessageContext
	Labels                  LabelChanges
	Rulesets                []RulesetChange
	Environments            []EnvironmentChange
	EnvironmentVariables    []EnvironmentVariableChange
	Actions                 []ActionsPolicyChange
	ActionsPolicies         []ActionsEventPolicyChange
	ActionsExecution        []ActionsExecutionChange
	ActionsExecutionContext *ActionsExecutionContext
	Security                []SecurityChange
	Pages                   []PagesChange
}

// HasChanges includes actionable normalized fallback matches, not only unequal
// effective values. Observation validation remains the producing operation's job.
func (comparison Comparison) HasChanges() bool {
	for _, selected := range comparison.Collaborators {
		if selected.Changes() {
			return true
		}
	}
	return len(comparison.Webhooks) > 0 || len(comparison.Runners) > 0 || len(comparison.RunnerDeletions) > 0 || len(comparison.DeployKeys) > 0 || len(comparison.Autolinks) > 0 || len(comparison.OIDCSubject) > 0 || len(comparison.BranchProtectionRules) > 0 || len(comparison.CollaboratorDeletions) > 0 || len(comparison.PullRequestExemptions) > 0 || len(comparison.EnvironmentVariables) > 0 || len(comparison.CopilotReviewEffort) > 0 || len(comparison.PullRequestLimits) > 0 || len(comparison.CopilotMCP) > 0 || len(comparison.CopilotPublicSearch) > 0 || len(comparison.CopilotInstructions) > 0 || len(comparison.Variables) > 0 || len(comparison.AgentsVariables) > 0 || len(comparison.CodeQLConfiguration) > 0 || len(comparison.Changes) > 0 || len(comparison.CommitMessages) > 0 || comparison.Labels.Len() > 0 || len(comparison.Rulesets) > 0 || len(comparison.Environments) > 0 || len(comparison.Actions) > 0 || len(comparison.ActionsPolicies) > 0 || len(comparison.ActionsExecution) > 0 || len(comparison.Security) > 0 || len(comparison.Pages) > 0
}

// Change contains a supported field identifier and a matching typed value pair.
type Change struct {
	field         FieldID
	values        changeValues
	normalization ObservationNormalization
}

// Field returns the validated field identity without exposing a mutable path.
func (change Change) Field() FieldID { return change.field }

func (change Change) Normalization() ObservationNormalization { return change.normalization }
