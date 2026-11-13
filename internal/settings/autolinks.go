package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const AutolinksPath = "repository.settings.autolinks"

type Autolink = uimodel.Autolink

type AutolinkTarget struct {
	ID, OwnerID int64
	FullName    string
	Archived    bool
}

// UpdatedAt presence is private acquisition evidence, not desired configuration.
type AutolinkState struct {
	Value            Autolink
	UpdatedAtPresent bool
	UpdatedAt        *string
}
type AutolinkObservation struct {
	Target  AutolinkTarget
	Entries []AutolinkState
}
type AutolinkWriteResult struct {
	Attempted  bool
	RecoveryID int64
	Verified   *AutolinkState
}
type AutolinkChange struct {
	Before *Autolink `json:"before,omitempty"`
	After  *Autolink `json:"after,omitempty"`
}

func CloneAutolink(v Autolink) Autolink {
	if v.ID != nil {
		id := *v.ID
		v.ID = &id
	}
	return v
}
func CloneAutolinks(values []Autolink) []Autolink {
	out := make([]Autolink, 0, len(values))
	for _, v := range values {
		out = append(out, CloneAutolink(v))
	}
	return out
}
func ValidateAutolinks(values []Autolink) error {
	ids := map[int64]bool{}
	for i, v := range values {
		if v.ID != nil {
			if *v.ID <= 0 || ids[*v.ID] {
				return fmt.Errorf("autolinks require unique positive ids")
			}
			ids[*v.ID] = true
		}
		if strings.TrimSpace(v.ReferencePrefix) == "" || strings.ContainsFunc(v.ReferencePrefix, unicode.IsControl) {
			return fmt.Errorf("autolinks require a nonblank reference_prefix without controls")
		}
		if v.ReferenceFormat != "numeric" && v.ReferenceFormat != "alphanumeric" {
			return fmt.Errorf("autolinks require numeric or alphanumeric reference_format")
		}
		if !strings.Contains(v.TargetURL, "<num>") || strings.ContainsFunc(v.TargetURL, unicode.IsControl) {
			return fmt.Errorf("autolinks require target_url with literal <num>")
		}
		u, err := url.Parse(strings.ReplaceAll(v.TargetURL, "<num>", "123"))
		if err != nil || u.Hostname() == "" || u.User != nil || !u.IsAbs() || (!strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https")) {
			return fmt.Errorf("autolinks require absolute HTTP(S) target_url without userinfo")
		}
		for _, prior := range values[:i] {
			if AutolinkPrefixesOverlap(v.ReferencePrefix, prior.ReferencePrefix) {
				return fmt.Errorf("autolink reference prefixes overlap")
			}
		}
	}
	return nil
}
func AutolinkPrefixesOverlap(a, b string) bool {
	return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
}
func AutolinkMatches(a, b Autolink) bool {
	return a.ReferencePrefix == b.ReferencePrefix && a.TargetURL == b.TargetURL && a.ReferenceFormat == b.ReferenceFormat
}
func (o AutolinkObservation) Validate() error {
	parts := strings.Split(o.Target.FullName, "/")
	if o.Target.ID <= 0 || o.Target.OwnerID <= 0 || len(parts) != 2 || parts[0] == "" || parts[1] == "" || o.Entries == nil {
		return fmt.Errorf("incomplete autolink repository inventory")
	}
	values := make([]Autolink, 0, len(o.Entries))
	for _, e := range o.Entries {
		if e.Value.ID == nil {
			return fmt.Errorf("missing acquired autolink identity")
		}
		if !e.UpdatedAtPresent && e.UpdatedAt != nil {
			return fmt.Errorf("invalid autolink timestamp evidence")
		}
		if e.UpdatedAt != nil {
			if _, err := time.Parse(time.RFC3339, *e.UpdatedAt); err != nil {
				return fmt.Errorf("invalid autolink timestamp evidence")
			}
		}
		values = append(values, e.Value)
	}
	return ValidateAutolinks(values)
}
func (o AutolinkObservation) Clone() AutolinkObservation {
	result := AutolinkObservation{Target: o.Target, Entries: make([]AutolinkState, 0, len(o.Entries))}
	for _, e := range o.Entries {
		e.Value = CloneAutolink(e.Value)
		if e.UpdatedAt != nil {
			v := *e.UpdatedAt
			e.UpdatedAt = &v
		}
		result.Entries = append(result.Entries, e)
	}
	return result
}
func (o AutolinkObservation) AfterUnarchive() AutolinkObservation {
	o = o.Clone()
	o.Target.Archived = false
	return o
}
func (o AutolinkObservation) Find(id int64) (AutolinkState, bool) {
	for _, e := range o.Entries {
		if e.Value.ID != nil && *e.Value.ID == id {
			return e, true
		}
	}
	return AutolinkState{}, false
}
func (o AutolinkObservation) WithCreated(e AutolinkState) (AutolinkObservation, error) {
	o = o.Clone()
	o.Entries = append(o.Entries, e)
	return o, o.Validate()
}
func (o AutolinkObservation) Without(id int64) AutolinkObservation {
	o = o.Clone()
	o.Entries = slices.DeleteFunc(o.Entries, func(e AutolinkState) bool { return *e.Value.ID == id })
	return o
}
func VerifyAutolinkInventory(expected, actual AutolinkObservation) error {
	if err := expected.Validate(); err != nil {
		return err
	}
	if err := actual.Validate(); err != nil {
		return err
	}
	if expected.Target != actual.Target || len(expected.Entries) != len(actual.Entries) {
		return fmt.Errorf("autolink repository or inventory changed")
	}
	for _, e := range expected.Entries {
		a, ok := actual.Find(*e.Value.ID)
		if !ok || !reflect.DeepEqual(e, a) {
			return fmt.Errorf("autolink identity or configuration changed")
		}
	}
	return nil
}
func ExportAutolinks(o AutolinkObservation) ([]Autolink, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	values := make([]Autolink, 0, len(o.Entries))
	for _, e := range o.Entries {
		values = append(values, CloneAutolink(e.Value))
	}
	slices.SortFunc(values, func(a, b Autolink) int { return strings.Compare(a.ReferencePrefix, b.ReferencePrefix) })
	return values, nil
}
func CompareAutolinks(o AutolinkObservation, desired []Autolink, prune bool) ([]AutolinkChange, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	if err := ValidateAutolinks(desired); err != nil {
		return nil, err
	}
	changes := []AutolinkChange{}
	keep := map[int64]bool{}
	for _, d := range desired {
		if d.ID != nil {
			e, ok := o.Find(*d.ID)
			if !ok || !AutolinkMatches(e.Value, d) {
				return nil, fmt.Errorf("bound autolink is missing or differs; explicit removal and later creation are required")
			}
			keep[*d.ID] = true
		} else {
			for _, e := range o.Entries {
				if AutolinkPrefixesOverlap(e.Value.ReferencePrefix, d.ReferencePrefix) {
					return nil, fmt.Errorf("create-only autolink overlaps an existing reference")
				}
			}
			v := CloneAutolink(d)
			changes = append(changes, AutolinkChange{After: &v})
		}
	}
	if prune {
		for _, e := range o.Entries {
			if !keep[*e.Value.ID] {
				v := CloneAutolink(e.Value)
				changes = append(changes, AutolinkChange{Before: &v})
			}
		}
	}
	return changes, nil
}
func ValidateAutolinkChanges(changes []AutolinkChange) error {
	values := []Autolink{}
	for _, c := range changes {
		if (c.Before == nil) == (c.After == nil) {
			return fmt.Errorf("autolink reports require exactly one configuration")
		}
		if c.Before != nil {
			if c.Before.ID == nil {
				return fmt.Errorf("autolink deletion requires identity")
			}
			values = append(values, *c.Before)
		}
		if c.After != nil {
			if c.After.ID != nil {
				return fmt.Errorf("autolink creation cannot adopt identity")
			}
			values = append(values, *c.After)
		}
	}
	return ValidateAutolinks(values)
}
func (c *AutolinkChange) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil || raw == nil {
		return fmt.Errorf("invalid autolink report")
	}
	for key, value := range raw {
		if (key != "before" && key != "after") || bytes.Equal(value, []byte("null")) {
			return fmt.Errorf("invalid autolink report member")
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(value, &fields) != nil || fields == nil {
			return fmt.Errorf("invalid autolink configuration")
		}
		for name, v := range fields {
			if (name != "id" && name != "reference_prefix" && name != "target_url" && name != "reference_format") || bytes.Equal(v, []byte("null")) {
				return fmt.Errorf("invalid autolink configuration member")
			}
		}
	}
	type wire AutolinkChange
	var result wire
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&result); err != nil {
		return err
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return fmt.Errorf("invalid autolink report")
	}
	*c = AutolinkChange(result)
	return nil
}
