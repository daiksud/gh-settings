package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
	"reflect"
	"slices"
	"strings"
	"unicode"
)

const ClassicProtectionPath = "repository.settings.branch_protection_rules"

type ClassicRule = uimodel.BranchProtectionRule

// Nullable preserves explicit API null separately from empty, zero and missing.
// Present is acquisition evidence, never a writable default.
type ClassicNullable[T any] struct {
	Present bool
	Value   *T
}
type ClassicCheck struct {
	Context string
	AppID   *string
}
type ClassicActor struct{ Kind, ID string }
type ClassicCompanions struct {
	AdminEnforced, AllowsForcePushes, BlocksCreations, DismissesStaleReviews                     bool
	LockAllowsFetchAndMerge, RequireLastPushApproval                                             bool
	RequiresApprovingReviews, RequiresCodeOwnerReviews, RequiresDeployments                      bool
	RequiresStatusChecks, RequiresStrictStatusChecks, RestrictsPushes, RestrictsReviewDismissals bool
	ReviewCount                                                                                  ClassicNullable[int]
	Deployments, Contexts                                                                        ClassicNullable[[]string]
	Checks                                                                                       ClassicNullable[[]ClassicCheck]
	PushActors, ReviewDismissalActors, ForcePushActors, PullRequestActors                        []ClassicActor
}
type ClassicState struct {
	Rule                 ClassicRule
	NodeID, RepositoryID string
	Companions           ClassicCompanions
}
type ClassicObservation struct {
	RepositoryID string
	Archived     bool
	Rules        []ClassicState
}
type ClassicSelection struct {
	RepositoryID string
	Before       *ClassicState
	Desired      ClassicRule
}

// ClassicChange contains supported presentation values only, never raw companions.
type ClassicChange struct {
	Before *ClassicRule `json:"before,omitempty"`
	After  *ClassicRule `json:"after,omitempty"`
}
type ClassicWriteResult struct {
	ID     int64
	NodeID string
	Wrote  bool
}

