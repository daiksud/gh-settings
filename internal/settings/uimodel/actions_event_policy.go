package uimodel

// ActionsEventPolicy applies its event restriction to selected repository workflows.
type ActionsEventPolicy struct {
	ID                *int64
	PolicyName        *string
	EnforcementStatus *string
	RestrictEvents    *ActionsRestrictEvents
	TargetWorkflows   *ActionsWorkflowTarget
}

type ActionsRestrictEvents struct{ AllowedEvents *[]string }

type ActionsWorkflowTarget struct {
	Include *[]string
	Exclude *[]string
}
