package settings

import (
	"cmp"
	"fmt"
	"slices"
	"time"
)

// WebhookPruneState is destructive evidence, not a public projection. Timestamps
// are comparison tokens only: equality cannot establish Secret equality or a lock.
type WebhookPruneState struct {
	State     WebhookState
	UpdatedAt string
}

func (s WebhookPruneState) validate() error {
	if err := s.State.Validate(); err != nil {
		return err
	}
	for _, value := range []string{s.State.CreatedAt, s.UpdatedAt} {
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return fmt.Errorf("incomplete webhook prune timestamp")
		}
	}
	return nil
}
func (s WebhookPruneState) clone() WebhookPruneState { s.State = s.State.Clone(); return s }
func webhookPruneStateEqual(a, b WebhookPruneState) bool {
	return a.UpdatedAt == b.UpdatedAt && WebhookPreserved(a.State, b.State, Webhook{})
}

// WebhookPruneObservation retains a complete private inventory and a fixed ID set.
// Neither Secret nor transport/delivery metadata enters this model.
type WebhookPruneObservation struct {
	target     WebhookTarget
	states     []WebhookPruneState
	candidates []int64
}

func NewWebhookPruneObservation(target WebhookTarget, states []WebhookPruneState) (WebhookPruneObservation, error) {
	o := WebhookPruneObservation{target: target, states: states}
	if err := o.Validate(); err != nil {
		return WebhookPruneObservation{}, err
	}
	return o.Clone(), nil
}
func (o WebhookPruneObservation) Validate() error {
	if err := o.target.Validate(); err != nil {
		return err
	}
	if o.states == nil {
		return fmt.Errorf("missing complete webhook prune inventory")
	}
	seen := map[int64]bool{}
	for _, s := range o.states {
		if err := s.validate(); err != nil {
			return err
		}
		if seen[s.State.ID] {
			return fmt.Errorf("duplicate webhook prune identity")
		}
		seen[s.State.ID] = true
	}
	candidates := map[int64]bool{}
	for _, id := range o.candidates {
		if !seen[id] || candidates[id] {
			return fmt.Errorf("invalid fixed webhook prune candidate")
		}
		candidates[id] = true
	}
	return nil
}
func (o WebhookPruneObservation) Clone() WebhookPruneObservation {
	o.states = slices.Clone(o.states)
	for i := range o.states {
		o.states[i] = o.states[i].clone()
	}
	o.candidates = slices.Clone(o.candidates)
	return o
}
func (o WebhookPruneObservation) Inventory() WebhookObservation {
	values := make([]WebhookState, 0, len(o.states))
	for _, s := range o.states {
		values = append(values, s.State.Clone())
	}
	return WebhookObservation{Complete: true, Target: o.target, Values: values}
}
func (o WebhookPruneObservation) Find(id int64) (WebhookPruneState, bool) {
	for _, s := range o.states {
		if s.State.ID == id {
			return s.clone(), true
		}
	}
	return WebhookPruneState{}, false
}
func PrepareWebhookPrune(o WebhookPruneObservation, desired []Webhook) (WebhookPruneObservation, error) {
	if err := o.Validate(); err != nil {
		return WebhookPruneObservation{}, err
	}
	if _, _, err := PrepareWebhooks(o.Inventory(), desired); err != nil {
		return WebhookPruneObservation{}, err
	}
	keep := map[int64]bool{}
	for _, d := range desired {
		if d.ID > 0 {
			keep[d.ID] = true
		}
	}
	o = o.Clone()
	o.candidates = []int64{}
	for _, s := range o.states {
		if !keep[s.State.ID] {
			o.candidates = append(o.candidates, s.State.ID)
		}
	}
	slices.Sort(o.candidates)
	return o, nil
}
func (o WebhookPruneObservation) Candidates() ([]int64, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	if o.candidates == nil {
		return nil, fmt.Errorf("webhook prune selection is unavailable")
	}
	return slices.Clone(o.candidates), nil
}
func (o WebhookPruneObservation) ValidateApply(unarchive bool) error {
	ids, err := o.Candidates()
	if err != nil {
		return err
	}
	if len(ids) > 0 && o.target.Archived && !unarchive {
		return fmt.Errorf("webhook deletion requires an active repository or explicit unarchive")
	}
	return nil
}
func (o WebhookPruneObservation) AfterUnarchive() WebhookPruneObservation {
	o = o.Clone()
	o.target.Archived = false
	return o
}
func (o WebhookPruneObservation) Verify(fresh WebhookPruneObservation) error {
	if o.Validate() != nil || fresh.Validate() != nil || o.target != fresh.target || len(o.states) != len(fresh.states) {
		return fmt.Errorf("webhook prune target or complete inventory changed")
	}
	for _, s := range o.states {
		actual, found := fresh.Find(s.State.ID)
		if !found || !webhookPruneStateEqual(s, actual) {
			return fmt.Errorf("webhook prune identity, nonsecret state or timestamp changed")
		}
	}
	return nil
}
func (o WebhookPruneObservation) VerifyDelete(id int64, fresh WebhookPruneObservation) error {
	ids, err := o.Candidates()
	if err != nil || !slices.Contains(ids, id) || o.target.Archived {
		return fmt.Errorf("webhook deletion requires an acquired candidate on an active target")
	}
	return o.Verify(fresh)
}
func (o WebhookPruneObservation) Without(id int64) (WebhookPruneObservation, error) {
	if err := o.VerifyDelete(id, o); err != nil {
		return WebhookPruneObservation{}, err
	}
	o = o.Clone()
	o.states = slices.DeleteFunc(o.states, func(s WebhookPruneState) bool { return s.State.ID == id })
	o.candidates = slices.DeleteFunc(o.candidates, func(candidate int64) bool { return candidate == id })
	return o, nil
}

