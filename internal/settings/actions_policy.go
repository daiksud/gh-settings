package settings

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const ActionsPolicyFieldCount = 14

// ActionsPolicyDeclaration contains only explicitly managed repository policy values.
type ActionsPolicyDeclaration struct {
	ForkApprovalPolicy           *string                    `yaml:"fork_pr_approval_policy,omitempty" json:"fork_pr_approval_policy,omitempty"`
	PrivateFork                  PrivateForkDeclaration     `yaml:"private_fork_workflows,omitempty" json:"private_fork_workflows,omitempty"`
	RetentionDays                *int                       `yaml:"artifact_and_log_retention_days,omitempty" json:"artifact_and_log_retention_days,omitempty"`
	DefaultWorkflowPermissions   *string                    `yaml:"default_workflow_permissions,omitempty" json:"default_workflow_permissions,omitempty"`
	CanApprovePullRequestReviews *bool                      `yaml:"can_approve_pull_request_reviews,omitempty" json:"can_approve_pull_request_reviews,omitempty"`
	Enabled                      *bool                      `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	AllowedActions               *string                    `yaml:"allowed_actions,omitempty" json:"allowed_actions,omitempty"`
	SHAPinningRequired           *bool                      `yaml:"sha_pinning_required,omitempty" json:"sha_pinning_required,omitempty"`
	SelectedActions              SelectedActionsDeclaration `yaml:"selected_actions,omitempty" json:"selected_actions,omitempty"`
}

type SelectedActionsDeclaration struct {
	GitHubOwnedAllowed *bool     `yaml:"github_owned_allowed,omitempty" json:"github_owned_allowed,omitempty"`
	VerifiedAllowed    *bool     `yaml:"verified_allowed,omitempty" json:"verified_allowed,omitempty"`
	PatternsAllowed    *[]string `yaml:"patterns_allowed,omitempty" json:"patterns_allowed,omitempty"`
}

type SelectedActionsState struct {
	GitHubOwnedAllowed bool
	VerifiedAllowed    bool
	PatternsAllowed    []string
}

// A nil Selected means inactive or not acquired, never an empty selected policy.
type ActionsPolicyState struct {
	Enabled                     bool
	AllowedActions              string
	SHAPinningRequired          bool
	Selected                    *SelectedActionsState
	AllowedActionsNormalization ObservationNormalization
}

func (d SelectedActionsDeclaration) Empty() bool {
	return d.GitHubOwnedAllowed == nil && d.VerifiedAllowed == nil && d.PatternsAllowed == nil
}
func (d SelectedActionsDeclaration) Complete() bool {
	return d.GitHubOwnedAllowed != nil && d.VerifiedAllowed != nil && d.PatternsAllowed != nil
}
func (d ActionsPolicyDeclaration) ExecutionEmpty() bool {
	return d.Enabled == nil && d.AllowedActions == nil && d.SHAPinningRequired == nil && d.SelectedActions.Empty()
}
func ValidAllowedActions(value string) bool {
	_, valid := ActionsPermissionsChoiceForValue(true, value)
	return valid
}

func ValidateActionPatterns(values []string) error {
	if values == nil {
		return fmt.Errorf("Actions patterns must be a non-null array")
	}
	if len(values) > 1000 {
		return fmt.Errorf("repository.settings.actions.general.allow_or_block_specified_actions_and_reusable_workflows must contain at most 1000 patterns")
	}
	for _, v := range values {
		if v == "" {
			return fmt.Errorf("Actions patterns must be nonempty strings")
		}
	}
	return nil
}

func (d ActionsPolicyDeclaration) Validate() error {
	if d.ForkApprovalPolicy != nil && !ValidForkApprovalPolicy(*d.ForkApprovalPolicy) {
		return fmt.Errorf("repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors must be a supported choice")
	}
	if d.RetentionDays != nil {
		if err := ValidateRetentionDays(*d.RetentionDays); err != nil {
			return err
		}
	}
	if d.DefaultWorkflowPermissions != nil && !ValidDefaultWorkflowPermissions(*d.DefaultWorkflowPermissions) {
		return fmt.Errorf("repository.settings.actions.general.workflow_permissions must be a supported choice")
	}
	if d.AllowedActions != nil && !ValidAllowedActions(*d.AllowedActions) {
		return fmt.Errorf("repository.settings.actions.general.actions_permissions must be a supported choice")
	}
	if !d.SelectedActions.Empty() && d.AllowedActions != nil && *d.AllowedActions != "selected" {
		return fmt.Errorf("selected Actions controls require the selected non-owner actions_permissions choice or omission")
	}
	if d.SelectedActions.PatternsAllowed != nil {
		return ValidateActionPatterns(*d.SelectedActions.PatternsAllowed)
	}
	return nil
}

func (s ActionsPolicyState) Validate() error {
	if !ValidAllowedActions(s.AllowedActions) {
		return fmt.Errorf("unsupported observed Actions permissions value")
	}
	if record := s.AllowedActionsNormalization; record != (ObservationNormalization{}) {
		_, fallback, _ := ActionsPermissionsValueForChoice(DefaultActionsPermissionsChoice)
		if s.AllowedActions != *fallback || record.Path != ActionsPermissionsPath || record.Fallback != string(DefaultActionsPermissionsChoice) || ValidAllowedActions(record.RawValue) || s.Selected != nil {
			return fmt.Errorf("invalid Actions permission normalization provenance")
		}
	}
	if s.Selected != nil {
		if s.AllowedActions != "selected" || s.Selected.PatternsAllowed == nil {
			return fmt.Errorf("invalid acquired selected Actions policy")
		}
		return ValidateActionPatterns(s.Selected.PatternsAllowed)
	}
	return nil
}

func (s ActionsPolicyState) RawAllowedActions() (string, error) {
	if err := s.Validate(); err != nil {
		return "", err
	}
	if s.AllowedActionsNormalization != (ObservationNormalization{}) {
		return s.AllowedActionsNormalization.RawValue, nil
	}
	return s.AllowedActions, nil
}

// PrepareActions validates all observable prerequisites before any resource write.
func PrepareActions(s ActionsPolicyState, d ActionsPolicyDeclaration) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := d.Validate(); err != nil {
		return err
	}
	if !s.Enabled && d.Enabled != nil && *d.Enabled && d.AllowedActions == nil && s.AllowedActionsNormalization != (ObservationNormalization{}) {
		return fmt.Errorf("enabling Actions from an unknown policy requires an explicit canonical actions_permissions choice")
	}
	if s.AllowedActions != "selected" && d.AllowedActions != nil && *d.AllowedActions == "selected" && !d.SelectedActions.Complete() {
		return fmt.Errorf("transition to selected requires all three selected Actions controls; inactive values cannot be read")
	}
	if d.SelectedActions.Empty() {
		return nil
	}
	if s.AllowedActions == "selected" {
		if s.Selected == nil {
			return fmt.Errorf("selected Actions observation is missing")
		}
		return nil
	}
	if d.AllowedActions == nil || *d.AllowedActions != "selected" {
		return fmt.Errorf("selected Actions values are not observable under the current unmanaged actions_permissions policy; declare the selected non-owner choice and all three selected Actions controls")
	}
	if !d.SelectedActions.Complete() {
		return fmt.Errorf("transition to selected requires all three selected Actions controls; inactive values cannot be read")
	}
	return nil
}

func (s ActionsPolicyState) Declaration() (ActionsPolicyDeclaration, error) {
	if err := s.Validate(); err != nil {
		return ActionsPolicyDeclaration{}, err
	}
	d := ActionsPolicyDeclaration{Enabled: &s.Enabled, AllowedActions: &s.AllowedActions, SHAPinningRequired: &s.SHAPinningRequired}
	if s.AllowedActions == "selected" {
		if s.Selected == nil {
			return ActionsPolicyDeclaration{}, fmt.Errorf("selected Actions observation is missing")
		}
		selected := *s.Selected
		patterns := append([]string{}, selected.PatternsAllowed...)
		d.SelectedActions = SelectedActionsDeclaration{GitHubOwnedAllowed: &selected.GitHubOwnedAllowed, VerifiedAllowed: &selected.VerifiedAllowed, PatternsAllowed: &patterns}
	}
	return d, nil
}

// ActionsPolicyChange is a closed native-value comparison. A nil before is permitted
// only for inactive selected settings; it never means an observed false/empty value.
type ActionsPolicyChange struct {
	field         string
	before, after any
	normalization ObservationNormalization
}

func (c ActionsPolicyChange) Field() string                           { return c.field }
func (c ActionsPolicyChange) Normalization() ObservationNormalization { return c.normalization }
func copyActionValue(v any) any {
	if a, ok := v.([]string); ok {
		return append([]string{}, a...)
	}
	return v
}
func (c ActionsPolicyChange) Values() (any, any) {
	return copyActionValue(c.before), copyActionValue(c.after)
}

func actionValue(field string, value any) (any, error) {
	switch field {
	case "actions.artifact_and_log_retention_days":
		if v, ok := value.(int); ok && ValidateRetentionDays(v) == nil {
			return v, nil
		}
		if v, ok := value.(float64); ok && v >= 1 && v <= 400 && v == float64(int(v)) {
			return int(v), nil
		}
	case "actions.enabled", "actions.sha_pinning_required", "actions.selected_actions.github_owned_allowed", "actions.selected_actions.verified_allowed", "actions.can_approve_pull_request_reviews":
		if v, ok := value.(bool); ok {
			return v, nil
		}
	case "actions.fork_pr_approval_policy":
		if v, ok := value.(string); ok && ValidForkApprovalPolicy(v) {
			return v, nil
		}
	case "actions.private_fork_workflows.run_workflows_from_fork_pull_requests", "actions.private_fork_workflows.send_write_tokens_to_workflows", "actions.private_fork_workflows.send_secrets_and_variables", "actions.private_fork_workflows.require_approval_for_fork_pr_workflows":
		if v, ok := value.(bool); ok {
			return v, nil
		}
	case "actions.default_workflow_permissions":
		if v, ok := value.(string); ok && ValidDefaultWorkflowPermissions(v) {
			return v, nil
		}
	case "actions.allowed_actions":
		if v, ok := value.(string); ok && ValidAllowedActions(v) {
			return v, nil
		}
	case "actions.selected_actions.patterns_allowed":
		v, ok := value.([]string)
		if !ok {
			if raw, yes := value.([]any); yes {
				v = []string{}
				ok = true
				for _, item := range raw {
					str, yes := item.(string)
					if !yes {
						ok = false
						break
					}
					v = append(v, str)
				}
			}
		}
		if ok && v != nil && ValidateActionPatterns(v) == nil {
			return append([]string{}, v...), nil
		}
	}
	return nil, fmt.Errorf("invalid Actions comparison value for %q", field)
}

func NewActionsPolicyChange(field string, before, after any) (ActionsPolicyChange, error) {
	after, err := actionValue(field, after)
	if err != nil {
		return ActionsPolicyChange{}, err
	}
	if before != nil {
		before, err = actionValue(field, before)
		if err != nil {
			return ActionsPolicyChange{}, err
		}
	} else if !slices.Contains([]string{"actions.selected_actions.github_owned_allowed", "actions.selected_actions.verified_allowed", "actions.selected_actions.patterns_allowed"}, field) {
		return ActionsPolicyChange{}, fmt.Errorf("Actions value %q must be observable", field)
	}
	if reflect.DeepEqual(before, after) {
		return ActionsPolicyChange{}, fmt.Errorf("Actions comparison must differ")
	}
	return ActionsPolicyChange{field: field, before: before, after: after}, nil
}

// ValidateActionsPolicyChanges also verifies the only legal unobservable transition.
func ValidateActionsPolicyChanges(changes []ActionsPolicyChange, execution ...ActionsExecutionChange) error {
	if len(execution) > 1 {
		return fmt.Errorf("duplicate Actions execution radio decision")
	}
	if len(execution) == 1 {
		radio := execution[0]
		_, err := NewActionsExecutionChange(ActionsPolicyState{Enabled: radio.beforeEnabled, AllowedActions: "all", AllowedActionsNormalization: radio.normalization}, radio.after)
		if err != nil {
			return err
		}
		selectedTarget := radio.after == uimodel.AllowOwnerAndSelectNonOwnerActionsAndReusableWorkflows
		for _, change := range changes {
			if change.field == "actions.enabled" || change.field == "actions.allowed_actions" {
				return fmt.Errorf("Actions radio overlaps native execution policy evidence")
			}
			if !selectedTarget && strings.HasPrefix(change.field, "actions.selected_actions.") {
				return fmt.Errorf("nonselected Actions radio cannot contain selected companion changes")
			}
		}
		if selectedTarget {
			transition, err := NewActionsPolicyChange("actions.allowed_actions", "all", "selected")
			if err != nil {
				return err
			}
			// This local component supplies structural transition evidence only;
			// it is never another rendered decision or desired declaration.
			changes = append(append([]ActionsPolicyChange{}, changes...), transition)
		}
	}
	seen := map[string]bool{}
	unknown := 0
	transition := false
	selectedChanges := 0
	leavingSelected := false
	for _, c := range changes {
		if seen[c.field] {
			return fmt.Errorf("duplicate Actions change")
		}
		seen[c.field] = true
		var err error
		if c.normalization != (ObservationNormalization{}) {
			_, err = NewActionsPolicyChangeWithNormalization(c.field, c.before, c.after, c.normalization)
		} else {
			_, err = NewActionsPolicyChange(c.field, c.before, c.after)
		}
		if err != nil {
			return err
		}
		if c.before == nil {
			unknown++
		}
		if strings.HasPrefix(c.field, "actions.selected_actions.") {
			selectedChanges++
		}
		if c.field == "actions.allowed_actions" && c.before != "selected" && c.after == "selected" {
			transition = true
		}
		if c.field == "actions.allowed_actions" && c.after != "selected" {
			leavingSelected = true
		}
	}
	if (unknown > 0 && !transition) || (transition && unknown != 3) || (leavingSelected && selectedChanges > 0) {
		return fmt.Errorf("invalid Actions policy/selection transition: inactive values must be unavailable and selected writes require selected policy")
	}
	return nil
}

type ActionsPolicyComparison struct {
	Managed          int
	Changes          []ActionsPolicyChange
	Execution        []ActionsExecutionChange
	ExecutionContext *ActionsExecutionContext
}

func CompareActionsPolicy(s ActionsPolicyState, d ActionsPolicyDeclaration) (ActionsPolicyComparison, error) {
	if err := PrepareActions(s, d); err != nil {
		return ActionsPolicyComparison{}, err
	}
	managed := 0
	changes := []ActionsPolicyChange{}
	var execution []ActionsExecutionChange
	var executionContext *ActionsExecutionContext
	radioIntent := false
	if s.AllowedActionsNormalization != (ObservationNormalization{}) {
		if d.AllowedActions != nil && (d.Enabled == nil || !*d.Enabled) {
			return ActionsPolicyComparison{}, fmt.Errorf("normalized Actions policy intent requires a complete enabling radio choice")
		}
		radioIntent = d.Enabled != nil && (d.AllowedActions != nil || !*d.Enabled)
		if radioIntent {
			allowed := ""
			if d.AllowedActions != nil {
				allowed = *d.AllowedActions
			}
			choice, valid := ActionsPermissionsChoiceForValue(*d.Enabled, allowed)
			if !valid {
				return ActionsPolicyComparison{}, fmt.Errorf("invalid desired Actions radio choice")
			}
			if s.Enabled || choice != uimodel.DisableActions {
				change, err := NewActionsExecutionChange(s, choice)
				if err != nil {
					return ActionsPolicyComparison{}, err
				}
				execution = append(execution, change)
			}
		}
	}
	if s.AllowedActionsNormalization == (ObservationNormalization{}) && d.Enabled != nil && (d.AllowedActions != nil || !*d.Enabled) {
		before, valid := ActionsPermissionsChoiceForValue(s.Enabled, s.AllowedActions)
		allowed := ""
		if d.AllowedActions != nil {
			allowed = *d.AllowedActions
		}
		after, afterValid := ActionsPermissionsChoiceForValue(*d.Enabled, allowed)
		if !valid || !afterValid {
			return ActionsPolicyComparison{}, fmt.Errorf("invalid Actions radio choice")
		}
		if before != after {
			executionContext = &ActionsExecutionContext{Before: before, After: after, ObservedAllowed: s.AllowedActions}
		}
	}
	add := func(field string, before, after any) {
		managed++
		if radioIntent && (field == "actions.enabled" || field == "actions.allowed_actions") {
			return
		}
		if !reflect.DeepEqual(before, after) {
			changes = append(changes, ActionsPolicyChange{field: field, before: copyActionValue(before), after: copyActionValue(after)})
		}
	}
	if d.Enabled != nil {
		add("actions.enabled", s.Enabled, *d.Enabled)
	}
	if d.AllowedActions != nil {
		add("actions.allowed_actions", s.AllowedActions, *d.AllowedActions)
	}
	if d.SHAPinningRequired != nil {
		add("actions.sha_pinning_required", s.SHAPinningRequired, *d.SHAPinningRequired)
	}
	var owned, verified, patterns any
	if s.Selected != nil {
		owned, verified, patterns = s.Selected.GitHubOwnedAllowed, s.Selected.VerifiedAllowed, s.Selected.PatternsAllowed
	}
	if v := d.SelectedActions.GitHubOwnedAllowed; v != nil {
		add("actions.selected_actions.github_owned_allowed", owned, *v)
	}
	if v := d.SelectedActions.VerifiedAllowed; v != nil {
		add("actions.selected_actions.verified_allowed", verified, *v)
	}
	if v := d.SelectedActions.PatternsAllowed; v != nil {
		add("actions.selected_actions.patterns_allowed", patterns, *v)
	}
	if err := ValidateActionsPolicyChanges(changes, execution...); err != nil {
		return ActionsPolicyComparison{}, err
	}
	return ActionsPolicyComparison{Managed: managed, Changes: changes, Execution: execution, ExecutionContext: executionContext}, nil
}
