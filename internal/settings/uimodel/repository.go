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
}

type ActionsSettings struct{ General *ActionsGeneral }
type SecurityAndQuality struct{ AdvancedSecurity *AdvancedSecurity }