// WithUpdated accepts fresh evidence only for the independently verified write's
// object. Every retained/candidate neighbor still must match the original evidence.
func (o WebhookPruneObservation) WithUpdated(desired Webhook, fresh WebhookPruneObservation) (WebhookPruneObservation, error) {
	before, found := o.Find(desired.ID)
	after, present := fresh.Find(desired.ID)
	if o.Validate() != nil || o.candidates == nil || slices.Contains(o.candidates, desired.ID) || fresh.Validate() != nil || desired.Validate() != nil || !found || !present || !WebhookPreserved(before.State, after.State, desired) {
		return WebhookPruneObservation{}, fmt.Errorf("verified webhook update evidence does not match")
	}
	o = o.Clone()
	for i := range o.states {
		if o.states[i].State.ID == desired.ID {
			o.states[i] = after.clone()
		}
	}
	if err := o.Verify(fresh); err != nil {
		return WebhookPruneObservation{}, err
	}
	return o, nil
}
func (o WebhookPruneObservation) WithCreated(created WebhookState, fresh WebhookPruneObservation) (WebhookPruneObservation, error) {
	_, exists := o.Find(created.ID)
	actual, found := fresh.Find(created.ID)
	if o.Validate() != nil || o.candidates == nil || fresh.Validate() != nil || created.Validate() != nil || exists || !found || !WebhookPreserved(created, actual.State, Webhook{}) {
		return WebhookPruneObservation{}, fmt.Errorf("verified webhook creation evidence does not match")
	}
	o = o.Clone()
	o.states = append(o.states, actual.clone())
	slices.SortFunc(o.states, func(a, b WebhookPruneState) int { return cmp.Compare(a.State.ID, b.State.ID) })
	if err := o.Verify(fresh); err != nil {
		return WebhookPruneObservation{}, err
	}
	return o, nil
}

func (o WebhookPruneObservation) VerifyCandidate(id int64, state WebhookPruneState, target WebhookTarget) error {
	before, found := o.Find(id)
	if err := o.VerifyDelete(id, o); err != nil {
		return err
	}
	if !found || state.validate() != nil || target != o.target || !webhookPruneStateEqual(before, state) {
		return fmt.Errorf("webhook deletion candidate or target changed")
	}
	return nil
}
