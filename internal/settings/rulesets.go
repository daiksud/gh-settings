package settings

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RulesetDeclaration retains presence at every writable boundary.
type RulesetDeclaration struct {
	Name         string             `json:"name" yaml:"name"`
	Target       *string            `json:"target,omitempty" yaml:"target,omitempty"`
	Enforcement  *string            `json:"enforcement,omitempty" yaml:"enforcement,omitempty"`
	Conditions   *RulesetConditions `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	BypassActors *[]BypassActor     `json:"bypass_actors,omitempty" yaml:"bypass_actors,omitempty"`
	Rules        *[]RulesetRule     `json:"rules,omitempty" yaml:"rules,omitempty"`
}
type RulesetConditions struct {
	RefName *RefNameCondition `json:"ref_name,omitempty" yaml:"ref_name,omitempty"`
}
type RefNameCondition struct {
	Include *[]string `json:"include,omitempty" yaml:"include,omitempty"`
	Exclude *[]string `json:"exclude,omitempty" yaml:"exclude,omitempty"`
}
type BypassActor struct {
	ActorType  string  `json:"actor_type" yaml:"actor_type"`
	ActorID    *int64  `json:"actor_id,omitempty" yaml:"actor_id,omitempty"`
	BypassMode *string `json:"bypass_mode,omitempty" yaml:"bypass_mode,omitempty"`
}
type RulesetRule struct {
	Type       string          `json:"type" yaml:"type"`
	Parameters *RuleParameters `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// RulesetState contains a complete supported writable policy and its transport ID.
type RulesetState struct {
	ID     int64
	Policy RulesetDeclaration
}
type RulesetDeclarations struct{ entries []RulesetDeclaration }
type RulesetObservation struct {
	entries      []RulesetState
	excludedPush int
}

func validRulesetText(value string) bool {
	if strings.TrimSpace(value) == "" || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func supportedRule(kind string) bool {
	return slices.Contains([]string{"deletion", "non_fast_forward", "required_linear_history", "required_signatures", "pull_request", "required_status_checks"}, kind)
}
func ValidateRuleset(d RulesetDeclaration, complete bool) error {
	if !validRulesetText(d.Name) {
		return fmt.Errorf("ruleset name must be a nonempty string without control characters")
	}
	if d.Target != nil && *d.Target != "branch" && *d.Target != "tag" {
		return fmt.Errorf("unsupported ruleset target %q", *d.Target)
	}
	if d.Enforcement != nil && !slices.Contains([]string{"active", "disabled", "evaluate"}, *d.Enforcement) {
		return fmt.Errorf("unsupported ruleset enforcement %q", *d.Enforcement)
	}
	if complete && (d.Target == nil || d.Enforcement == nil || d.Conditions == nil || d.Conditions.RefName == nil || d.Conditions.RefName.Include == nil || d.Conditions.RefName.Exclude == nil || d.BypassActors == nil || d.Rules == nil) {
		return fmt.Errorf("ruleset %q requires target, enforcement, ref include/exclude, bypass_actors and rules", d.Name)
	}
	if d.Conditions != nil && d.Conditions.RefName != nil {
		for _, list := range []*[]string{d.Conditions.RefName.Include, d.Conditions.RefName.Exclude} {
			if list == nil {
				continue
			}
			seen := map[string]bool{}
			for _, v := range *list {
				if !validRulesetText(v) || seen[v] {
					return fmt.Errorf("invalid or duplicate ruleset ref pattern %q", v)
				}
				seen[v] = true
			}
		}
	}
	if d.Rules != nil {
		seen := map[string]bool{}
		for _, r := range *d.Rules {
			if !supportedRule(r.Type) || seen[r.Type] {
				return fmt.Errorf("unsupported or duplicate ruleset rule %q", r.Type)
			}
			seen[r.Type] = true
			if r.Type == "pull_request" && d.Target != nil && *d.Target != "branch" {
				return fmt.Errorf("pull_request rule requires branch target")
			}
			if err := validateRule(r, complete); err != nil {
				return err
			}
		}
	}
	if d.BypassActors != nil {
		seen := map[string]bool{}
		for _, a := range *d.BypassActors {
			switch a.ActorType {
			case "Integration", "RepositoryRole", "Team", "User":
				if a.ActorID == nil || *a.ActorID <= 0 {
					return fmt.Errorf("bypass actor %q requires a positive actor_id", a.ActorType)
				}
			case "OrganizationAdmin", "DeployKey":
				if a.ActorID != nil {
					return fmt.Errorf("bypass actor %q must omit actor_id", a.ActorType)
				}
			default:
				return fmt.Errorf("unsupported bypass actor %q", a.ActorType)
			}
			key := actorKey(a)
			if seen[key] {
				return fmt.Errorf("duplicate bypass actor %s", key)
			}
			seen[key] = true
			if complete && a.BypassMode == nil {
				return fmt.Errorf("new bypass actor requires bypass_mode")
			}
			if a.BypassMode != nil {
				if !slices.Contains([]string{"always", "pull_request", "exempt"}, *a.BypassMode) {
					return fmt.Errorf("unsupported bypass_mode %q", *a.BypassMode)
				}
				if *a.BypassMode == "pull_request" && (a.ActorType == "DeployKey" || (d.Target != nil && *d.Target != "branch")) {
					return fmt.Errorf("pull_request bypass requires branch and non-DeployKey actor")
				}
			}
		}
	}
	return nil
}
func actorKey(a BypassActor) string {
	if a.ActorID == nil {
		return a.ActorType
	}
	return fmt.Sprintf("%s:%d", a.ActorType, *a.ActorID)
}
func cloneRuleset(d RulesetDeclaration) RulesetDeclaration {
	if d.Target != nil {
		v := *d.Target
		d.Target = &v
	}
	if d.Enforcement != nil {
		v := *d.Enforcement
		d.Enforcement = &v
	}
	if d.Conditions != nil {
		c := *d.Conditions
		d.Conditions = &c
		if c.RefName != nil {
			r := *c.RefName
			d.Conditions.RefName = &r
			for _, p := range []**[]string{&r.Include, &r.Exclude} {
				if *p != nil {
					v := append([]string{}, (**p)...)
					slices.Sort(v)
					*p = &v
				}
			}
			d.Conditions.RefName = &r
		}
	}
	if d.BypassActors != nil {
		a := append([]BypassActor{}, (*d.BypassActors)...)
		for i := range a {
			if a[i].ActorID != nil {
				v := *a[i].ActorID
				a[i].ActorID = &v
			}
			if a[i].BypassMode != nil {
				v := *a[i].BypassMode
				a[i].BypassMode = &v
			}
		}
		slices.SortFunc(a, func(x, y BypassActor) int {
			if c := strings.Compare(x.ActorType, y.ActorType); c != 0 {
				return c
			}
			var ix, iy int64
			if x.ActorID != nil {
				ix = *x.ActorID
			}
			if y.ActorID != nil {
				iy = *y.ActorID
			}
			if ix < iy {
				return -1
			}
			if ix > iy {
				return 1
			}
			return 0
		})
		d.BypassActors = &a
	}
	if d.Rules != nil {
		r := append([]RulesetRule{}, (*d.Rules)...)
		for i := range r {
			r[i] = cloneRulesetRule(r[i])
		}
		slices.SortFunc(r, func(a, b RulesetRule) int { return strings.Compare(a.Type, b.Type) })
		d.Rules = &r
	}
	return d
}
func NewRulesetDeclarations(entries []RulesetDeclaration) (RulesetDeclarations, error) {
	result := RulesetDeclarations{entries: make([]RulesetDeclaration, 0, len(entries))}
	seen := map[string]bool{}
	for _, d := range entries {
		if err := ValidateRuleset(d, false); err != nil {
			return RulesetDeclarations{}, err
		}
		if seen[d.Name] {
			return RulesetDeclarations{}, fmt.Errorf("duplicate ruleset name %q", d.Name)
		}
		seen[d.Name] = true
		result.entries = append(result.entries, cloneRuleset(d))
	}
	slices.SortFunc(result.entries, func(a, b RulesetDeclaration) int { return strings.Compare(a.Name, b.Name) })
	return result, nil
}
func (d RulesetDeclarations) Values() []RulesetDeclaration {
	out := make([]RulesetDeclaration, 0, len(d.entries))
	for _, v := range d.entries {
		out = append(out, cloneRuleset(v))
	}
	return out
}
func NewRulesetObservation(entries []RulesetState, excludedPush int) (RulesetObservation, error) {
	if excludedPush < 0 {
		return RulesetObservation{}, fmt.Errorf("invalid excluded push count")
	}
	out := RulesetObservation{entries: make([]RulesetState, 0, len(entries)), excludedPush: excludedPush}
	names := map[string]bool{}
	ids := map[int64]bool{}
	for _, v := range entries {
		if err := ValidateRuleset(v.Policy, true); err != nil {
			return RulesetObservation{}, err
		}
		if v.ID <= 0 || ids[v.ID] || names[v.Policy.Name] {
			return RulesetObservation{}, fmt.Errorf("ambiguous ruleset observation")
		}
		ids[v.ID] = true
		names[v.Policy.Name] = true
		out.entries = append(out.entries, RulesetState{ID: v.ID, Policy: cloneRuleset(v.Policy)})
	}
	return out, nil
}
func (o RulesetObservation) Values() ([]RulesetState, error) {
	if o.entries == nil {
		return nil, fmt.Errorf("missing rulesets observation")
	}
	out := make([]RulesetState, 0, len(o.entries))
	for _, v := range o.entries {
		out = append(out, RulesetState{ID: v.ID, Policy: cloneRuleset(v.Policy)})
	}
	return out, nil
}
func (o RulesetObservation) ExcludedPush() int { return o.excludedPush }

// RulesetChange owns validated before/after policies; no arbitrary payload enters CRUD.
type RulesetChange struct {
	operation     string
	id            int64
	before, after *RulesetDeclaration
}

func (c RulesetChange) Operation() string { return c.operation }
func (c RulesetChange) ID() int64         { return c.id }
func (c RulesetChange) Name() string {
	if c.before != nil {
		return c.before.Name
	}
	if c.after != nil {
		return c.after.Name
	}
	return ""
}
func (c RulesetChange) Before() *RulesetDeclaration {
	if c.before == nil {
		return nil
	}
	v := cloneRuleset(*c.before)
	return &v
}
func (c RulesetChange) After() *RulesetDeclaration {
	if c.after == nil {
		return nil
	}
	v := cloneRuleset(*c.after)
	return &v
}
func NewRulesetChange(operation string, id int64, before, after *RulesetDeclaration) (RulesetChange, error) {
	for _, d := range []*RulesetDeclaration{before, after} {
		if d != nil {
			if err := ValidateRuleset(*d, true); err != nil {
				return RulesetChange{}, err
			}
		}
	}
	valid := (operation == "create" && id == 0 && before == nil && after != nil) || (operation == "delete" && id > 0 && before != nil && after == nil) || (operation == "update" && id > 0 && before != nil && after != nil && before.Name == after.Name && !reflect.DeepEqual(cloneRuleset(*before), cloneRuleset(*after)))
	if !valid {
		return RulesetChange{}, fmt.Errorf("invalid ruleset operation")
	}
	c := RulesetChange{operation: operation, id: id}
	if before != nil {
		v := cloneRuleset(*before)
		c.before = &v
	}
	if after != nil {
		v := cloneRuleset(*after)
		c.after = &v
	}
	return c, nil
}
func ReconcileRulesets(current RulesetObservation, desired *RulesetDeclarations) ([]RulesetChange, error) {
	if desired == nil {
		return nil, nil
	}
	values, err := current.Values()
	if err != nil {
		return nil, err
	}
	observed := map[string]RulesetState{}
	for _, v := range values {
		observed[v.Policy.Name] = v
	}
	out := []RulesetChange{}
	for _, d := range desired.Values() {
		before, present := observed[d.Name]
		delete(observed, d.Name)
		after := cloneRuleset(d)
		var prior *RulesetDeclaration
		var id int64
		op := "create"
		if present {
			op = "update"
			id = before.ID
			prior = &before.Policy
			after = mergeRuleset(before.Policy, d)
			if reflect.DeepEqual(after, cloneRuleset(before.Policy)) {
				continue
			}
		}
		if err := ValidateRuleset(after, true); err != nil {
			return nil, err
		}
		c, err := NewRulesetChange(op, id, prior, &after)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	names := []string{}
	for n := range observed {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, n := range names {
		v := observed[n]
		c, err := NewRulesetChange("delete", v.ID, &v.Policy, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}
func mergeRuleset(before, desired RulesetDeclaration) RulesetDeclaration {
	out := cloneRuleset(before)
	d := cloneRuleset(desired)
	if d.Target != nil {
		out.Target = d.Target
	}
	if d.Enforcement != nil {
		out.Enforcement = d.Enforcement
	}
	if d.Rules != nil {
		old := map[string]RulesetRule{}
		for _, r := range *before.Rules {
			old[r.Type] = r
		}
		for i := range *d.Rules {
			r := &(*d.Rules)[i]
			if prior, ok := old[r.Type]; ok {
				r.Parameters = mergeRuleParameters(prior.Parameters, r.Parameters)
			}
		}
		out.Rules = d.Rules
	}
	if d.Conditions != nil && d.Conditions.RefName != nil {
		r := d.Conditions.RefName
		if r.Include != nil {
			out.Conditions.RefName.Include = r.Include
		}
		if r.Exclude != nil {
			out.Conditions.RefName.Exclude = r.Exclude
		}
	}
	if d.BypassActors != nil {
		existing := map[string]BypassActor{}
		for _, a := range *before.BypassActors {
			existing[actorKey(a)] = a
		}
		for i, a := range *d.BypassActors {
			if a.BypassMode == nil {
				if old, ok := existing[actorKey(a)]; ok {
					(*d.BypassActors)[i].BypassMode = old.BypassMode
				}
			}
		}
		out.BypassActors = d.BypassActors
	}
	return cloneRuleset(out)
}
