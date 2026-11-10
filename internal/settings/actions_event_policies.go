package settings

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ActionsEventPolicy owns one repository policy applying to all workflows.
// ID zero requests creation; positive IDs select existing repository objects.
type ActionsEventPolicy struct {
	ID          int64
	Name        string
	Enforcement string
	Events      []string
}

var actionsPolicyEvents = []string{
	"branch_protection_rule", "check_run", "check_suite", "create", "delete", "deployment", "deployment_status", "discussion", "discussion_comment", "fork", "gollum", "image_version", "issue_comment", "issues", "label", "merge_group", "milestone", "page_build", "project", "project_card", "project_column", "public", "pull_request", "pull_request_review", "pull_request_review_comment", "pull_request_target", "push", "registry_package", "release", "repository_dispatch", "schedule", "status", "watch", "workflow_call", "workflow_dispatch", "workflow_run",
}

// SupportedActionsPolicyEvents returns the official supported event vocabulary.
func SupportedActionsPolicyEvents() []string { return slices.Clone(actionsPolicyEvents) }

func validateActionsPolicyName(name string) error {
	if strings.TrimSpace(name) == "" || !utf8.ValidString(name) {
		return fmt.Errorf("Actions policy name must be nonempty")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return fmt.Errorf("Actions policy name must not contain control characters")
		}
	}
	return nil
}

// ValidateActionsEventPolicy validates the complete supported writable object.
func ValidateActionsEventPolicy(policy ActionsEventPolicy) error {
	if policy.ID < 0 {
		return fmt.Errorf("Actions policy id must be positive when supplied")
	}
	if err := validateActionsPolicyName(policy.Name); err != nil {
		return err
	}
	if policy.Enforcement != "disabled" && policy.Enforcement != "active" {
		return fmt.Errorf("Actions policy %q enforcement must be disabled or active", policy.Name)
	}
	if len(policy.Events) == 0 {
		return fmt.Errorf("Actions policy %q restrict_events must not be empty", policy.Name)
	}
	seen := map[string]bool{}
	for _, event := range policy.Events {
		if !slices.Contains(actionsPolicyEvents, event) {
			return fmt.Errorf("Actions policy %q has unsupported event %q", policy.Name, event)
		}
		if seen[event] {
			return fmt.Errorf("Actions policy %q has duplicate event %q", policy.Name, event)
		}
		seen[event] = true
	}
	return nil
}

func cloneActionsEventPolicy(policy ActionsEventPolicy) ActionsEventPolicy {
	policy.Events = slices.Clone(policy.Events)
	return policy
}
func canonicalActionsEventPolicy(policy ActionsEventPolicy) ActionsEventPolicy {
	policy = cloneActionsEventPolicy(policy)
	slices.Sort(policy.Events)
	return policy
}

// ActionsPolicyDeclarations owns validated, unique whole-object selections.
// Both omission and an empty collection leave all remote policies unmanaged.
type ActionsPolicyDeclarations struct{ entries []ActionsEventPolicy }

func NewActionsPolicyDeclarations(values []ActionsEventPolicy) (ActionsPolicyDeclarations, error) {
	entries := make([]ActionsEventPolicy, len(values))
	ids := map[int64]bool{}
	names := map[string]bool{}
	for i, value := range values {
		if err := ValidateActionsEventPolicy(value); err != nil {
			return ActionsPolicyDeclarations{}, err
		}
		if names[value.Name] {
			return ActionsPolicyDeclarations{}, fmt.Errorf("duplicate Actions policy name %q", value.Name)
		}
		names[value.Name] = true
		if value.ID != 0 {
			if ids[value.ID] {
				return ActionsPolicyDeclarations{}, fmt.Errorf("duplicate Actions policy id %d", value.ID)
			}
			ids[value.ID] = true
		}
		entries[i] = canonicalActionsEventPolicy(value)
	}
	return ActionsPolicyDeclarations{entries: entries}, nil
}
func (d ActionsPolicyDeclarations) Len() int { return len(d.entries) }
func (d ActionsPolicyDeclarations) Values() []ActionsEventPolicy {
	result := make([]ActionsEventPolicy, len(d.entries))
	for i, value := range d.entries {
		result[i] = cloneActionsEventPolicy(value)
	}
	return result
}