func CloneClassicRule(r ClassicRule) ClassicRule {
	r.ID = cloneClassicValue(r.ID)
	r.RequireConversationResolutionBeforeMerging = cloneClassicValue(r.RequireConversationResolutionBeforeMerging)
	r.RequireSignedCommits = cloneClassicValue(r.RequireSignedCommits)
	r.RequireLinearHistory = cloneClassicValue(r.RequireLinearHistory)
	r.AllowDeletions = cloneClassicValue(r.AllowDeletions)
	r.LockBranch = cloneClassicValue(r.LockBranch)
	return r
}
func cloneClassicValue[T any](v *T) *T {
	if v == nil {
		return nil
	}
	c := *v
	return &c
}
func ClassicFlags(r ClassicRule) []*bool {
	return []*bool{r.RequireConversationResolutionBeforeMerging, r.RequireSignedCommits, r.RequireLinearHistory, r.AllowDeletions, r.LockBranch}
}
func ValidateClassicRules(rules []ClassicRule) error {
	ids, creates := map[int64]bool{}, map[string]bool{}
	for _, r := range rules {
		if strings.TrimSpace(r.BranchNamePattern) == "" || strings.ContainsFunc(r.BranchNamePattern, unicode.IsControl) {
			return fmt.Errorf("%s requires a nonblank exact branch_name_pattern without controls", ClassicProtectionPath)
		}
		if r.ID != nil {
			if *r.ID <= 0 || ids[*r.ID] {
				return fmt.Errorf("invalid or duplicate classic rule id")
			}
			ids[*r.ID] = true
		} else {
			if creates[r.BranchNamePattern] {
				return fmt.Errorf("duplicate create-only classic pattern %q", r.BranchNamePattern)
			}
			creates[r.BranchNamePattern] = true
		}
	}
	return nil
}
func (o ClassicObservation) Validate() error {
	if o.RepositoryID == "" || o.Rules == nil {
		return fmt.Errorf("incomplete classic protection inventory")
	}
	ids, nodes := map[int64]bool{}, map[string]bool{}
	for _, s := range o.Rules {
		if err := ValidateClassicRules([]ClassicRule{s.Rule}); err != nil {
			return err
		}
		if s.Rule.ID == nil || ids[*s.Rule.ID] || s.NodeID == "" || nodes[s.NodeID] || s.RepositoryID != o.RepositoryID {
			return fmt.Errorf("incomplete or ambiguous classic rule identity")
		}
		ids[*s.Rule.ID], nodes[s.NodeID] = true, true
		for _, v := range ClassicFlags(s.Rule) {
			if v == nil {
				return fmt.Errorf("incomplete classic protection controls")
			}
		}
		c := s.Companions
		if !c.ReviewCount.Present || !c.Deployments.Present || !c.Contexts.Present || !c.Checks.Present {
			return fmt.Errorf("missing classic companion state")
		}
		if c.RequiresApprovingReviews && c.ReviewCount.Value == nil || c.RequiresDeployments && c.Deployments.Value == nil || c.RequiresStatusChecks && (c.Contexts.Value == nil || c.Checks.Value == nil) {
			return fmt.Errorf("enabled classic group has unavailable companion state")
		}
		for _, actors := range [][]ClassicActor{c.PushActors, c.ReviewDismissalActors, c.ForcePushActors, c.PullRequestActors} {
			if actors == nil {
				return fmt.Errorf("missing classic actor inventory")
			}
			seen := map[string]bool{}
			for _, a := range actors {
				if a.ID == "" || (a.Kind != "User" && a.Kind != "Team" && a.Kind != "App") || seen[a.ID] {
					return fmt.Errorf("invalid classic actor identity")
				}
				seen[a.ID] = true
			}
		}
	}
	return nil
}
func PrepareClassicRules(o ClassicObservation, desired []ClassicRule) ([]ClassicSelection, error) {
	if err := ValidateClassicRules(desired); err != nil {
		return nil, err
	}
	if err := o.Validate(); err != nil {
		return nil, err
	}
	result := make([]ClassicSelection, 0, len(desired))
	for _, d := range desired {
		var selected *ClassicState
		for _, s := range o.Rules {
			if d.ID != nil && *d.ID == *s.Rule.ID {
				copy := s
				selected = &copy
			}
			if s.Rule.BranchNamePattern == d.BranchNamePattern && (d.ID == nil || *d.ID != *s.Rule.ID) {
				// Existing duplicate patterns remain independently selectable without renaming.
				if d.ID == nil {
					return nil, fmt.Errorf("create-only classic pattern %q already exists", d.BranchNamePattern)
				}
			}
		}
		if d.ID != nil && selected == nil {
			return nil, fmt.Errorf("classic rule id %d is missing", *d.ID)
		}
		if selected != nil && selected.Rule.BranchNamePattern != d.BranchNamePattern {
			for _, s := range o.Rules {
				if *s.Rule.ID != *d.ID && s.Rule.BranchNamePattern == d.BranchNamePattern {
					return nil, fmt.Errorf("classic pattern rename collides with existing rule")
				}
			}
		}
		result = append(result, ClassicSelection{RepositoryID: o.RepositoryID, Before: selected, Desired: CloneClassicRule(d)})
	}
	// Requested outcomes may not introduce a collision through two edits/creates.
	for i, a := range result {
		for _, b := range result[i+1:] {
			if a.Desired.BranchNamePattern == b.Desired.BranchNamePattern && (a.Before == nil || b.Before == nil || a.Before.Rule.BranchNamePattern != a.Desired.BranchNamePattern || b.Before.Rule.BranchNamePattern != b.Desired.BranchNamePattern) {
				return nil, fmt.Errorf("classic requested patterns collide")
			}
		}
	}
	return result, nil
}
func ClassicMatches(observed, desired ClassicRule) bool {
	if observed.BranchNamePattern != desired.BranchNamePattern {
		return false
	}
	before, after := ClassicFlags(observed), ClassicFlags(desired)
	for i, v := range after {
		if v != nil && (before[i] == nil || *v != *before[i]) {
			return false
		}
	}
	return true
}
func CompareClassicRules(o ClassicObservation, desired []ClassicRule) ([]ClassicChange, error) {
	selections, err := PrepareClassicRules(o, desired)
	if err != nil {
		return nil, err
	}
	var changes []ClassicChange
	for _, s := range selections {
		if s.Before != nil && ClassicMatches(s.Before.Rule, s.Desired) {
			continue
		}
		after := CloneClassicRule(s.Desired)
		c := ClassicChange{After: &after}
		if s.Before != nil {
			v := CloneClassicRule(s.Before.Rule)
			c.Before = &v
		}
		changes = append(changes, c)
	}
	return changes, nil
}
func ExportClassicRules(o ClassicObservation) ([]ClassicRule, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	r := make([]ClassicRule, 0, len(o.Rules))
	for _, s := range o.Rules {
		r = append(r, CloneClassicRule(s.Rule))
	}
	slices.SortFunc(r, func(a, b ClassicRule) int {
		if *a.ID < *b.ID {
			return -1
		}
		if *a.ID > *b.ID {
			return 1
		}
		return 0
	})
	return r, nil
}
func ValidateClassicChanges(changes []ClassicChange) error {
	desired := make([]ClassicRule, 0, len(changes))
	seen := map[int64]bool{}
	for _, c := range changes {
		if c.Before == nil && c.After == nil {
			return fmt.Errorf("missing classic change")
		}
		if c.After != nil {
			desired = append(desired, *c.After)
		}
		if c.Before == nil {
			if c.After.ID != nil {
				return fmt.Errorf("classic creation cannot have id")
			}
			continue
		}
		if err := ValidateClassicRules([]ClassicRule{*c.Before}); err != nil {
			return err
		}
		if c.Before.ID == nil || seen[*c.Before.ID] {
			return fmt.Errorf("missing or duplicate classic change identity")
		}
		seen[*c.Before.ID] = true
		for _, b := range ClassicFlags(*c.Before) {
			if b == nil {
				return fmt.Errorf("incomplete classic before policy")
			}
		}
		if c.After == nil {
			continue
		}
		if c.After.ID == nil || *c.Before.ID != *c.After.ID {
			return fmt.Errorf("classic update identity differs")
		}
		if ClassicMatches(*c.Before, *c.After) {
			return fmt.Errorf("no-op classic report")
		}
	}
	for _, c := range changes {
		if c.After == nil {
			continue
		}
		for _, other := range changes {
			if other.Before != nil && c.After.BranchNamePattern == other.Before.BranchNamePattern &&
				(c.Before == nil || c.Before.BranchNamePattern != c.After.BranchNamePattern) {
				return fmt.Errorf("classic change pattern collides with an acquired rule")
			}
		}
	}
	return ValidateClassicRules(desired)
}

