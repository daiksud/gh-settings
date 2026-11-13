package settings

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
)

// ClassicPruneCandidate is private acquired evidence, never a report or apply input.
type ClassicPruneCandidate struct {
	repositoryID string
	archived     bool
	state        ClassicState
	acquired     bool
}

type ClassicPruneObservation struct {
	repositoryID string
	archived     bool
	candidates   []ClassicPruneCandidate
}

func cloneClassicState(state ClassicState) ClassicState {
	state.Rule = CloneClassicRule(state.Rule)
	c := &state.Companions
	c.ReviewCount.Value = cloneClassicValue(c.ReviewCount.Value)
	for _, v := range []*ClassicNullable[[]string]{&c.Deployments, &c.Contexts} {
		if v.Value != nil {
			values := slices.Clone(*v.Value)
			v.Value = &values
		}
	}
	if c.Checks.Value != nil {
		values := slices.Clone(*c.Checks.Value)
		for i := range values {
			values[i].AppID = cloneClassicValue(values[i].AppID)
		}
		c.Checks.Value = &values
	}
	c.PushActors = slices.Clone(c.PushActors)
	c.ReviewDismissalActors = slices.Clone(c.ReviewDismissalActors)
	c.ForcePushActors = slices.Clone(c.ForcePushActors)
	c.PullRequestActors = slices.Clone(c.PullRequestActors)
	return state
}

func PrepareClassicPrune(observed ClassicObservation, desired []ClassicRule) (ClassicPruneObservation, error) {
	if _, err := PrepareClassicRules(observed, desired); err != nil {
		return ClassicPruneObservation{}, err
	}
	retained := map[int64]bool{}
	for _, rule := range desired {
		if rule.ID != nil {
			retained[*rule.ID] = true
		}
	}
	result := ClassicPruneObservation{repositoryID: observed.RepositoryID, archived: observed.Archived, candidates: []ClassicPruneCandidate{}}
	for _, state := range observed.Rules {
		if !retained[*state.Rule.ID] {
			result.candidates = append(result.candidates, ClassicPruneCandidate{repositoryID: observed.RepositoryID, archived: observed.Archived, state: cloneClassicState(state), acquired: true})
		}
	}
	slices.SortFunc(result.candidates, func(a, b ClassicPruneCandidate) int { return cmp.Compare(a.ID(), b.ID()) })
	return result, nil
}

func (o ClassicPruneObservation) Candidates() ([]ClassicPruneCandidate, error) {
	if o.repositoryID == "" || o.candidates == nil {
		return nil, fmt.Errorf("missing classic prune acquisition")
	}
	result := slices.Clone(o.candidates)
	for i := range result {
		result[i].state = cloneClassicState(result[i].state)
	}
	return result, nil
}

func (o ClassicPruneObservation) ValidateApply(unarchive bool) error {
	if _, err := o.Candidates(); err != nil {
		return err
	}
	if len(o.candidates) > 0 && o.archived && !unarchive {
		return fmt.Errorf("classic rule deletion requires an active repository or explicit archived: false")
	}
	return nil
}

// AfterUnarchive accounts only for the caller's independently confirmed transition.
func (o ClassicPruneObservation) AfterUnarchive() ClassicPruneObservation {
	o.candidates = slices.Clone(o.candidates)
	o.archived = false
	for i := range o.candidates {
		o.candidates[i].archived = false
	}
	return o
}

// Recheck never adopts newly appeared unlisted rules into the frozen selection.
func (o ClassicPruneObservation) Recheck(fresh ClassicObservation) error {
	if _, err := o.Candidates(); err != nil {
		return err
	}
	if err := fresh.Validate(); err != nil {
		return err
	}
	if fresh.RepositoryID != o.repositoryID || fresh.Archived != o.archived {
		return fmt.Errorf("classic prune target identity or archive state changed")
	}
	for _, candidate := range o.candidates {
		found := false
		for _, state := range fresh.Rules {
			if *state.Rule.ID == candidate.ID() || state.NodeID == candidate.NodeID() {
				if err := candidate.Verify(state, fresh.RepositoryID, fresh.Archived); err != nil {
					return err
				}
				found = true
			}
		}
		if !found {
			return fmt.Errorf("classic prune candidate disappeared")
		}
	}
	return nil
}

func (c ClassicPruneCandidate) ID() int64 {
	if c.state.Rule.ID == nil {
		return 0
	}
	return *c.state.Rule.ID
}
func (c ClassicPruneCandidate) NodeID() string  { return c.state.NodeID }
func (c ClassicPruneCandidate) Pattern() string { return c.state.Rule.BranchNamePattern }
func (c ClassicPruneCandidate) VerifyTarget(repositoryID string, archived bool) error {
	if !c.acquired || c.repositoryID == "" || c.archived || archived || repositoryID != c.repositoryID {
		return fmt.Errorf("classic deletion target unavailable, changed or archived")
	}
	return nil
}
func (c ClassicPruneCandidate) Verify(state ClassicState, repositoryID string, archived bool) error {
	if err := c.VerifyTarget(repositoryID, archived); err != nil {
		return err
	}
	if err := (ClassicObservation{RepositoryID: repositoryID, Archived: archived, Rules: []ClassicState{state}}).Validate(); err != nil {
		return err
	}
	if !reflect.DeepEqual(c.state, state) {
		return fmt.Errorf("classic deletion identity or raw policy changed")
	}
	return nil
}
func (c ClassicPruneCandidate) Change() (ClassicChange, error) {
	if !c.acquired {
		return ClassicChange{}, fmt.Errorf("missing classic deletion snapshot")
	}
	before := CloneClassicRule(c.state.Rule)
	return ClassicChange{Before: &before}, nil
}

func CompareClassicRulesWithPrune(observed ClassicObservation, desired []ClassicRule, prune bool) ([]ClassicChange, error) {
	changes, err := CompareClassicRules(observed, desired)
	if err != nil || !prune {
		return changes, err
	}
	prepared, err := PrepareClassicPrune(observed, desired)
	if err != nil {
		return nil, err
	}
	for _, candidate := range prepared.candidates {
		change, err := candidate.Change()
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return changes, nil
}
