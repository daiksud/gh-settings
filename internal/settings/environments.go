package settings

import (
	"bytes"
	"cmp"
	"encoding/json"
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
	Name                           string                   `json:"name" yaml:"name"`
	WaitTimer                      *int                     `json:"wait_timer,omitempty" yaml:"wait_timer,omitempty"`
	PreventSelfReview              *bool                    `json:"prevent_self_review,omitempty" yaml:"prevent_self_review,omitempty"`
	CanAdminsBypass                *bool                    `json:"can_admins_bypass,omitempty" yaml:"can_admins_bypass,omitempty"`
	Reviewers                      *[]EnvironmentReviewer   `json:"reviewers,omitempty" yaml:"reviewers,omitempty"`
	DeploymentBranchPolicy         *EnvironmentBranchPolicy `json:"deployment_branch_policy" yaml:"deployment_branch_policy"`
	DeploymentBranchPolicyDeclared bool                     `json:"-" yaml:"-"`
	DeploymentBranchPatterns       *[]EnvironmentPattern    `json:"deployment_branch_patterns,omitempty" yaml:"deployment_branch_patterns,omitempty"`
}

// Keep explicit null distinct from an omitted policy, without supplying API
// defaults. Reports, PUT bodies and YAML all use the same presence semantics.
func (p EnvironmentDeclaration) fields() map[string]any {
	fields := map[string]any{"name": p.Name}
	if p.WaitTimer != nil {
		fields["wait_timer"] = *p.WaitTimer
	}
	if p.PreventSelfReview != nil {
		fields["prevent_self_review"] = *p.PreventSelfReview
	}
	if p.CanAdminsBypass != nil {
		fields["can_admins_bypass"] = *p.CanAdminsBypass
	}
	if p.Reviewers != nil {
		fields["reviewers"] = *p.Reviewers
	}
	if p.DeploymentBranchPolicyDeclared || p.DeploymentBranchPolicy != nil {
		fields["deployment_branch_policy"] = p.DeploymentBranchPolicy
	}
	if p.DeploymentBranchPatterns != nil {
		fields["deployment_branch_patterns"] = *p.DeploymentBranchPatterns
	}
	return fields
}
func (p EnvironmentDeclaration) MarshalJSON() ([]byte, error) { return json.Marshal(p.fields()) }
func (p EnvironmentDeclaration) MarshalYAML() (any, error)    { return p.fields(), nil }
func (p *EnvironmentDeclaration) UnmarshalJSON(data []byte) error {
	type plain EnvironmentDeclaration
	var value plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	_, value.DeploymentBranchPolicyDeclared = fields["deployment_branch_policy"]
	*p = EnvironmentDeclaration(value)
	return nil
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
	ID                          int64
	Policy                      EnvironmentDeclaration
	Patterns                    []EnvironmentPatternState
	DeploymentModeNormalization EnvironmentDeploymentNormalization
}
type EnvironmentObservation struct{ entries []EnvironmentState }
type EnvironmentChange struct {
	operation                   string
	before, after               *EnvironmentDeclaration
	deploymentModeNormalization EnvironmentDeploymentNormalization
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

// ValidateObservationPolicy validates complete policy values without asserting acquisition or resource identity.
func (policy EnvironmentDeclaration) ValidateObservationPolicy() error {
	return validateEnvironment(policy, true)
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
	if complete && (policy.WaitTimer == nil || policy.PreventSelfReview == nil || policy.CanAdminsBypass == nil || policy.Reviewers == nil || (!policy.DeploymentBranchPolicyDeclared && policy.DeploymentBranchPolicy == nil)) {
		return fmt.Errorf("environment %q requires complete timer/self-review/admin-bypass/reviewer/branch-policy observation", policy.Name)
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
	}
	if policy.DeploymentBranchPatterns != nil {
		if _, err := normalizeEnvironmentPatterns(*policy.DeploymentBranchPatterns); err != nil {
			return err
		}
		if policy.DeploymentBranchPolicyDeclared || policy.DeploymentBranchPolicy != nil {
			if policy.DeploymentBranchPolicy == nil || !policy.DeploymentBranchPolicy.CustomBranchPolicies {
				return fmt.Errorf("environment deployment patterns require custom branch policy mode")
			}
		}
	}
	return nil
}

func cloneEnvironment(policy EnvironmentDeclaration) EnvironmentDeclaration {
	policy.WaitTimer = copyRuleValue(policy.WaitTimer)
	policy.PreventSelfReview = copyRuleValue(policy.PreventSelfReview)
	policy.CanAdminsBypass = copyRuleValue(policy.CanAdminsBypass)
	policy.DeploymentBranchPolicy = copyRuleValue(policy.DeploymentBranchPolicy)
	if policy.DeploymentBranchPolicy != nil {
		policy.DeploymentBranchPolicyDeclared = true
	}
	policy.DeploymentBranchPatterns = copyRuleList(policy.DeploymentBranchPatterns)
	if policy.DeploymentBranchPatterns != nil {
		if patterns, err := normalizeEnvironmentPatterns(*policy.DeploymentBranchPatterns); err == nil {
			policy.DeploymentBranchPatterns = &patterns
		}
	}
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
		if err := validateEnvironment(policy, false); err != nil {
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
		if err := entry.validateDeploymentNormalization(); err != nil {
			return EnvironmentObservation{}, err
		}
		key := EnvironmentIdentity(entry.Policy.Name)
		if entry.ID <= 0 || seenIDs[entry.ID] || seenNames[key] {
			return EnvironmentObservation{}, fmt.Errorf("ambiguous environment observation")
		}
		seenNames[key], seenIDs[entry.ID] = true, true
		result.entries = append(result.entries, EnvironmentState{ID: entry.ID, Policy: cloneEnvironment(entry.Policy), Patterns: slices.Clone(entry.Patterns), DeploymentModeNormalization: entry.DeploymentModeNormalization})
	}
	return result, nil
}

func (o EnvironmentObservation) Values() ([]EnvironmentState, error) {
	if o.entries == nil {
		return nil, fmt.Errorf("missing environment observation")
	}
	out := make([]EnvironmentState, 0, len(o.entries))
	for _, v := range o.entries {
		out = append(out, EnvironmentState{ID: v.ID, Policy: cloneEnvironment(v.Policy), Patterns: slices.Clone(v.Patterns), DeploymentModeNormalization: v.DeploymentModeNormalization})
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
	byName := map[string]EnvironmentState{}
	for _, state := range current {
		byName[EnvironmentIdentity(state.Policy.Name)] = state
	}
	changes := []EnvironmentChange{}
	for _, policy := range desired.Values() {
		state, exists := byName[EnvironmentIdentity(policy.Name)]
		before := state.Policy
		if policy.DeploymentBranchPatterns != nil && policy.DeploymentBranchPolicy == nil && !policy.DeploymentBranchPolicyDeclared {
			if !exists || before.DeploymentBranchPolicy == nil || !before.DeploymentBranchPolicy.CustomBranchPolicies {
				return nil, fmt.Errorf("environment %q deployment patterns require explicit custom mode or an existing custom-mode environment", policy.Name)
			}
		}
		if !exists {
			after := cloneEnvironment(policy)
			changes = append(changes, EnvironmentChange{operation: "create", after: &after})
			continue
		}
		policy.Name = before.Name
		record := EnvironmentDeploymentNormalization{}
		if policy.DeploymentBranchPolicyDeclared || policy.DeploymentBranchPolicy != nil {
			record = state.DeploymentModeNormalization
		}
		if record == (EnvironmentDeploymentNormalization{}) && !EnvironmentHasAPIOmissions(policy) && EnvironmentMatchesRequest(before, policy) {
			continue
		}
		prior, after := cloneEnvironment(before), cloneEnvironment(policy)
		var change EnvironmentChange
		var err error
		if record != (EnvironmentDeploymentNormalization{}) {
			change, err = NewEnvironmentChangeWithDeploymentNormalization("update", &prior, &after, record)
		} else {
			change, err = NewEnvironmentChange("update", &prior, &after)
		}
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
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

// NewEnvironmentChange requires a complete observed before policy, but after
// is the actual request, whose omissions intentionally defer to the API.
func NewEnvironmentChange(operation string, before, after *EnvironmentDeclaration) (EnvironmentChange, error) {
	return newEnvironmentChange(operation, before, after, EnvironmentDeploymentNormalization{})
}

func newEnvironmentChange(operation string, before, after *EnvironmentDeclaration, record EnvironmentDeploymentNormalization) (EnvironmentChange, error) {
	if after == nil {
		return EnvironmentChange{}, fmt.Errorf("environment change requires after policy")
	}
	if err := validateEnvironment(*after, false); err != nil {
		return EnvironmentChange{}, err
	}
	if record != (EnvironmentDeploymentNormalization{}) {
		if operation != "update" || before == nil || (!after.DeploymentBranchPolicyDeclared && after.DeploymentBranchPolicy == nil) {
			return EnvironmentChange{}, fmt.Errorf("normalized environment mode requires one explicit update request")
		}
		if err := validateEnvironmentDeploymentRecord(*before, record); err != nil {
			return EnvironmentChange{}, err
		}
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
		if before.Name != after.Name || (record == (EnvironmentDeploymentNormalization{}) && reflect.DeepEqual(cloneEnvironment(*before), cloneEnvironment(*after))) {
			return EnvironmentChange{}, fmt.Errorf("invalid environment update identity or unchanged policy")
		}
	default:
		return EnvironmentChange{}, fmt.Errorf("unsupported environment operation %q", operation)
	}
	result := EnvironmentChange{operation: operation, deploymentModeNormalization: record}
	value := cloneEnvironment(*after)
	result.after = &value
	if before != nil {
		value := cloneEnvironment(*before)
		result.before = &value
	}
	return result, nil
}

// Omitted fields are sent as omissions and their server-selected result is not
// predicted by plan. Such declarations remain update intents, not verified no-ops.
func EnvironmentHasAPIOmissions(p EnvironmentDeclaration) bool {
	return p.WaitTimer == nil || p.PreventSelfReview == nil || p.CanAdminsBypass == nil || p.Reviewers == nil || (!p.DeploymentBranchPolicyDeclared && p.DeploymentBranchPolicy == nil)
}

// Verify only requested values. The full readback is still typed and validated;
// unspecified API outcomes are neither copied from old state nor invented.
func EnvironmentMatchesRequest(actual, requested EnvironmentDeclaration) bool {
	a, r := cloneEnvironment(actual), cloneEnvironment(requested)
	return EnvironmentIdentity(a.Name) == EnvironmentIdentity(r.Name) &&
		(r.WaitTimer == nil || reflect.DeepEqual(a.WaitTimer, r.WaitTimer)) &&
		(r.PreventSelfReview == nil || reflect.DeepEqual(a.PreventSelfReview, r.PreventSelfReview)) &&
		(r.CanAdminsBypass == nil || reflect.DeepEqual(a.CanAdminsBypass, r.CanAdminsBypass)) &&
		(r.Reviewers == nil || reflect.DeepEqual(a.Reviewers, r.Reviewers)) &&
		(!r.DeploymentBranchPolicyDeclared || reflect.DeepEqual(a.DeploymentBranchPolicy, r.DeploymentBranchPolicy)) &&
		(r.DeploymentBranchPatterns == nil || reflect.DeepEqual(a.DeploymentBranchPatterns, r.DeploymentBranchPatterns))
}