// VerifyClassicPreservation compares all unowned known controls, including owned
// controls omitted from this partial request. Exact nullable raw state is retained.
func VerifyClassicPreservation(before, after ClassicState, desired ClassicRule) error {
	if before.NodeID != after.NodeID || before.RepositoryID != after.RepositoryID || *before.Rule.ID != *after.Rule.ID {
		return fmt.Errorf("classic rule identity changed")
	}
	if !ClassicMatches(after.Rule, desired) {
		return fmt.Errorf("classic requested controls did not persist")
	}
	if !reflect.DeepEqual(before.Companions, after.Companions) {
		return fmt.Errorf("unowned classic protection changed")
	}
	b, a, d := ClassicFlags(before.Rule), ClassicFlags(after.Rule), ClassicFlags(desired)
	for i, v := range d {
		if v == nil && !reflect.DeepEqual(b[i], a[i]) {
			return fmt.Errorf("omitted classic control changed")
		}
	}
	return nil
}

func (c *ClassicChange) UnmarshalJSON(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var valid func(any) bool
	valid = func(v any) bool {
		if v == nil {
			return false
		}
		if m, ok := v.(map[string]any); ok {
			for k, x := range m {
				if k != strings.ToLower(k) || !valid(x) {
					return false
				}
			}
		}
		return true
	}
	if !valid(value) {
		return fmt.Errorf("null or noncanonical classic report")
	}
	type plain ClassicChange
	var result plain
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return err
	}
	*c = ClassicChange(result)
	return nil
}
