package github

import (
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func declareUIActionsForkApproval(choice *uimodel.ForkApprovalChoice) (*string, error) {
	if choice == nil {
		return nil, nil
	}
	value, valid := settings.ForkApprovalValueForChoice(*choice)
	if !valid {
		return nil, fmt.Errorf("unsupported UI fork workflow approval choice")
	}
	return &value, nil
}

func WithActionsForkApprovalPolicyUI(ui uimodel.ActionsGeneral, policy *string) (uimodel.ActionsGeneral, error) {
	if policy == nil {
		return ui, nil
	}
	choice, valid := settings.ForkApprovalChoiceForValue(*policy)
	if !valid {
		return uimodel.ActionsGeneral{}, fmt.Errorf("fork workflow approval is outside UI choices")
	}
	ui.ApprovalForRunningForkPullRequestWorkflowsFromContributors = &choice
	return ui, nil

}

func WithActionsForkApprovalObservation(ui uimodel.ActionsGeneral, observation settings.ForkApprovalObservation) (uimodel.ActionsGeneral, error) {
	value, err := observation.Value()
	if err != nil {
		return uimodel.ActionsGeneral{}, err
	}
	return WithActionsForkApprovalPolicyUI(ui, &value)
}
