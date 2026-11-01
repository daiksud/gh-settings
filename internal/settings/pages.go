package settings

import (
	"fmt"
	"reflect"
	"strings"
)

const PagesFieldCount = 6

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

// Delete selects the exact UI Branch None sentinel; other branch casing is literal.
func (d PagesDeclaration) Delete() bool {
	return d.Source != nil && d.Source.Branch == "none"
}

func (d PagesDeclaration) Empty() bool {
	return d.BuildType == nil && d.Source == nil && d.CNAME == nil && d.HTTPSEnforced == nil
}
func validPagesMode(s string) bool {
	_, valid := PagesModeChoiceForValue(s)
	return valid
}
func ValidatePagesDomain(s string) error {
	if len(s) == 0 || len(s) > 253 || !strings.Contains(s, ".") {
		return fmt.Errorf("repository.settings.pages.custom_domain must be an ASCII DNS hostname or null")
	}
	for _, label := range strings.Split(s, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("invalid repository.settings.pages.custom_domain DNS label")
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
				return fmt.Errorf("repository.settings.pages.custom_domain must be an ASCII DNS hostname")
			}
		}
	}
	return nil
}
func (s PagesSource) Validate() error {
	if !ValidDefaultBranch(s.Branch) {
		return fmt.Errorf("repository.settings.pages.build_and_deployment.branch must be a nonempty literal branch name")
	}
	if _, valid := PagesFolderChoiceForValue(s.Path); !valid {
		return fmt.Errorf("repository.settings.pages.build_and_deployment.select_folder must be root or docs")
	}
	return nil
}
func (d PagesDeclaration) Validate() error {
	if d.Delete() {
		if d.BuildType == nil || *d.BuildType != "legacy" || d.Source.Path != "" || d.CNAME != nil || d.HTTPSEnforced != nil {
			return fmt.Errorf("Pages branch none requires explicit deploy_from_a_branch and forbids select_folder, custom_domain and enforce_https")
		}
		return nil
	}
	if d.BuildType != nil && !validPagesMode(*d.BuildType) {
		return fmt.Errorf("repository.settings.pages.build_and_deployment.source must be deploy_from_a_branch or github_actions")
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
	acquired            bool
	absent              bool
	state               PagesState
	folderNormalization ObservationNormalization
	modeNormalization   ObservationNormalization
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
	return PagesObservation{acquired: true, state: copyPagesState(s)}, nil
}

// NewAbsentPagesObservation records a successful endpoint-specific absence observation.
func NewAbsentPagesObservation() PagesObservation {
	return PagesObservation{acquired: true, absent: true}
}
func (o PagesObservation) Absent() (bool, error) {
	if !o.acquired {
		return false, fmt.Errorf("Pages observation was not acquired")
	}
	return o.absent, nil
}
func (o PagesObservation) Values() (PagesState, error) {
	if !o.acquired || o.absent {
		return PagesState{}, fmt.Errorf("Pages configured state requires an acquired present site")
	}
	return copyPagesState(o.state), nil
}
func (o PagesObservation) Declaration() (PagesDeclaration, error) {
	if absent, err := o.Absent(); err != nil {
		return PagesDeclaration{}, err
	} else if absent {
		mode := "legacy"
		return PagesDeclaration{BuildType: &mode, Source: &PagesSource{Branch: "none"}}, nil
	}
	s, err := o.Values()
	if err != nil {
		return PagesDeclaration{}, err
	}
	return PagesDeclaration{BuildType: &s.BuildType, Source: s.Source, CNAME: &PagesDomain{Value: s.CNAME}, HTTPSEnforced: &s.HTTPSEnforced}, nil
}
func PreparePages(o PagesObservation, d PagesDeclaration) (PagesState, error) {
	if d.Delete() {
		return PagesState{}, fmt.Errorf("Pages deletion has no configured after state")
	}
	if err := d.Validate(); err != nil {
		return PagesState{}, err
	}
	s, err := o.Values()
	if err != nil {
		return PagesState{}, err
	}
	if d.Source != nil && d.BuildType == nil && o.modeNormalization != (ObservationNormalization{}) {
		return PagesState{}, fmt.Errorf("Pages source requires explicit legacy mode when the observed mode is outside UI choices")
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
	absent        bool
	normalization ObservationNormalization
}

func (c PagesChange) Field() string                           { return c.field }
func (c PagesChange) Values() (any, any)                      { return c.before, c.after }
func (c PagesChange) BeforeAbsent() bool                      { return c.absent }
func (c PagesChange) BeforeInactive() bool                    { return c.inactive }
func (c PagesChange) Normalization() ObservationNormalization { return c.normalization }
func validPagesChangeValue(field string, v any) bool {
	switch field {
	case "pages.lifecycle":
		s, ok := v.(string)
		return ok && (s == "present" || s == "absent")
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
func NewPagesChange(field string, before, after any, inactive bool) (PagesChange, error) {
	beforeValid := validPagesChangeValue(field, before)
	if inactive {
		beforeValid = (field == "pages.source.branch" || field == "pages.source.path") && before == nil
	}
	if !beforeValid || !validPagesChangeValue(field, after) || reflect.DeepEqual(before, after) {
		return PagesChange{}, fmt.Errorf("invalid Pages comparison %q", field)
	}
	return PagesChange{field: field, before: before, after: after, inactive: inactive}, nil
}
func ValidatePagesChanges(changes []PagesChange) error {
	seen := map[string]PagesChange{}
	for _, c := range changes {
		if _, ok := seen[c.field]; ok {
			return fmt.Errorf("duplicate Pages comparison")
		}
		var err error
		if c.normalization != (ObservationNormalization{}) {
			if c.field == "pages.build_type" {
				_, err = NewPagesModeChangeWithNormalization(c.field, c.before, c.after, c.inactive, c.normalization)
			} else {
				_, err = NewPagesChangeWithNormalization(c.field, c.before, c.after, c.inactive, c.normalization)
			}
		} else if c.absent {
			_, err = NewPagesCreationChange(c.field, c.after)
		} else {
			_, err = NewPagesChange(c.field, c.before, c.after, c.inactive)
		}
		if err != nil {
			return err
		}
		seen[c.field] = c
	}
	if life, ok := seen["pages.lifecycle"]; ok {
		if life.after == "absent" {
			if len(changes) != 1 {
				return fmt.Errorf("Pages deletion cannot carry settings changes")
			}
			return nil
		}
		if _, ok := seen["pages.build_type"]; !ok {
			return fmt.Errorf("Pages creation requires mode")
		}
		for _, c := range changes {
			if c.field != "pages.lifecycle" && !c.absent {
				return fmt.Errorf("Pages creation requires absent before values")
			}
		}
		mode := seen["pages.build_type"].after
		if mode == "legacy" {
			if _, ok := seen["pages.source.branch"]; !ok {
				return fmt.Errorf("Pages creation requires branch")
			}
			if _, ok := seen["pages.source.path"]; !ok {
				return fmt.Errorf("Pages creation requires folder")
			}
		}
	}
	for _, c := range changes {
		if c.absent {
			if _, ok := seen["pages.lifecycle"]; !ok {
				return fmt.Errorf("absent Pages comparison requires creation")
			}
		}
		if c.inactive {
			m, ok := seen["pages.build_type"]
			if !ok || m.before != "workflow" || m.after != "legacy" || !seen["pages.source.branch"].inactive || !seen["pages.source.path"].inactive {
				return fmt.Errorf("inactive Pages source requires a complete transition to legacy")
			}
		}
	}
	if mode, ok := seen["pages.build_type"]; ok {
		if mode.before == "workflow" && mode.after == "legacy" && (!seen["pages.source.branch"].inactive || !seen["pages.source.path"].inactive) {
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

// NewPagesCreationChange carries no invented configured before value.
func NewPagesCreationChange(field string, after any) (PagesChange, error) {
	if field == "pages.lifecycle" || !validPagesChangeValue(field, after) || field == "pages.source.branch" && after == "none" {
		return PagesChange{}, fmt.Errorf("invalid Pages creation field %q", field)
	}
	return PagesChange{field: field, after: after, absent: true}, nil
}
func ComparePages(o PagesObservation, d PagesDeclaration) (int, []PagesChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	absent, err := o.Absent()
	if err != nil {
		return 0, nil, err
	}
	if d.Empty() {
		return 0, nil, nil
	}
	if d.Delete() {
		if absent {
			return 1, nil, nil
		}
		c, _ := NewPagesChange("pages.lifecycle", "present", "absent", false)
		return 1, []PagesChange{c}, nil
	}
	if absent {
		if d.BuildType == nil || (*d.BuildType == "legacy" && d.Source == nil) {
			return 0, nil, fmt.Errorf("absent Pages site requires explicit source and complete branch/select_folder for branch deployment")
		}
		c, _ := NewPagesChange("pages.lifecycle", "absent", "present", false)
		changes := []PagesChange{c}
		add := func(f string, v any) { c, _ := NewPagesCreationChange(f, v); changes = append(changes, c) }
		add("pages.build_type", *d.BuildType)
		if d.Source != nil {
			add("pages.source.branch", d.Source.Branch)
			add("pages.source.path", d.Source.Path)
		}
		if d.CNAME != nil {
			var v any
			if d.CNAME.Value != nil {
				v = strings.ToLower(*d.CNAME.Value)
			}
			add("pages.cname", v)
		}
		if d.HTTPSEnforced != nil {
			add("pages.https_enforced", *d.HTTPSEnforced)
		}
		return len(changes), changes, nil
	}
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
		if o.modeNormalization != (ObservationNormalization{}) {
			managed++
			change, err := NewPagesModeChangeWithNormalization("pages.build_type", before.BuildType, after.BuildType, false, o.modeNormalization)
			if err != nil {
				return 0, nil, err
			}
			changes = append(changes, change)
		} else {
			add("pages.build_type", before.BuildType, after.BuildType, false)
		}
	}
	if d.Source != nil {
		var branch, path any
		if before.Source != nil {
			branch, path = before.Source.Branch, before.Source.Path
		}
		add("pages.source.branch", branch, after.Source.Branch, before.Source == nil)
		if o.folderNormalization != (ObservationNormalization{}) {
			managed++
			change, err := NewPagesChangeWithNormalization("pages.source.path", path, after.Source.Path, before.Source == nil, o.folderNormalization)
			if err != nil {
				return 0, nil, err
			}
			changes = append(changes, change)
		} else {
			add("pages.source.path", path, after.Source.Path, before.Source == nil)
		}
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
