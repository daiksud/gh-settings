package settings

import (
	"fmt"
	"reflect"
	"strings"
)

const PagesFieldCount = 5

// PagesDomain distinguishes an omitted declaration from an explicit domain removal.
type PagesDomain struct{ Value *string }
type PagesSource struct {
	Branch string `json:"branch" yaml:"branch"`
	Path   string `json:"path" yaml:"path"`
}
type PagesDeclaration struct {
	BuildType     *string
	Source        *PagesSource
	CNAME         *PagesDomain
	HTTPSEnforced *bool
}

func (d PagesDeclaration) Empty() bool {
	return d.BuildType == nil && d.Source == nil && d.CNAME == nil && d.HTTPSEnforced == nil
}
func validPagesMode(s string) bool { return s == "legacy" || s == "workflow" }
func ValidatePagesDomain(s string) error {
	if len(s) == 0 || len(s) > 253 || !strings.Contains(s, ".") {
		return fmt.Errorf("pages.cname must be an ASCII DNS hostname or null")
	}
	for _, label := range strings.Split(s, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("invalid pages.cname DNS label")
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return fmt.Errorf("pages.cname must be an ASCII DNS hostname")
			}
		}
	}
	return nil
}
func (s PagesSource) Validate() error {
	if !ValidDefaultBranch(s.Branch) {
		return fmt.Errorf("pages.source.branch must be a nonempty literal branch name")
	}
	if s.Path != "/" && s.Path != "/docs" {
		return fmt.Errorf("pages.source.path must be / or /docs")
	}
	return nil
}
func (d PagesDeclaration) Validate() error {
	if d.BuildType != nil && !validPagesMode(*d.BuildType) {
		return fmt.Errorf("pages.build_type must be legacy or workflow")
	}
	if d.Source != nil {
		if err := d.Source.Validate(); err != nil {
			return err
		}
		if d.BuildType != nil && *d.BuildType == "workflow" {
			return fmt.Errorf("workflow Pages cannot declare source")
		}
	}
	if d.CNAME != nil && d.CNAME.Value != nil {
		return ValidatePagesDomain(*d.CNAME.Value)
	}
	return nil
}

type PagesState struct {
	BuildType     string
	Source        *PagesSource
	CNAME         *string
	HTTPSEnforced bool
	Public        bool
}
type PagesObservation struct {
	acquired bool
	state    PagesState
}

func copyPagesState(s PagesState) PagesState {
	if s.Source != nil {
		v := *s.Source
		s.Source = &v
	}
	if s.CNAME != nil {
		v := strings.ToLower(*s.CNAME)
		s.CNAME = &v
	}
	return s
}
func NewPagesObservation(s PagesState) (PagesObservation, error) {
	if !validPagesMode(s.BuildType) {
		return PagesObservation{}, fmt.Errorf("missing or unsupported Pages build_type")
	}
	if s.BuildType == "legacy" {
		if s.Source == nil {
			return PagesObservation{}, fmt.Errorf("missing legacy Pages source")
		}
		if err := s.Source.Validate(); err != nil {
			return PagesObservation{}, err
		}
	} else {
		s.Source = nil
	}
	if s.CNAME != nil {
		if err := ValidatePagesDomain(*s.CNAME); err != nil {
			return PagesObservation{}, err
		}
	}
	return PagesObservation{true, copyPagesState(s)}, nil
}
func (o PagesObservation) Values() (PagesState, error) {
	if !o.acquired {
		return PagesState{}, fmt.Errorf("Pages observation was not acquired")
	}
	return copyPagesState(o.state), nil
}
func (o PagesObservation) Declaration() (PagesDeclaration, error) {
	s, err := o.Values()
	if err != nil {
		return PagesDeclaration{}, err
	}
	return PagesDeclaration{BuildType: &s.BuildType, Source: s.Source, CNAME: &PagesDomain{Value: s.CNAME}, HTTPSEnforced: &s.HTTPSEnforced}, nil
}
func PreparePages(o PagesObservation, d PagesDeclaration) (PagesState, error) {
	if err := d.Validate(); err != nil {
		return PagesState{}, err
	}
	s, err := o.Values()
	if err != nil {
		return PagesState{}, err
	}
	if d.BuildType != nil && *d.BuildType == "legacy" && s.BuildType != "legacy" && d.Source == nil {
		return PagesState{}, fmt.Errorf("transition to legacy Pages requires a complete source")
	}
	if d.BuildType != nil {
		s.BuildType = *d.BuildType
	}
	if d.Source != nil {
		if s.BuildType != "legacy" {
			return PagesState{}, fmt.Errorf("Pages source requires legacy mode")
		}
		v := *d.Source
		s.Source = &v
	}
	if s.BuildType == "workflow" {
		s.Source = nil
	}
	if d.CNAME != nil {
		s.CNAME = d.CNAME.Value
	}
	if d.HTTPSEnforced != nil {
		s.HTTPSEnforced = *d.HTTPSEnforced
	}
	return copyPagesState(s), nil
}

