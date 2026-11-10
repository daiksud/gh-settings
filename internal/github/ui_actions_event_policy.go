package github

import (
	"fmt"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func ActionsPoliciesDeclaration(values *[]uimodel.ActionsEventPolicy) (*settings.ActionsPolicyDeclarations, error) {
	if values == nil {
		return nil, nil
	}
	entries := make([]settings.ActionsEventPolicy, len(*values))
	for i, value := range *values {
		if value.PolicyName == nil || value.EnforcementStatus == nil {
			return nil, fmt.Errorf("Actions policy %d requires policy_name and enforcement_status", i)
		}
		entries[i] = settings.ActionsEventPolicy{Name: *value.PolicyName, Enforcement: *value.EnforcementStatus}
		if restriction := value.RestrictEvents; restriction != nil {
			if restriction.Enabled != nil && !*restriction.Enabled {
				if restriction.AllowedEvents != nil {
					return nil, fmt.Errorf("restrict_events.enabled false forbids allowed_events")
				}
				entries[i].DisableEvents = true
			} else {
				if restriction.AllowedEvents == nil {
					return nil, fmt.Errorf("restrict_events requires allowed_events")
				}
				entries[i].Events = append([]string{}, (*restriction.AllowedEvents)...)
			}
		}
		if restriction := value.RestrictActors; restriction != nil {
			if restriction.Enabled != nil && !*restriction.Enabled {
				if restriction.AllowedActors != nil {
					return nil, fmt.Errorf("restrict_actors.enabled false forbids allowed_actors")
				}
				entries[i].DisableActors = true
			} else {
				if restriction.AllowedActors == nil {
					return nil, fmt.Errorf("restrict_actors requires allowed_actors")
				}
				actors := make([]settings.ActionsPolicyActor, len(*restriction.AllowedActors))
				for j, actor := range *restriction.AllowedActors {
					if (actor.User == nil) == (actor.Role == nil) {
						return nil, fmt.Errorf("allowed_actors[%d] requires exactly one user or role", j)
					}
					if actor.User != nil {
						if *actor.User <= 0 {
							return nil, fmt.Errorf("allowed_actors[%d].user must be positive", j)
						}
						actors[j] = settings.ActionsPolicyActor{Type: "User", ID: *actor.User}
					} else {
						id, ok := actionsPolicyRoleIDs[*actor.Role]
						if !ok {
							return nil, fmt.Errorf("allowed_actors[%d] has unsupported role %q", j, *actor.Role)
						}
						actors[j] = settings.ActionsPolicyActor{Type: "RepositoryRole", ID: id}
					}
				}
				entries[i].Actors = &actors
			}
		}
		if value.TargetWorkflows != nil {
			if value.TargetWorkflows.Include == nil || value.TargetWorkflows.Exclude == nil {
				return nil, fmt.Errorf("target_workflows requires include and exclude arrays")
			}
			entries[i].TargetWorkflows = &settings.ActionsWorkflowTarget{Include: append([]string{}, (*value.TargetWorkflows.Include)...), Exclude: append([]string{}, (*value.TargetWorkflows.Exclude)...)}
		}
		if value.ID != nil {
			if *value.ID <= 0 {
				return nil, fmt.Errorf("Actions policy %d id must be positive", i)
			}
			entries[i].ID = *value.ID
		}
	}
	result, err := settings.NewActionsPolicyDeclarations(entries)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func ActionsEventPolicyUI(value settings.ActionsEventPolicy) (uimodel.ActionsEventPolicy, error) {
	if err := settings.ValidateActionsEventPolicyState(value); err != nil {
		return uimodel.ActionsEventPolicy{}, err
	}
	name, enforcement := value.Name, value.Enforcement
	off := false
	result := uimodel.ActionsEventPolicy{PolicyName: &name, EnforcementStatus: &enforcement,
		RestrictEvents: &uimodel.ActionsRestrictEvents{Enabled: &off},
		RestrictActors: &uimodel.ActionsRestrictActors{Enabled: &off}}
	if value.Events != nil {
		events := append([]string{}, value.Events...)
		result.RestrictEvents = &uimodel.ActionsRestrictEvents{AllowedEvents: &events}
	}
	if value.Actors != nil {
		actors := make([]uimodel.ActionsPolicyActor, len(*value.Actors))
		for i, actor := range *value.Actors {
			switch actor.Type {
			case "User":
				id := actor.ID
				actors[i].User = &id
			case "RepositoryRole":
				for role, id := range actionsPolicyRoleIDs {
					if actor.ID == id {
						name := role
						actors[i].Role = &name
						break
					}
				}
				if actors[i].Role == nil {
					return uimodel.ActionsEventPolicy{}, fmt.Errorf("unsupported Actions policy role ID %d", actor.ID)
				}
			default:
				return uimodel.ActionsEventPolicy{}, fmt.Errorf("unsupported Actions policy actor type %q", actor.Type)
			}
		}
		result.RestrictActors = &uimodel.ActionsRestrictActors{AllowedActors: &actors}
	}
	if value.TargetWorkflows != nil {
		target := settings.CanonicalActionsWorkflowTarget(value.TargetWorkflows)
		result.TargetWorkflows = &uimodel.ActionsWorkflowTarget{Include: &target.Include, Exclude: &target.Exclude}
	}
	if value.ID != 0 {
		id := value.ID
		result.ID = &id
	}
	return result, nil
}

// GitHub repository role IDs verified against the Actions policy UI.
var actionsPolicyRoleIDs = map[string]int64{"write": 4, "maintain": 2, "repository_admin": 5}
