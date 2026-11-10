package uimodel

// Repository groups UI-oriented configuration beside its About dialog.
type Repository struct {
	About    *About
	Labels   *[]Label
	Settings *RepositorySettings
}

type RepositorySettings struct {
	Agents             *AgentsSettings
	Copilot            *CopilotSettings
	General            *General
	Rulesets           *[]Ruleset
	Actions            *ActionsSettings
	SecurityAndQuality *SecurityAndQuality
	Pages              *Pages
	Environments       *[]Environment
}

type RepositoryVariable struct {
	Name  string
	Value *string
}
type AgentsSettings struct{ Variables *[]RepositoryVariable }

type ActionsSettings struct {
	General   *ActionsGeneral
	Variables *[]RepositoryVariable
}
type SecurityAndQuality struct{ AdvancedSecurity *AdvancedSecurity }

type CopilotSettings struct{ CodeReview *CopilotCodeReview }
type CopilotCodeReview struct{ UseCustomInstructions *bool }