// PagesChange owns the nullable-domain and inactive-source report boundaries.
type PagesChange struct {
	field         string
	before, after any
	inactive      bool
}

func (c PagesChange) Field() string        { return c.field }
func (c PagesChange) Values() (any, any)   { return c.before, c.after }
func (c PagesChange) BeforeInactive() bool { return c.inactive }
func NewPagesChange(field string, before, after any, inactive bool) (PagesChange, error) {
	valid := func(v any, previous bool) bool {
		if previous && inactive {
			return (field == "pages.source.branch" || field == "pages.source.path") && v == nil
		}
		switch field {
		case "pages.build_type":
			s, ok := v.(string)
			return ok && validPagesMode(s)
		case "pages.source.branch":
			s, ok := v.(string)
			return ok && (PagesSource{Branch: s, Path: "/"}).Validate() == nil
		case "pages.source.path":
			s, ok := v.(string)
			return ok && (s == "/" || s == "/docs")
		case "pages.cname":
			if v == nil {
				return true
			}
			s, ok := v.(string)
			return ok && ValidatePagesDomain(s) == nil && s == strings.ToLower(s)
		case "pages.https_enforced":
			_, ok := v.(bool)
			return ok
		}
		return false
	}
	if !valid(before, true) || !valid(after, false) || reflect.DeepEqual(before, after) || inactive && field != "pages.source.branch" && field != "pages.source.path" {
		return PagesChange{}, fmt.Errorf("invalid Pages comparison %q", field)
	}
	return PagesChange{field, before, after, inactive}, nil
}
func ValidatePagesChanges(changes []PagesChange) error {
	seen := map[string]PagesChange{}
	for _, c := range changes {
		if _, ok := seen[c.field]; ok {
			return fmt.Errorf("duplicate Pages comparison")
		}
		if _, err := NewPagesChange(c.field, c.before, c.after, c.inactive); err != nil {
			return err
		}
		seen[c.field] = c
	}
	for _, c := range changes {
		if c.inactive {
			m, ok := seen["pages.build_type"]
			if !ok || m.before != "workflow" || m.after != "legacy" || !seen["pages.source.branch"].inactive || !seen["pages.source.path"].inactive {
				return fmt.Errorf("inactive Pages source requires a complete transition to legacy")
			}
		}
	}
	if mode, ok := seen["pages.build_type"]; ok {
		if mode.after == "legacy" && (!seen["pages.source.branch"].inactive || !seen["pages.source.path"].inactive) {
			return fmt.Errorf("transition to legacy Pages requires both inactive source comparisons")
		}
		if mode.after == "workflow" {
			if _, ok := seen["pages.source.branch"]; ok {
				return fmt.Errorf("transition to workflow Pages cannot declare source")
			}
			if _, ok := seen["pages.source.path"]; ok {
				return fmt.Errorf("transition to workflow Pages cannot declare source")
			}
		}
	}
	return nil
}
func ComparePages(o PagesObservation, d PagesDeclaration) (int, []PagesChange, error) {
	after, err := PreparePages(o, d)
	if err != nil {
		return 0, nil, err
	}
	before, _ := o.Values()
	managed := 0
	changes := []PagesChange{}
	add := func(field string, b, a any, inactive bool) {
		managed++
		if !reflect.DeepEqual(b, a) {
			c, _ := NewPagesChange(field, b, a, inactive)
			changes = append(changes, c)
		}
	}
	if d.BuildType != nil {
		add("pages.build_type", before.BuildType, after.BuildType, false)
	}
	if d.Source != nil {
		var branch, path any
		if before.Source != nil {
			branch, path = before.Source.Branch, before.Source.Path
		}
		add("pages.source.branch", branch, after.Source.Branch, before.Source == nil)
		add("pages.source.path", path, after.Source.Path, before.Source == nil)
	}
	if d.CNAME != nil {
		var b, a any
		if before.CNAME != nil {
			b = *before.CNAME
		}
		if after.CNAME != nil {
			a = *after.CNAME
		}
		add("pages.cname", b, a, false)
	}
	if d.HTTPSEnforced != nil {
		add("pages.https_enforced", before.HTTPSEnforced, after.HTTPSEnforced, false)
	}
	return managed, changes, nil
}
