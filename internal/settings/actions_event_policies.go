package settings

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ActionsEventPolicy owns one repository policy with actor and/or event restrictions.
// ID zero requests creation; positive IDs select existing repository objects.
type ActionsEventPolicy struct {
	ID          int64
	Name        string
	Enforcement string
	// Nil rules preserve the observed restriction on update. An empty actor list
	// enables an allowlist containing no actors; an empty event list is invalid.
	Actors          *[]ActionsPolicyActor
	DisableActors   bool
	DisableEvents   bool
	Events          []string
	TargetWorkflows *ActionsWorkflowTarget
}

// ActionsPolicyActor is the complete identity of an allowed actor.
type ActionsPolicyActor struct {
	Type string
	ID   int64
}

// ActionsWorkflowTarget selects workflow paths. Nil on a declaration preserves an
// existing selection, or uses All workflows when creating a policy.
type ActionsWorkflowTarget struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

func ValidateActionsWorkflowTarget(target *ActionsWorkflowTarget) error {
	if target == nil {
		return nil
	}
	if len(target.Include)+len(target.Exclude) == 0 {
		return fmt.Errorf("target_workflows requires at least one include or exclude pattern; use include: [~ALL] for All workflows")
	}
	for _, list := range [][]string{target.Include, target.Exclude} {
		seen := map[string]bool{}
		for _, pattern := range list {
			if strings.TrimSpace(pattern) == "" || !utf8.ValidString(pattern) {
				return fmt.Errorf("workflow pattern must be nonempty")
			}
			for _, r := range pattern {
				if unicode.IsControl(r) {
					return fmt.Errorf("workflow pattern must not contain control characters")
				}
			}
			if seen[pattern] {
				return fmt.Errorf("duplicate workflow pattern %q", pattern)
			}
			seen[pattern] = true
		}
	}
	if slices.Contains(target.Exclude, "~ALL") || (slices.Contains(target.Include, "~ALL") && len(target.Include) != 1) {
		return fmt.Errorf("~ALL is allowed only alone in workflow include")
	}
	return nil
}

