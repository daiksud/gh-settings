package uimodel

// Repository groups UI-oriented configuration beside its About dialog.
type Repository struct {
	About    *About
	Labels   *[]Label
	Settings *RepositorySettings
}

type Collaborator struct {
	Username string
	ID       *int64
}

type RepositorySettings struct {
	Webhooks              *[]Webhook
	DeployKeys            *[]DeployKey
	Autolinks             *[]Autolink
	BranchProtectionRules *[]BranchProtectionRule
	Collaborators         *[]Collaborator
	InteractionLimits     *InteractionLimits
	SecretsAndVariables   *SecretsAndVariables
	Copilot               *CopilotSettings
	General               *General
	Rulesets              *[]Ruleset
	Actions               *ActionsSettings
	SecurityAndQuality    *SecurityAndQuality
	Pages                 *Pages
	Environments          *[]Environment

	// IgnoredPullRequestLimitKeys records inactive input key names for diagnostics.
	IgnoredPullRequestLimitKeys []string
}

type RepositoryVariable struct {
	Name  string
	Value *string
}

type SecretsAndVariables struct {
	Actions *RepositoryVariablesSettings
	Agents  *RepositoryVariablesSettings
}

type RepositoryVariablesSettings struct{ Variables *[]RepositoryVariable }

type ActionsSettings struct {
	Runners  *[]RunnerLabels
	OIDC     *OIDCSettings
	Policies *[]ActionsEventPolicy
	General  *ActionsGeneral
}
type SecurityAndQuality struct{ AdvancedSecurity *AdvancedSecurity }

type CopilotSettings struct{ CodeReview *CopilotCodeReview }
type CopilotCodeReview struct {
	UseCustomInstructions                              *bool
	AllowCopilotToUseMCPToolsWhenReviewingPullRequests *bool
	ReviewEffortLevel                                  *string
}

// InteractionLimits contains only the persistent Pull request limits controls.
type InteractionLimits struct{ PullRequestLimits *PullRequestLimits }
type PullRequestLimits struct {
	ExemptionList                                    *[]string
	LimitOpenPullRequestsFromUsersWithoutWriteAccess *bool
	MaximumOpenPullRequestsPerUser                   *int
	CountDraftPullRequestsTowardTheLimit             *bool
}