// ActionsEventPolicyState retains identity even when the remote object cannot be
// represented without losing unsupported rules, targeting, or inherited ownership.
// Exactly one of Policy and Unsupported must be populated.
type ActionsEventPolicyState struct {
	ID          int64
	Name        string
	Policy      *ActionsEventPolicy
	Unsupported string
}

// ActionsPolicyObservation represents a complete acquired repository policy read.
// Its zero value is missing, distinct from an acquired empty collection.
type ActionsPolicyObservation struct {
	acquired bool
	entries  []ActionsEventPolicyState
}

func NewActionsPolicyObservation(values []ActionsEventPolicyState) (ActionsPolicyObservation, error) {
	entries := make([]ActionsEventPolicyState, len(values))
	ids := map[int64]bool{}
	for i, value := range values {
		if value.ID <= 0 || ids[value.ID] {
			return ActionsPolicyObservation{}, fmt.Errorf("invalid or duplicate observed Actions policy id %d", value.ID)
		}
		ids[value.ID] = true
		if err := validateActionsPolicyName(value.Name); err != nil {
			return ActionsPolicyObservation{}, err
		}
		if (value.Policy == nil) == (value.Unsupported == "") {
			return ActionsPolicyObservation{}, fmt.Errorf("Actions policy id %d requires either supported state or unsupported reason", value.ID)
		}
		if value.Policy != nil {
			if value.Policy.ID != value.ID || value.Policy.Name != value.Name {
				return ActionsPolicyObservation{}, fmt.Errorf("Actions policy observation identity mismatch for id %d", value.ID)
			}
			if err := ValidateActionsEventPolicy(*value.Policy); err != nil {
				return ActionsPolicyObservation{}, err
			}
			policy := canonicalActionsEventPolicy(*value.Policy)
			value.Policy = &policy
		}
		entries[i] = value
	}
	return ActionsPolicyObservation{acquired: true, entries: entries}, nil
}
func (o ActionsPolicyObservation) Values() ([]ActionsEventPolicyState, error) {
	if !o.acquired {
		return nil, fmt.Errorf("missing Actions policies observation")
	}
	values := make([]ActionsEventPolicyState, len(o.entries))
	for i, value := range o.entries {
		if value.Policy != nil {
			policy := cloneActionsEventPolicy(*value.Policy)
			value.Policy = &policy
		}
		values[i] = value
	}
	return values, nil
}

// ActionsEventPolicyChange is a validated create or whole-object update.
type ActionsEventPolicyChange struct {
	before *ActionsEventPolicy
	after  ActionsEventPolicy
}