// CanonicalActionsWorkflowTarget returns an owned, order-independent selection.
// Empty includes and ~ALL both mean every workflow except the exclusions.
func CanonicalActionsWorkflowTarget(target *ActionsWorkflowTarget) *ActionsWorkflowTarget {
	if target == nil {
		return nil
	}
	result := &ActionsWorkflowTarget{Include: append([]string{}, target.Include...), Exclude: append([]string{}, target.Exclude...)}
	if len(result.Exclude) == 0 && len(result.Include) == 0 {
		result.Include = []string{"~ALL"}
	}
	if len(result.Exclude) > 0 && len(result.Include) == 1 && result.Include[0] == "~ALL" {
		result.Include = []string{}
	}
	slices.Sort(result.Include)
	slices.Sort(result.Exclude)
	return result
}
func EqualActionsWorkflowTargets(a, b *ActionsWorkflowTarget) bool {
	if a == nil {
		a = &ActionsWorkflowTarget{}
	}
	if b == nil {
		b = &ActionsWorkflowTarget{}
	}
	a, b = CanonicalActionsWorkflowTarget(a), CanonicalActionsWorkflowTarget(b)
	return slices.Equal(a.Include, b.Include) && slices.Equal(a.Exclude, b.Exclude)
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

// ValidateActionsEventPolicy validates a declaration, including explicit rule removal.
func ValidateActionsEventPolicy(policy ActionsEventPolicy) error {
	if err := ValidateActionsWorkflowTarget(policy.TargetWorkflows); err != nil {
		return err
	}
	if policy.ID < 0 {
		return fmt.Errorf("Actions policy id must be positive when supplied")
	}
	if err := validateActionsPolicyName(policy.Name); err != nil {
		return err
	}
	if policy.Enforcement != "disabled" && policy.Enforcement != "active" {
		return fmt.Errorf("Actions policy %q enforcement must be disabled or active", policy.Name)
	}
	if policy.DisableActors && policy.DisableEvents {
		return fmt.Errorf("Actions policy %q cannot disable every restriction", policy.Name)
	}
	if policy.DisableActors && policy.Actors != nil {
		return fmt.Errorf("Actions policy %q cannot disable restrict_actors and supply allowed actors", policy.Name)
	}
	if policy.DisableEvents && policy.Events != nil {
		return fmt.Errorf("Actions policy %q cannot disable restrict_events and supply allowed events", policy.Name)
	}
	if policy.ID == 0 && policy.Actors == nil && policy.Events == nil {
		return fmt.Errorf("Actions policy %q creation requires at least one active restriction", policy.Name)
	}
	if policy.Actors != nil {
		seen := map[ActionsPolicyActor]bool{}
		for _, actor := range *policy.Actors {
			switch actor.Type {
			case "User":
				if actor.ID <= 0 {
					return fmt.Errorf("Actions policy %q User actor id must be positive", policy.Name)
				}
			case "RepositoryRole":
				if actor.ID != 2 && actor.ID != 4 && actor.ID != 5 {
					return fmt.Errorf("Actions policy %q has unsupported RepositoryRole actor id %d", policy.Name, actor.ID)
				}
			default:
				return fmt.Errorf("Actions policy %q has unsupported actor type %q", policy.Name, actor.Type)
			}
			if seen[actor] {
				return fmt.Errorf("Actions policy %q has duplicate actor %s/%d", policy.Name, actor.Type, actor.ID)
			}
			seen[actor] = true
		}
	}
	if policy.Events != nil && len(policy.Events) == 0 {
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

// ValidateActionsEventPolicyState validates a resolved policy, not update intent.
func ValidateActionsEventPolicyState(policy ActionsEventPolicy) error {
	if policy.DisableActors || policy.DisableEvents {
		return fmt.Errorf("Actions policy %q effective state must not contain disable intent", policy.Name)
	}
	if err := ValidateActionsEventPolicy(policy); err != nil {
		return err
	}
	if policy.Actors == nil && policy.Events == nil {
		return fmt.Errorf("Actions policy %q requires at least one active restriction", policy.Name)
	}
	return nil
}

func cloneActionsEventPolicy(policy ActionsEventPolicy) ActionsEventPolicy {
	if policy.Actors != nil {
		actors := slices.Clone(*policy.Actors)
		policy.Actors = &actors
	}
	policy.TargetWorkflows = CanonicalActionsWorkflowTarget(policy.TargetWorkflows)
	policy.Events = slices.Clone(policy.Events)
	return policy
}
func canonicalActionsEventPolicy(policy ActionsEventPolicy) ActionsEventPolicy {
	policy = cloneActionsEventPolicy(policy)
	slices.Sort(policy.Events)
	if policy.Actors != nil {
		slices.SortFunc(*policy.Actors, func(a, b ActionsPolicyActor) int {
			if order := strings.Compare(a.Type, b.Type); order != 0 {
				return order
			}
			if a.ID < b.ID {
				return -1
			}
			if a.ID > b.ID {
				return 1
			}
			return 0
		})
	}
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
			if err := ValidateActionsEventPolicyState(*value.Policy); err != nil {
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

// resolveActionsEventPolicy preserves omitted rules and consumes removal intent.
func resolveActionsEventPolicy(before *ActionsEventPolicy, after ActionsEventPolicy) ActionsEventPolicy {
	if before != nil {
		if after.TargetWorkflows == nil {
			after.TargetWorkflows = before.TargetWorkflows
		}
		if after.Actors == nil && !after.DisableActors {
			after.Actors = before.Actors
		}
		if after.Events == nil && !after.DisableEvents {
			after.Events = before.Events
		}
	}
	if after.DisableActors {
		after.Actors = nil
	}
	if after.DisableEvents {
		after.Events = nil
	}
	after.DisableActors, after.DisableEvents = false, false
	return canonicalActionsEventPolicy(after)
}

func NewActionsEventPolicyChange(before *ActionsEventPolicy, after ActionsEventPolicy) (ActionsEventPolicyChange, error) {
	if err := ValidateActionsEventPolicy(after); err != nil {
		return ActionsEventPolicyChange{}, err
	}
	if before != nil {
		if err := ValidateActionsEventPolicyState(*before); err != nil {
			return ActionsEventPolicyChange{}, err
		}
	}
	after = resolveActionsEventPolicy(before, after)
	if err := ValidateActionsEventPolicyState(after); err != nil {
		return ActionsEventPolicyChange{}, err
	}
	if before == nil {
		if after.ID != 0 {
			return ActionsEventPolicyChange{}, fmt.Errorf("Actions policy creation must omit id")
		}
		return ActionsEventPolicyChange{after: after}, nil
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

// NewActionsEventPolicyStateChange reconstructs a change from complete resolved
// states. Unlike a declaration, an absent rule in after means removal.
func NewActionsEventPolicyStateChange(before *ActionsEventPolicy, after ActionsEventPolicy) (ActionsEventPolicyChange, error) {
	if err := ValidateActionsEventPolicyState(after); err != nil {
		return ActionsEventPolicyChange{}, err
	}
	if before != nil {
		if err := ValidateActionsEventPolicyState(*before); err != nil {
			return ActionsEventPolicyChange{}, err
		}
		after.DisableActors = before.Actors != nil && after.Actors == nil
		after.DisableEvents = before.Events != nil && after.Events == nil
	}
	// Nil targeting in a complete state means All workflows, not omission.
	if after.TargetWorkflows == nil {
		after.TargetWorkflows = CanonicalActionsWorkflowTarget(&ActionsWorkflowTarget{})
	}
	return NewActionsEventPolicyChange(before, after)
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
	actorsEqual := a.Actors == nil && b.Actors == nil
	if a.Actors != nil && b.Actors != nil {
		actorsEqual = slices.Equal(*a.Actors, *b.Actors)
	}
	return a.ID == b.ID && a.Name == b.Name && a.Enforcement == b.Enforcement &&
		actorsEqual && (a.Events == nil) == (b.Events == nil) && slices.Equal(a.Events, b.Events) &&
		a.DisableActors == b.DisableActors && a.DisableEvents == b.DisableEvents &&
		EqualActionsWorkflowTargets(a.TargetWorkflows, b.TargetWorkflows)
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
			resolved := resolveActionsEventPolicy(before, after)
			if err := ValidateActionsEventPolicyState(resolved); err != nil {
				return nil, err
			}
			if equalActionsEventPolicies(*before, resolved) {
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
		policy := cloneActionsEventPolicy(*value.Policy)
		if policy.TargetWorkflows == nil {
			policy.TargetWorkflows = CanonicalActionsWorkflowTarget(&ActionsWorkflowTarget{})
		}
		policies = append(policies, policy)
	}
	slices.SortFunc(policies, func(a, b ActionsEventPolicy) int { return strings.Compare(a.Name, b.Name) })
	return NewActionsPolicyDeclarations(policies)
}
