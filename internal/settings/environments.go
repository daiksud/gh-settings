package settings

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Environment names are identity. Numeric environment IDs are observation
// metadata only, never declaration fields or created-ID writeback input.
type EnvironmentDeclaration struct {
	Name                   string                   `json:"name" yaml:"name"`
	WaitTimer              *int                     `json:"wait_timer,omitempty" yaml:"wait_timer,omitempty"`
	PreventSelfReview      *bool                    `json:"prevent_self_review,omitempty" yaml:"prevent_self_review,omitempty"`
	Reviewers              *[]EnvironmentReviewer   `json:"reviewers,omitempty" yaml:"reviewers,omitempty"`
	DeploymentBranchPolicy *EnvironmentBranchPolicy `json:"deployment_branch_policy" yaml:"deployment_branch_policy"`
}

type EnvironmentReviewer struct {
	Type string `json:"type" yaml:"type"`
	ID   int64  `json:"id" yaml:"id"`
}

type EnvironmentBranchPolicy struct {
	ProtectedBranches    bool `json:"protected_branches" yaml:"protected_branches"`
	CustomBranchPolicies bool `json:"custom_branch_policies" yaml:"custom_branch_policies"`
}

type EnvironmentDeclarations struct{ entries []EnvironmentDeclaration }
type EnvironmentState struct {
	ID     int64
	Policy EnvironmentDeclaration
}
type EnvironmentObservation struct{ entries []EnvironmentState }
type EnvironmentChange struct {
	operation     string
	before, after *EnvironmentDeclaration
}

// Client name equivalence uses Unicode simple folding, without trimming.
func EnvironmentIdentity(name string) string {
	return strings.Map(func(r rune) rune {
		least := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < least {
				least = next
			}
		}
		return least
	}, name)
}

func validateEnvironment(policy EnvironmentDeclaration, complete bool) error {
	if policy.Name == "" || strings.TrimSpace(policy.Name) != policy.Name || !utf8.ValidString(policy.Name) || utf8.RuneCountInString(policy.Name) > 255 {
		return fmt.Errorf("environment name must contain 1–255 characters without surrounding whitespace")
	}
	for _, r := range policy.Name {
		if unicode.IsControl(r) {
			return fmt.Errorf("environment name must not contain control characters")
		}
	}
	if complete && (policy.WaitTimer == nil || policy.PreventSelfReview == nil || policy.Reviewers == nil) {
		return fmt.Errorf("environment %q requires complete timer/self-review/reviewer observation", policy.Name)
	}
	if policy.WaitTimer != nil && (*policy.WaitTimer < 0 || *policy.WaitTimer > 43200) {
		return fmt.Errorf("environment wait_timer must be between 0 and 43200 minutes")
	}
	if policy.Reviewers != nil {
		if len(*policy.Reviewers) > 6 {
			return fmt.Errorf("environment supports at most six reviewers")
		}
		seen := map[string]bool{}
		for _, reviewer := range *policy.Reviewers {
			key := fmt.Sprintf("%s:%d", reviewer.Type, reviewer.ID)
			if (reviewer.Type != "User" && reviewer.Type != "Team") || reviewer.ID <= 0 || seen[key] {
				return fmt.Errorf("invalid or duplicate environment reviewer")
			}
			seen[key] = true
		}
		if policy.PreventSelfReview != nil && *policy.PreventSelfReview && len(*policy.Reviewers) == 0 {
			return fmt.Errorf("environment prevent_self_review requires a nonempty reviewer set")
		}
	}
	if policy.DeploymentBranchPolicy != nil {
		p := policy.DeploymentBranchPolicy
		if p.ProtectedBranches == p.CustomBranchPolicies {
			return fmt.Errorf("environment deployment branch policy requires opposite booleans")
		}
		if p.CustomBranchPolicies {
			return fmt.Errorf("custom environment branch/tag policy management is not supported")
		}
	}
	return nil
}

func cloneEnvironment(policy EnvironmentDeclaration) EnvironmentDeclaration {
	policy.WaitTimer = copyRuleValue(policy.WaitTimer)
	policy.PreventSelfReview = copyRuleValue(policy.PreventSelfReview)
	policy.DeploymentBranchPolicy = copyRuleValue(policy.DeploymentBranchPolicy)
	policy.Reviewers = copyRuleList(policy.Reviewers)
	if policy.Reviewers != nil {
		slices.SortFunc(*policy.Reviewers, func(a, b EnvironmentReviewer) int {
			if order := strings.Compare(a.Type, b.Type); order != 0 {
				return order
			}
			return cmp.Compare(a.ID, b.ID)
		})
	}
	return policy
}

func NewEnvironmentDeclarations(entries []EnvironmentDeclaration) (EnvironmentDeclarations, error) {
	result := EnvironmentDeclarations{entries: make([]EnvironmentDeclaration, 0, len(entries))}
	seen := map[string]bool{}
	for _, entry := range entries {
		policy := cloneEnvironment(entry)
		defaultRuleValue(&policy.WaitTimer, 0)
		defaultRuleValue(&policy.PreventSelfReview, false)
		defaultRuleValue(&policy.Reviewers, []EnvironmentReviewer{})
		if err := validateEnvironment(policy, true); err != nil {
			return EnvironmentDeclarations{}, err
		}
		key := EnvironmentIdentity(policy.Name)
		if seen[key] {
			return EnvironmentDeclarations{}, fmt.Errorf("duplicate environment name %q", policy.Name)
		}
		seen[key] = true
		result.entries = append(result.entries, policy)
	}
	slices.SortFunc(result.entries, func(a, b EnvironmentDeclaration) int {
		return strings.Compare(EnvironmentIdentity(a.Name), EnvironmentIdentity(b.Name))
	})
	return result, nil
}