func NewActionsEventPolicyChange(before *ActionsEventPolicy, after ActionsEventPolicy) (ActionsEventPolicyChange, error) {
	if err := ValidateActionsEventPolicy(after); err != nil {
		return ActionsEventPolicyChange{}, err
	}
	after = canonicalActionsEventPolicy(after)
	if before == nil {
		if after.ID != 0 {
			return ActionsEventPolicyChange{}, fmt.Errorf("Actions policy creation must omit id")
		}
		return ActionsEventPolicyChange{after: after}, nil
	}
	if err := ValidateActionsEventPolicy(*before); err != nil {
		return ActionsEventPolicyChange{}, err
	}
	if before.ID <= 0 || after.ID != before.ID {
		return ActionsEventPolicyChange{}, fmt.Errorf("Actions policy update requires matching positive id")
	}
	owned := canonicalActionsEventPolicy(*before)
	if equalActionsEventPolicies(owned, after) {
		return ActionsEventPolicyChange{}, fmt.Errorf("unchanged Actions policy")
	}
	return ActionsEventPolicyChange{before: &owned, after: after}, nil
}
func (c ActionsEventPolicyChange) Before() *ActionsEventPolicy {
	if c.before == nil {
		return nil
	}
	value := cloneActionsEventPolicy(*c.before)
	return &value
}
func (c ActionsEventPolicyChange) After() ActionsEventPolicy { return cloneActionsEventPolicy(c.after) }
func (c ActionsEventPolicyChange) ID() int64                 { return c.after.ID }
func (c ActionsEventPolicyChange) Name() string              { return c.after.Name }
func (c ActionsEventPolicyChange) Operation() string {
	if c.before == nil {
		return "create"
	}
	return "update"
}
func equalActionsEventPolicies(a, b ActionsEventPolicy) bool {
	return a.ID == b.ID && a.Name == b.Name && a.Enforcement == b.Enforcement && slices.Equal(a.Events, b.Events)
}

// ReconcileActionsPolicies selects only declared IDs and never adopts by name.
func ReconcileActionsPolicies(observed ActionsPolicyObservation, desired *ActionsPolicyDeclarations) ([]ActionsEventPolicyChange, error) {
	if desired == nil || desired.Len() == 0 {
		return nil, nil
	}
	values, err := observed.Values()
	if err != nil {
		return nil, err
	}
	byID := map[int64]ActionsEventPolicyState{}
	byName := map[string][]int64{}
	for _, value := range values {
		byID[value.ID] = value
		byName[value.Name] = append(byName[value.Name], value.ID)
	}
	declarations := desired.Values()
	slices.SortFunc(declarations, func(a, b ActionsEventPolicy) int { return strings.Compare(a.Name, b.Name) })
	var changes []ActionsEventPolicyChange
	for _, after := range declarations {
		for _, id := range byName[after.Name] {
			if id != after.ID {
				return nil, fmt.Errorf("Actions policy name %q already belongs to id %d; specify its id to update and avoid name collisions", after.Name, id)
			}
		}
		var before *ActionsEventPolicy
		if after.ID != 0 {
			state, exists := byID[after.ID]
			if !exists {
				return nil, fmt.Errorf("Actions policy id %d does not exist in this repository; refusing create fallback", after.ID)
			}
			if state.Unsupported != "" {
				return nil, fmt.Errorf("Actions policy id %d is unsupported: %s", after.ID, state.Unsupported)
			}
			before = state.Policy
			if equalActionsEventPolicies(*before, after) {
				continue
			}
		}
		change, err := NewActionsEventPolicyChange(before, after)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}
func CompareActionsPolicies(observed ActionsPolicyObservation, desired *ActionsPolicyDeclarations) (Comparison, error) {
	changes, err := ReconcileActionsPolicies(observed, desired)
	if err != nil {
		return Comparison{}, err
	}
	result := Comparison{ActionsPolicies: changes}
	if desired != nil && desired.Len() > 0 {
		result.Managed = 1
	}
	return result, nil
}

// ExportActionsPolicies refuses lossy exports of unsupported remote objects.
func ExportActionsPolicies(observed ActionsPolicyObservation) (ActionsPolicyDeclarations, error) {
	values, err := observed.Values()
	if err != nil {
		return ActionsPolicyDeclarations{}, err
	}
	policies := make([]ActionsEventPolicy, 0, len(values))
	for _, value := range values {
		if value.Unsupported != "" {
			return ActionsPolicyDeclarations{}, fmt.Errorf("cannot export Actions policy id %d: %s", value.ID, value.Unsupported)
		}
		policies = append(policies, *value.Policy)
	}
	slices.SortFunc(policies, func(a, b ActionsEventPolicy) int { return strings.Compare(a.Name, b.Name) })
	return NewActionsPolicyDeclarations(policies)
}
