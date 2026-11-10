package uimodel

// ActionsEventPolicy applies its restrictions to selected repository workflows.
type ActionsEventPolicy struct {
	ID                *int64
	PolicyName        *string
	EnforcementStatus *string
	RestrictEvents    *ActionsRestrictEvents
	RestrictActors    *ActionsRestrictActors
	TargetWorkflows   *ActionsWorkflowTarget
}

type ActionsRestrictEvents struct {
	Enabled       *bool
	AllowedEvents *[]string
}

// ActionsRestrictActors selects users and repository roles allowed to run workflows.
// A present empty AllowedActors slice intentionally selects no actors.
type ActionsRestrictActors struct {
	Enabled       *bool
	AllowedActors *[]ActionsPolicyActor
}

// ActionsPolicyActor has exactly one identity: User or Role.
type ActionsPolicyActor struct {
	User *int64
	Role *string
}

type ActionsWorkflowTarget struct {
	Include *[]string
	Exclude *[]string
}