func (d EnvironmentDeclarations) Len() int { return len(d.entries) }
func (d EnvironmentDeclarations) Values() []EnvironmentDeclaration {
	out := make([]EnvironmentDeclaration, 0, len(d.entries))
	for _, p := range d.entries {
		out = append(out, cloneEnvironment(p))
	}
	return out
}

// Observation construction never applies declaration defaults to missing data.
func NewEnvironmentObservation(entries []EnvironmentState) (EnvironmentObservation, error) {
	result := EnvironmentObservation{entries: make([]EnvironmentState, 0, len(entries))}
	seenNames, seenIDs := map[string]bool{}, map[int64]bool{}
	for _, entry := range entries {
		if err := validateEnvironment(entry.Policy, true); err != nil {
			return EnvironmentObservation{}, err
		}
		key := EnvironmentIdentity(entry.Policy.Name)
		if entry.ID <= 0 || seenIDs[entry.ID] || seenNames[key] {
			return EnvironmentObservation{}, fmt.Errorf("ambiguous environment observation")
		}
		seenNames[key], seenIDs[entry.ID] = true, true
		result.entries = append(result.entries, EnvironmentState{ID: entry.ID, Policy: cloneEnvironment(entry.Policy)})
	}
	return result, nil
}

func (o EnvironmentObservation) Values() ([]EnvironmentState, error) {
	if o.entries == nil {
		return nil, fmt.Errorf("missing environment observation")
	}
	out := make([]EnvironmentState, 0, len(o.entries))
	for _, v := range o.entries {
		out = append(out, EnvironmentState{ID: v.ID, Policy: cloneEnvironment(v.Policy)})
	}
	return out, nil
}

func ReconcileEnvironments(observation EnvironmentObservation, desired *EnvironmentDeclarations) ([]EnvironmentChange, error) {
	if desired == nil || desired.Len() == 0 {
		return nil, nil
	}
	current, err := observation.Values()
	if err != nil {
		return nil, err
	}
	byName := map[string]EnvironmentDeclaration{}
	for _, state := range current {
		byName[EnvironmentIdentity(state.Policy.Name)] = state.Policy
	}
	changes := []EnvironmentChange{}
	for _, policy := range desired.Values() {
		before, exists := byName[EnvironmentIdentity(policy.Name)]
		if !exists {
			after := cloneEnvironment(policy)
			changes = append(changes, EnvironmentChange{operation: "create", after: &after})
			continue
		}
		policy.Name = before.Name
		if reflect.DeepEqual(cloneEnvironment(before), policy) {
			continue
		}
		prior, after := cloneEnvironment(before), cloneEnvironment(policy)
		changes = append(changes, EnvironmentChange{operation: "update", before: &prior, after: &after})
	}
	return changes, nil
}

func (c EnvironmentChange) Operation() string { return c.operation }
func (c EnvironmentChange) Name() string {
	if c.before != nil {
		return c.before.Name
	}
	if c.after != nil {
		return c.after.Name
	}
	return ""
}
func (c EnvironmentChange) Before() *EnvironmentDeclaration {
	if c.before == nil {
		return nil
	}
	v := cloneEnvironment(*c.before)
	return &v
}
func (c EnvironmentChange) After() *EnvironmentDeclaration {
	if c.after == nil {
		return nil
	}
	v := cloneEnvironment(*c.after)
	return &v
}

// NewEnvironmentChange validates complete same-version presentation values;
// it never supplies declaration defaults to incomplete report observations.
func NewEnvironmentChange(operation string, before, after *EnvironmentDeclaration) (EnvironmentChange, error) {
	if after == nil {
		return EnvironmentChange{}, fmt.Errorf("environment change requires after policy")
	}
	if err := validateEnvironment(*after, true); err != nil {
		return EnvironmentChange{}, err
	}
	switch operation {
	case "create":
		if before != nil {
			return EnvironmentChange{}, fmt.Errorf("environment create has before policy")
		}
	case "update":
		if before == nil {
			return EnvironmentChange{}, fmt.Errorf("environment update requires before policy")
		}
		if err := validateEnvironment(*before, true); err != nil {
			return EnvironmentChange{}, err
		}
		if before.Name != after.Name || reflect.DeepEqual(cloneEnvironment(*before), cloneEnvironment(*after)) {
			return EnvironmentChange{}, fmt.Errorf("invalid environment update identity or unchanged policy")
		}
	default:
		return EnvironmentChange{}, fmt.Errorf("unsupported environment operation %q", operation)
	}
	result := EnvironmentChange{operation: operation}
	value := cloneEnvironment(*after)
	result.after = &value
	if before != nil {
		value := cloneEnvironment(*before)
		result.before = &value
	}
	return result, nil
}
