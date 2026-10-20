package settings

import (
	"fmt"
	"reflect"
	"slices"
)

const ActionsPolicyFieldCount = 6

// ActionsPolicyDeclaration contains only explicitly managed repository policy values.
type ActionsPolicyDeclaration struct {
	Enabled            *bool                      `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	AllowedActions     *string                    `yaml:"allowed_actions,omitempty" json:"allowed_actions,omitempty"`
	SHAPinningRequired *bool                      `yaml:"sha_pinning_required,omitempty" json:"sha_pinning_required,omitempty"`
	SelectedActions    SelectedActionsDeclaration `yaml:"selected_actions,omitempty" json:"selected_actions,omitempty"`
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
	Enabled            bool
	AllowedActions     string
	SHAPinningRequired bool
	Selected           *SelectedActionsState
}

func (d SelectedActionsDeclaration) Empty() bool {
	return d.GitHubOwnedAllowed == nil && d.VerifiedAllowed == nil && d.PatternsAllowed == nil
}
func (d SelectedActionsDeclaration) Complete() bool {
	return d.GitHubOwnedAllowed != nil && d.VerifiedAllowed != nil && d.PatternsAllowed != nil
}
func (d ActionsPolicyDeclaration) Empty() bool {
	return d.Enabled == nil && d.AllowedActions == nil && d.SHAPinningRequired == nil && d.SelectedActions.Empty()
}
func ValidAllowedActions(value string) bool {
	return value == "all" || value == "local_only" || value == "selected"
}

func ValidateActionPatterns(values []string) error {
	if values == nil {
		return fmt.Errorf("Actions patterns must be a non-null array")
	}
	if len(values) > 1000 {
		return fmt.Errorf("actions.selected_actions.patterns_allowed must contain at most 1000 patterns")
	}
	for _, v := range values {
		if v == "" {
			return fmt.Errorf("Actions patterns must be nonempty strings")
		}
	}
	return nil
}

func (d ActionsPolicyDeclaration) Validate() error {
	if d.AllowedActions != nil && !ValidAllowedActions(*d.AllowedActions) {
		return fmt.Errorf("actions.allowed_actions must be all, local_only or selected")
	}
	if !d.SelectedActions.Empty() && d.AllowedActions != nil && *d.AllowedActions != "selected" {
		return fmt.Errorf("actions.selected_actions requires allowed_actions selected or omitted")
	}
	if d.SelectedActions.PatternsAllowed != nil {
		return ValidateActionPatterns(*d.SelectedActions.PatternsAllowed)
	}
	return nil
}

func (s ActionsPolicyState) Validate() error {
	if !ValidAllowedActions(s.AllowedActions) {
		return fmt.Errorf("unsupported observed actions.allowed_actions %q", s.AllowedActions)
	}
	if s.Selected != nil {
		if s.AllowedActions != "selected" || s.Selected.PatternsAllowed == nil {
			return fmt.Errorf("invalid acquired selected Actions policy")
		}
		return ValidateActionPatterns(s.Selected.PatternsAllowed)
	}
	return nil
}

// PrepareActions validates all observable prerequisites before any resource write.
func PrepareActions(s ActionsPolicyState, d ActionsPolicyDeclaration) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := d.Validate(); err != nil {
		return err
	}
	if s.AllowedActions != "selected" && d.AllowedActions != nil && *d.AllowedActions == "selected" && !d.SelectedActions.Complete() {
		return fmt.Errorf("transition to selected requires all three selected_actions values; inactive values cannot be read")
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
		return fmt.Errorf("selected Actions values are not observable while allowed_actions is %q; declare selected and all three selected_actions values", s.AllowedActions)
	}
	if !d.SelectedActions.Complete() {
		return fmt.Errorf("transition to selected requires all three selected_actions values; inactive values cannot be read")
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
}

func (c ActionsPolicyChange) Field() string { return c.field }
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
	case "actions.enabled", "actions.sha_pinning_required", "actions.selected_actions.github_owned_allowed", "actions.selected_actions.verified_allowed":
		if v, ok := value.(bool); ok {
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
func ValidateActionsPolicyChanges(changes []ActionsPolicyChange) error {
	seen := map[string]bool{}
	unknown := 0
	transition := false
	for _, c := range changes {
		if seen[c.field] {
			return fmt.Errorf("duplicate Actions change")
		}
		seen[c.field] = true
		if _, err := NewActionsPolicyChange(c.field, c.before, c.after); err != nil {
			return err
		}
		if c.before == nil {
			unknown++
		}
		if c.field == "actions.allowed_actions" && c.before != "selected" && c.after == "selected" {
			transition = true
		}
	}
	if unknown > 0 && (unknown != 3 || !transition) {
		return fmt.Errorf("unobservable selected values require a complete transition to selected")
	}
	return nil
}

func CompareActionsPolicy(s ActionsPolicyState, d ActionsPolicyDeclaration) (int, []ActionsPolicyChange, error) {
	if err := PrepareActions(s, d); err != nil {
		return 0, nil, err
	}
	managed := 0
	changes := []ActionsPolicyChange{}
	add := func(field string, before, after any) {
		managed++
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
	return managed, changes, nil
}
