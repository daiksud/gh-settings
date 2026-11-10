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
		if value.PolicyName == nil || value.EnforcementStatus == nil || value.RestrictEvents == nil || value.RestrictEvents.AllowedEvents == nil {
			return nil, fmt.Errorf("Actions policy %d requires policy_name, enforcement_status and restrict_events.allowed_events", i)
		}
		entries[i] = settings.ActionsEventPolicy{Name: *value.PolicyName, Enforcement: *value.EnforcementStatus, Events: append([]string{}, (*value.RestrictEvents.AllowedEvents)...)}
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
	if _, err := settings.NewActionsPolicyDeclarations([]settings.ActionsEventPolicy{value}); err != nil {
		return uimodel.ActionsEventPolicy{}, err
	}
	name, enforcement := value.Name, value.Enforcement
	events := append([]string{}, value.Events...)
	result := uimodel.ActionsEventPolicy{PolicyName: &name, EnforcementStatus: &enforcement, RestrictEvents: &uimodel.ActionsRestrictEvents{AllowedEvents: &events}}
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
