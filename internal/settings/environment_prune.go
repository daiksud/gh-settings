package settings

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
)

// EnvironmentSecretMetadata contains only the list endpoint's metadata. Secret
// values are never acquired. This evidence never enters declarations or reports.
type EnvironmentSecretMetadata struct{ Name, CreatedAt, UpdatedAt string }

// EnvironmentPruneCandidate binds a complete deletion snapshot to its parent
// and repository. All evidence is private; only the supported policy is reportable.
type EnvironmentPruneCandidate struct {
	target    ArchiveState
	state     EnvironmentState
	variables []RepositoryVariable
	secrets   []EnvironmentSecretMetadata
	acquired  bool
}

type EnvironmentPruneObservation struct {
	target     ArchiveState
	candidates []EnvironmentPruneCandidate
}

func EnvironmentRetainedNames(d Declaration) []string {
	seen := map[string]bool{}
	names := []string{}
	if d.Environments != nil {
		for _, entry := range d.Environments.Values() {
			seen[EnvironmentIdentity(entry.Name)] = true
			names = append(names, entry.Name)
		}
	}
	for _, entry := range d.EnvironmentVariables {
		if !seen[EnvironmentIdentity(entry.Environment)] {
			seen[EnvironmentIdentity(entry.Environment)] = true
			names = append(names, entry.Environment)
		}
	}
	slices.SortFunc(names, func(a, b string) int { return strings.Compare(EnvironmentIdentity(a), EnvironmentIdentity(b)) })
	return names
}

