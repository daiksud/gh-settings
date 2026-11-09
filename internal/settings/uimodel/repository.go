package uimodel

// Repository groups UI-oriented configuration beside its About dialog.
type Repository struct {
	About    *About
	Labels   *[]Label
	Settings *RepositorySettings
}

type RepositorySettings struct {
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
type ActionsSettings struct {
	General   *ActionsGeneral
	Variables *[]RepositoryVariable
}
type SecurityAndQuality struct{ AdvancedSecurity *AdvancedSecurity }
