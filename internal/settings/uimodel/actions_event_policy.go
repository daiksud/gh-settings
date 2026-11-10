package uimodel

// ActionsEventPolicy applies its event restriction to all repository workflows.
type ActionsEventPolicy struct {
	ID                *int64
	PolicyName        *string
	EnforcementStatus *string
	RestrictEvents    *ActionsRestrictEvents
}

type ActionsRestrictEvents struct{ AllowedEvents *[]string }