func NewEnvironmentPruneCandidate(repo repository.Repository, target ArchiveState, state EnvironmentState, variables VariableInventory, secrets []EnvironmentSecretMetadata, customRulesKnownEmpty bool) (EnvironmentPruneCandidate, error) {
	if err := target.Validate(repo); err != nil {
		return EnvironmentPruneCandidate{}, err
	}
	if !customRulesKnownEmpty || secrets == nil {
		return EnvironmentPruneCandidate{}, fmt.Errorf("Environment deletion requires acquired custom-protection and secret metadata")
	}
	observation, err := NewEnvironmentObservation([]EnvironmentState{state})
	if err != nil {
		return EnvironmentPruneCandidate{}, err
	}
	states, _ := observation.Values()
	state = states[0]
	if state.DeploymentModeNormalization != (EnvironmentDeploymentNormalization{}) || hasEnvironmentRefTypeNormalizations(state.Patterns) {
		return EnvironmentPruneCandidate{}, fmt.Errorf("cannot prune an Environment with unsupported normalized policy")
	}
	if mode := state.Policy.DeploymentBranchPolicy; mode != nil && mode.CustomBranchPolicies {
		if state.Patterns == nil || state.Policy.DeploymentBranchPatterns == nil {
			return EnvironmentPruneCandidate{}, fmt.Errorf("Environment deletion requires complete active patterns")
		}
		changes, err := ReconcileEnvironmentPatterns(state.Patterns, state.Policy.DeploymentBranchPatterns)
		if err != nil || len(changes) != 0 {
			return EnvironmentPruneCandidate{}, fmt.Errorf("Environment deletion pattern identities are incomplete")
		}
	} else if len(state.Patterns) != 0 || state.Policy.DeploymentBranchPatterns != nil {
		return EnvironmentPruneCandidate{}, fmt.Errorf("inactive Environment patterns cannot be asserted acquired")
	}
	values, err := variables.Values()
	if err != nil {
		return EnvironmentPruneCandidate{}, fmt.Errorf("Environment deletion requires complete variable inventory")
	}
	metadata := slices.Clone(secrets)
	seen := map[string]bool{}
	for i, secret := range metadata {
		name, err := VariableName(secret.Name)
		_, createdErr := time.Parse(time.RFC3339, secret.CreatedAt)
		_, updatedErr := time.Parse(time.RFC3339, secret.UpdatedAt)
		if err != nil || createdErr != nil || updatedErr != nil || seen[name] {
			return EnvironmentPruneCandidate{}, fmt.Errorf("invalid or duplicate Environment secret metadata")
		}
		metadata[i].Name, seen[name] = name, true
	}
	slices.SortFunc(metadata, func(a, b EnvironmentSecretMetadata) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(values, func(a, b RepositoryVariable) int { return strings.Compare(a.Name, b.Name) })
	slices.SortFunc(state.Patterns, func(a, b EnvironmentPatternState) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(*state.Policy.Reviewers, func(a, b EnvironmentReviewer) int {
		if a.Type != b.Type {
			return strings.Compare(a.Type, b.Type)
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return EnvironmentPruneCandidate{target: target, state: state, variables: values, secrets: metadata, acquired: true}, nil
}

func NewEnvironmentPruneObservation(repo repository.Repository, target ArchiveState, candidates []EnvironmentPruneCandidate) (EnvironmentPruneObservation, error) {
	if err := target.Validate(repo); err != nil {
		return EnvironmentPruneObservation{}, err
	}
	entries := make([]EnvironmentPruneCandidate, 0, len(candidates))
	names, ids := map[string]bool{}, map[int64]bool{}
	for _, candidate := range candidates {
		if !candidate.acquired || !sameEnvironmentPruneTarget(candidate.target, target) || names[EnvironmentIdentity(candidate.Name())] || ids[candidate.ID()] {
			return EnvironmentPruneObservation{}, fmt.Errorf("invalid or duplicate Environment deletion candidate")
		}
		names[EnvironmentIdentity(candidate.Name())], ids[candidate.ID()] = true, true
		entries = append(entries, candidate)
	}
	slices.SortFunc(entries, func(a, b EnvironmentPruneCandidate) int {
		return strings.Compare(EnvironmentIdentity(a.Name()), EnvironmentIdentity(b.Name()))
	})
	return EnvironmentPruneObservation{target: target, candidates: entries}, nil
}

func sameEnvironmentPruneTarget(a, b ArchiveState) bool {
	return a.ID == b.ID && strings.EqualFold(a.FullName, b.FullName) && a.Archived == b.Archived
}
func (c EnvironmentPruneCandidate) Name() string { return c.state.Policy.Name }
func (c EnvironmentPruneCandidate) ID() int64    { return c.state.ID }
func (c EnvironmentPruneCandidate) ValidateTarget(repo repository.Repository, current ArchiveState) error {
	if !c.acquired || c.target.Validate(repo) != nil || current.Validate(repo) != nil || !sameEnvironmentPruneTarget(c.target, current) || current.Archived {
		return fmt.Errorf("Environment deletion target identity or archive state changed or is archived")
	}
	return nil
}
func (c EnvironmentPruneCandidate) Equal(other EnvironmentPruneCandidate) bool {
	if !sameEnvironmentPruneTarget(c.target, other.target) {
		return false
	}
	other.target.FullName = c.target.FullName
	return reflect.DeepEqual(c, other)
}
func (c EnvironmentPruneCandidate) Change() (EnvironmentChange, error) {
	if !c.acquired {
		return EnvironmentChange{}, fmt.Errorf("missing Environment deletion snapshot")
	}
	return NewEnvironmentChange("delete", &c.state.Policy, nil)
}
func (o EnvironmentPruneObservation) Candidates() ([]EnvironmentPruneCandidate, error) {
	if o.candidates == nil {
		return nil, fmt.Errorf("missing Environment prune observation")
	}
	return slices.Clone(o.candidates), nil
}

// ValidateApply rejects a known undeletable target before unrelated writes.
// Read-only plans may still describe archived targets; apply can proceed only
// with an explicit unarchive that will be independently confirmed first.
func (o EnvironmentPruneObservation) ValidateApply(unarchive bool) error {
	if o.candidates == nil {
		return fmt.Errorf("missing Environment prune observation")
	}
	if len(o.candidates) > 0 && o.target.Archived && !unarchive {
		return fmt.Errorf("Environment deletion requires an active repository or explicit archived: false")
	}
	return nil
}

// AfterUnarchive accounts only for the caller's confirmed repository transition.
func (o EnvironmentPruneObservation) AfterUnarchive() EnvironmentPruneObservation {
	o.candidates = slices.Clone(o.candidates)
	o.target.Archived = false
	for i := range o.candidates {
		o.candidates[i].target.Archived = false
	}
	return o
}
func (o EnvironmentPruneObservation) Equal(other EnvironmentPruneObservation) bool {
	if o.candidates == nil || other.candidates == nil || !sameEnvironmentPruneTarget(o.target, other.target) || len(o.candidates) != len(other.candidates) {
		return false
	}
	for i := range o.candidates {
		if !o.candidates[i].Equal(other.candidates[i]) {
			return false
		}
	}
	return true
}
