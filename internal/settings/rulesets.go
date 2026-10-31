package settings

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RulesetDeclaration selects one whole supported object; declaration defaults
// are normalized independently of server observations and their optional absence.
type RulesetDeclaration struct {
	ID           *int64             `json:"id,omitempty" yaml:"id,omitempty"`
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

// RulesetState contains the observed supported policy and its transport ID.
// An omitted optional dismissal restriction remains unknown, never disabled.
type RulesetState struct {
	ID                             int64
	Policy                         RulesetDeclaration
	EnforcementNormalization       ObservationNormalization
	QualitySeverityNormalization   ObservationNormalization
	RequiredApprovalsNormalization RulesetRequiredApprovalsNormalization
	MergeMethodsNormalization      RulesetMergeMethodsNormalization
	TeamApprovalsNormalizations    []RulesetTeamApprovalsNormalization
	BypassModeNormalizations       []RulesetBypassModeNormalization
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
	return slices.Contains([]string{"deletion", "non_fast_forward", "required_linear_history", "required_signatures", "pull_request", "required_status_checks", "code_scanning", "code_quality"}, kind)
}
func ValidateRuleset(d RulesetDeclaration, complete bool) error {
	if d.ID != nil && *d.ID <= 0 {
		return fmt.Errorf("ruleset id must be a positive integer")
	}
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
				if _, supported := RulesetBypassModeChoiceForValue(*a.BypassMode); !supported {
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
	if d.ID != nil {
		v := *d.ID
		d.ID = &v
	}
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
	ids := map[int64]bool{}
	for _, d := range entries {
		var err error
		d, err = normalizeRulesetObject(d)
		if err != nil {
			return RulesetDeclarations{}, err
		}
		if seen[d.Name] {
			return RulesetDeclarations{}, fmt.Errorf("duplicate ruleset name %q", d.Name)
		}
		seen[d.Name] = true
		if d.ID != nil {
			if ids[*d.ID] {
				return RulesetDeclarations{}, fmt.Errorf("duplicate ruleset id %d", *d.ID)
			}
			ids[*d.ID] = true
		}
		result.entries = append(result.entries, cloneRuleset(d))
	}
	slices.SortFunc(result.entries, func(a, b RulesetDeclaration) int { return strings.Compare(a.Name, b.Name) })
	return result, nil
}

// A declaration selects one whole supported object. Defaults are independent of
// remote state and therefore identical for create and update.
func normalizeRulesetObject(d RulesetDeclaration) (RulesetDeclaration, error) {
	d = cloneRuleset(d)
	if d.Target == nil {
		value := "branch"
		d.Target = &value
	}
	if d.Enforcement == nil {
		return RulesetDeclaration{}, fmt.Errorf("ruleset %q requires enforcement", d.Name)
	}
	if d.Conditions == nil || d.Conditions.RefName == nil || d.Conditions.RefName.Include == nil {
		return RulesetDeclaration{}, fmt.Errorf("ruleset %q requires conditions.ref_name.include", d.Name)
	}
	if d.Conditions.RefName.Exclude == nil {
		values := []string{}
		d.Conditions.RefName.Exclude = &values
	}
	if d.BypassActors == nil {
		values := []BypassActor{}
		d.BypassActors = &values
	}
	if d.Rules == nil {
		values := []RulesetRule{}
		d.Rules = &values
	}
	for i := range *d.Rules {
		normalizeRuleObject(&(*d.Rules)[i])
	}
	if err := ValidateRuleset(d, true); err != nil {
		return RulesetDeclaration{}, err
	}
	return d, nil
}
func (d RulesetDeclarations) Values() []RulesetDeclaration {
	out := make([]RulesetDeclaration, 0, len(d.entries))
	for _, v := range d.entries {
		out = append(out, cloneRuleset(v))
	}
	return out
}

func (d RulesetDeclarations) Len() int { return len(d.entries) }
func NewRulesetObservation(entries []RulesetState, excludedPush int) (RulesetObservation, error) {
	if excludedPush < 0 {
		return RulesetObservation{}, fmt.Errorf("invalid excluded push count")
	}
	out := RulesetObservation{entries: make([]RulesetState, 0, len(entries)), excludedPush: excludedPush}
	names := map[string]bool{}
	ids := map[int64]bool{}
	for _, v := range entries {
		if _, err := v.RawEnforcement(); err != nil {
			return RulesetObservation{}, err
		}
		if v.ID <= 0 || ids[v.ID] || names[v.Policy.Name] {
			return RulesetObservation{}, fmt.Errorf("ambiguous ruleset observation")
		}
		ids[v.ID] = true
		names[v.Policy.Name] = true
		out.entries = append(out.entries, cloneRulesetState(v))
	}
	return out, nil
}
func (o RulesetObservation) Values() ([]RulesetState, error) {
	if o.entries == nil {
		return nil, fmt.Errorf("missing rulesets observation")
	}
	out := make([]RulesetState, 0, len(o.entries))
	for _, v := range o.entries {
		out = append(out, cloneRulesetState(v))
	}
	return out, nil
}
func (o RulesetObservation) ExcludedPush() int { return o.excludedPush }

// ExportDeclarations excludes write-only dismissal settings, including values
// returned by GET. It must not apply writable declaration defaults.
func (o RulesetObservation) ExportDeclarations() (RulesetDeclarations, error) {
	values, err := o.Values()
	if err != nil {
		return RulesetDeclarations{}, err
	}
	out := RulesetDeclarations{entries: make([]RulesetDeclaration, 0, len(values))}
	for _, value := range values {
		policy, id := value.Policy, value.ID
		policy.ID = &id
		omitDismissalRestriction(&policy)
		out.entries = append(out.entries, policy)
	}
	slices.SortFunc(out.entries, func(a, b RulesetDeclaration) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

// RulesetChange owns validated before/after policies; no arbitrary payload enters CRUD.
type RulesetChange struct {
	operation      string
	id             int64
	before, after  *RulesetDeclaration
	display        RulesetDisplayChange
	normalizations RulesetNormalizations
}

// RulesetDisplayChange contains the complete supported selected objects and IDs.
// It is a closed presentation value, never apply input.
type RulesetDisplayChange struct {
	operation      string
	before, after  *RulesetDeclaration
	normalizations RulesetNormalizations
}

func (c RulesetDisplayChange) Operation() string { return c.operation }
func (c RulesetDisplayChange) Name() string {
	if c.before != nil {
		return c.before.Name
	}
	if c.after != nil {
		return c.after.Name
	}
	return ""
}
func (c RulesetDisplayChange) Before() *RulesetDeclaration {
	if c.before == nil {
		return nil
	}
	v := cloneRuleset(*c.before)
	return &v
}
func (c RulesetDisplayChange) After() *RulesetDeclaration {
	if c.after == nil {
		return nil
	}
	v := cloneRuleset(*c.after)
	return &v
}

func NewRulesetDisplayChange(operation string, before, after *RulesetDeclaration) (RulesetDisplayChange, error) {
	return newRulesetDisplayChange(operation, before, after, RulesetNormalizations{})
}

func newRulesetDisplayChange(operation string, before, after *RulesetDeclaration, normalizations RulesetNormalizations) (RulesetDisplayChange, error) {
	if !normalizations.IsZero() {
		if before == nil || before.ID == nil {
			return RulesetDisplayChange{}, fmt.Errorf("normalized Ruleset display requires selected ID")
		}
		if err := validateRulesetNormalizationChange(operation, *before.ID, before, after, normalizations); err != nil {
			return RulesetDisplayChange{}, err
		}
	}
	for _, d := range []*RulesetDeclaration{before, after} {
		if d != nil {
			if err := ValidateRuleset(*d, true); err != nil {
				return RulesetDisplayChange{}, err
			}
		}
	}
	valid := (operation == "create" && before == nil && after != nil && after.ID == nil) ||
		(operation == "update" && before != nil && after != nil && before.ID != nil && after.ID != nil && *before.ID == *after.ID && (!reflect.DeepEqual(cloneRuleset(*before), cloneRuleset(*after)) || after.HasWriteOnlyDismissal() || !normalizations.IsZero()))
	if !valid {
		return RulesetDisplayChange{}, fmt.Errorf("invalid ruleset display operation")
	}
	out := RulesetDisplayChange{operation: operation, normalizations: cloneRulesetNormalizations(normalizations)}
	if before != nil {
		v := cloneRuleset(*before)
		omitDismissalRestriction(&v)
		out.before = &v
	}
	if after != nil {
		v := cloneRuleset(*after)
		omitDismissalRestriction(&v)
		out.after = &v
	}
	return out, nil
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
func (c RulesetChange) Display() RulesetDisplayChange {
	return RulesetDisplayChange{operation: c.display.operation, before: c.display.Before(), after: c.display.After(), normalizations: cloneRulesetNormalizations(c.display.normalizations)}
}
func NewRulesetChange(operation string, id int64, before, after *RulesetDeclaration) (RulesetChange, error) {
	return newRulesetChange(operation, id, before, after, RulesetNormalizations{})
}

func newRulesetChange(operation string, id int64, before, after *RulesetDeclaration, normalizations RulesetNormalizations) (RulesetChange, error) {
	if !normalizations.IsZero() {
		if err := validateRulesetNormalizationChange(operation, id, before, after, normalizations); err != nil {
			return RulesetChange{}, err
		}
	}
	for _, d := range []*RulesetDeclaration{before, after} {
		if d != nil {
			if err := ValidateRuleset(*d, true); err != nil {
				return RulesetChange{}, err
			}
		}
	}
	valid := (operation == "create" && id == 0 && before == nil && after != nil && after.ID == nil) || (operation == "update" && id > 0 && before != nil && after != nil && (before.ID == nil || *before.ID == id) && (after.ID == nil || *after.ID == id) && (!reflect.DeepEqual(cloneRuleset(*before), cloneRuleset(*after)) || after.HasWriteOnlyDismissal() || !normalizations.IsZero()))
	if !valid {
		return RulesetChange{}, fmt.Errorf("invalid ruleset operation")
	}
	c := RulesetChange{operation: operation, id: id, normalizations: cloneRulesetNormalizations(normalizations)}
	if before != nil {
		v := cloneRuleset(*before)
		v.ID = copyRuleValue(&id)
		c.before = &v
	}
	if after != nil {
		v := cloneRuleset(*after)
		if operation == "update" {
			v.ID = copyRuleValue(&id)
		}
		c.after = &v
	}
	display, err := newRulesetDisplayChange(operation, c.before, c.after, normalizations)
	if err != nil {
		return RulesetChange{}, err
	}
	c.display = display
	return c, nil
}
func ReconcileRulesets(current RulesetObservation, desired *RulesetDeclarations) ([]RulesetChange, error) {
	if desired == nil || len(desired.entries) == 0 {
		return nil, nil
	}
	values, err := current.Values()
	if err != nil {
		return nil, err
	}
	byID := map[int64]RulesetState{}
	byName := map[string]RulesetState{}
	for _, value := range values {
		byID[value.ID] = value
		byName[value.Policy.Name] = value
	}
	out := []RulesetChange{}
	for _, declared := range desired.Values() {
		after := cloneRuleset(declared)
		if declared.ID == nil {
			if existing, present := byName[declared.Name]; present {
				return nil, fmt.Errorf("ruleset %q already exists (id %d); specify id: %d to update it", declared.Name, existing.ID, existing.ID)
			}
			change, err := NewRulesetChange("create", 0, nil, &after)
			if err != nil {
				return nil, err
			}
			out = append(out, change)
			continue
		}
		before, present := byID[*declared.ID]
		if !present {
			return nil, fmt.Errorf("ruleset id %d does not exist in this repository's supported branch/tag objects; refusing create fallback", *declared.ID)
		}
		if existing, present := byName[declared.Name]; present && existing.ID != before.ID {
			return nil, fmt.Errorf("ruleset name %q belongs to existing id %d; refusing rename collision", declared.Name, existing.ID)
		}
		prior, id := cloneRuleset(before.Policy), before.ID
		prior.ID = &id
		comparableBefore := cloneRuleset(prior)
		omitDismissalRestriction(&comparableBefore)
		normalizations := RulesetNormalizations{Enforcement: before.EnforcementNormalization, QualitySeverity: before.QualitySeverityNormalization, RequiredApprovals: before.RequiredApprovalsNormalization, MergeMethods: before.MergeMethodsNormalization, TeamApprovals: before.TeamApprovalsNormalizations, BypassModes: before.BypassModeNormalizations}
		if normalizations.IsZero() && !after.HasWriteOnlyDismissal() && reflect.DeepEqual(comparableBefore, after) {
			continue
		}
		var change RulesetChange
		var err error
		if !normalizations.IsZero() {
			change, err = NewRulesetChangeWithNormalizations("update", id, &prior, &after, normalizations)
		} else {
			change, err = NewRulesetChange("update", id, &prior, &after)
		}
		if err != nil {
			return nil, err
		}
		out = append(out, change)
	}
	slices.SortFunc(out, func(a, b RulesetChange) int { return strings.Compare(a.Name(), b.Name()) })
	return out, nil
}

// WriteOnlyDismissalRulesets identifies declarations whose values cannot be
// compared or exported reliably. Explicit apply always sends these values.
func (d *RulesetDeclarations) WriteOnlyDismissalRulesets() []string {
	var names []string
	if d != nil {
		for _, policy := range d.entries {
			if policy.HasWriteOnlyDismissal() {
				names = append(names, policy.Name)
			}
		}
	}
	return names
}

// HasWriteOnlyDismissal reports explicit write-only intent, not observed drift.
func (policy RulesetDeclaration) HasWriteOnlyDismissal() bool {
	if policy.Rules != nil {
		for _, rule := range *policy.Rules {
			if rule.Parameters != nil && rule.Parameters.DismissalRestriction != nil {
				return true
			}
		}
	}
	return false
}

func omitDismissalRestriction(policy *RulesetDeclaration) {
	if policy.Rules != nil {
		for i := range *policy.Rules {
			if p := (*policy.Rules)[i].Parameters; p != nil {
				p.DismissalRestriction = nil
			}
		}
	}
}
