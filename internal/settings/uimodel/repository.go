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
	Autolinks             *[]Autolink
	BranchProtectionRules *[]BranchProtectionRule
	Collaborators         *[]Collaborator
	InteractionLimits     *InteractionLimits
	Agents                *AgentsSettings
	Copilot               *CopilotSettings
	General               *General
	Rulesets              *[]Ruleset
	Actions               *ActionsSettings
	SecurityAndQuality    *SecurityAndQuality
	Pages                 *Pages
	Environments          *[]Environment
}

type RepositoryVariable struct {
	Name  string
	Value *string
}
type AgentsSettings struct{ Variables *[]RepositoryVariable }

type ActionsSettings struct {
	Policies  *[]ActionsEventPolicy
	General   *ActionsGeneral
	Variables *[]RepositoryVariable
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
