package settings

import (
	"fmt"
	"reflect"
	"sort"
)

// PropertyDeclarations owns only named values. A nil value explicitly unsets one.
type PropertyDeclarations map[string]any

func ValidPropertyName(s string) bool {
	if len(s) == 0 || len(s) > 75 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == '$' || c == '#') {
			return false
		}
	}
	return true
}
func propertyString(s string) bool {
	if len(s) > 75 {
		return false
	}
	for _, c := range s {
		if c < 32 || c > 126 || c == '"' {
			return false
		}
	}
	return true
}

// PropertyValue validates the wire/YAML union and defensively canonicalizes sets.
func PropertyValue(v any) (any, error) {
	switch x := v.(type) {
	case nil:
		return nil, nil
	case string:
		if propertyString(x) {
			return x, nil
		}
	case []string:
		a := append([]string{}, x...)
		sort.Strings(a)
		for i, s := range a {
			if !propertyString(s) || i > 0 && a[i-1] == s {
				return nil, fmt.Errorf("invalid or duplicate custom property selection")
			}
		}
		return a, nil
	case []any:
		a := make([]string, len(x))
		for i, v := range x {
			var ok bool
			a[i], ok = v.(string)
			if !ok {
				return nil, fmt.Errorf("custom property arrays require strings")
			}
		}
		return PropertyValue(a)
	}
	return nil, fmt.Errorf("custom property value must be null, a printable ASCII string of at most 75 characters, or a unique string array")
}
func (d PropertyDeclarations) Validate() error {
	for name, v := range d {
		if !ValidPropertyName(name) {
			return fmt.Errorf("invalid custom property name %q", name)
		}
		if _, err := PropertyValue(v); err != nil {
			return fmt.Errorf("custom property %q: %w", name, err)
		}
	}
	return nil
}

// PropertySchema retains unknown optional constraints rather than defaulting them.
type PropertySchema struct {
	Name, Type, Source, Editable string
	Required, Explicit           *bool
	DefaultPresent               bool
	Default                      any
	Allowed                      *[]string
	Regex                        string
	RegexObserved                bool
}

func (s PropertySchema) Adopted() error {
	if !ValidPropertyName(s.Name) || s.Source != "organization" || s.Editable != "org_and_repo_actors" || s.Required == nil || *s.Required || s.Explicit == nil || *s.Explicit || !s.DefaultPresent || s.Default != nil || s.Regex != "" {
		return fmt.Errorf("custom property %q requires a complete repository-editable organization schema without required/default/regex constraints", s.Name)
	}
	switch s.Type {
	case "string":
		if !s.RegexObserved {
			return fmt.Errorf("custom property %q has unavailable string constraints", s.Name)
		}
	case "true_false":
	case "single_select", "multi_select":
		if s.Allowed == nil {
			return fmt.Errorf("custom property %q has unavailable allowed values", s.Name)
		}
		if _, err := PropertyValue(*s.Allowed); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported custom property type %q", s.Type)
	}
	return nil
}
func propertyTyped(v any, kind string) bool {
	if v == nil {
		return true
	}
	if kind == "multi_select" {
		_, ok := v.([]string)
		return ok
	}
	s, ok := v.(string)
	return ok && (kind == "string" || kind == "single_select" || kind == "true_false" && (s == "true" || s == "false"))
}
func (s PropertySchema) ValidateValue(v any) (any, error) {
	if err := s.Adopted(); err != nil {
		return nil, err
	}
	v, err := PropertyValue(v)
	if err != nil {
		return nil, err
	}
	if !propertyTyped(v, s.Type) {
		return nil, fmt.Errorf("custom property %q does not match schema type %s", s.Name, s.Type)
	}
	if v != nil && (s.Type == "single_select" || s.Type == "multi_select") {
		allowed := map[string]bool{}
		for _, x := range *s.Allowed {
			allowed[x] = true
		}
		values, ok := v.([]string)
		if !ok {
			values = []string{v.(string)}
		}
		for _, x := range values {
			if !allowed[x] {
				return nil, fmt.Errorf("custom property %q selection is not allowed by its schema", s.Name)
			}
		}
	}
	return v, nil
}

type PropertyObservation struct {
	acquired bool
	schemas  map[string]PropertySchema
	values   PropertyDeclarations
}

func copyPropertySchema(s PropertySchema) PropertySchema {
	if s.Required != nil {
		v := *s.Required
		s.Required = &v
	}
	if s.Explicit != nil {
		v := *s.Explicit
		s.Explicit = &v
	}
	if s.Allowed != nil {
		v := append([]string{}, (*s.Allowed)...)
		s.Allowed = &v
	}
	s.Default, _ = PropertyValue(s.Default)
	return s
}
func NewPropertyObservation(schemas []PropertySchema, values PropertyDeclarations) (PropertyObservation, error) {
	o := PropertyObservation{true, map[string]PropertySchema{}, PropertyDeclarations{}}
	for _, s := range schemas {
		if !ValidPropertyName(s.Name) || s.Type == "" {
			return PropertyObservation{}, fmt.Errorf("incomplete custom property schema identity")
		}
		if _, ok := o.schemas[s.Name]; ok {
			return PropertyObservation{}, fmt.Errorf("duplicate custom property schema")
		}
		if _, err := PropertyValue(s.Default); err != nil {
			return PropertyObservation{}, err
		}
		o.schemas[s.Name] = copyPropertySchema(s)
	}
	if err := values.Validate(); err != nil {
		return PropertyObservation{}, err
	}
	for name, v := range values {
		if _, ok := o.schemas[name]; !ok {
			return PropertyObservation{}, fmt.Errorf("custom property value has no observed schema")
		}
		if v != nil {
			o.values[name], _ = PropertyValue(v)
		}
	}
	return o, nil
}
func (o PropertyObservation) Values() (PropertyDeclarations, error) {
	if !o.acquired {
		return nil, fmt.Errorf("custom properties observation was not acquired")
	}
	m := PropertyDeclarations{}
	for name, v := range o.values {
		m[name], _ = PropertyValue(v)
	}
	return m, nil
}
func (o PropertyObservation) Schema(name string) (PropertySchema, error) {
	if !o.acquired {
		return PropertySchema{}, fmt.Errorf("custom properties observation was not acquired")
	}
	s, ok := o.schemas[name]
	if !ok {
		return PropertySchema{}, fmt.Errorf("unknown custom property %q", name)
	}
	return copyPropertySchema(s), nil
}
func (o PropertyObservation) Declaration() (PropertyDeclarations, error) {
	if !o.acquired {
		return nil, fmt.Errorf("custom properties observation was not acquired")
	}
	d := PropertyDeclarations{}
	for name, s := range o.schemas {
		v, err := s.ValidateValue(o.values[name])
		if err != nil {
			return nil, err
		}
		d[name] = v
	}
	return d, nil
}

// SelectedDeclaration exports only explicitly named property values.
func (o PropertyObservation) SelectedDeclaration(names []string) (PropertyDeclarations, error) {
	if !o.acquired {
		return nil, fmt.Errorf("custom properties observation was not acquired")
	}
	selected := append([]string{}, names...)
	sort.Strings(selected)
	d := PropertyDeclarations{}
	for i, name := range selected {
		if !ValidPropertyName(name) {
			return nil, fmt.Errorf("invalid custom property name %q", name)
		}
		if i > 0 && selected[i-1] == name {
			return nil, fmt.Errorf("duplicate custom property selection %q", name)
		}
		s, err := o.Schema(name)
		if err != nil {
			return nil, err
		}
		v, err := s.ValidateValue(o.values[name])
		if err != nil {
			return nil, err
		}
		d[name] = v
	}
	return d, nil
}

type PropertyChange struct {
	name, kind    string
	before, after any
}

func (c PropertyChange) Name() string { return c.name }
func (c PropertyChange) Type() string { return c.kind }
func (c PropertyChange) Values() (any, any) {
	b, _ := PropertyValue(c.before)
	a, _ := PropertyValue(c.after)
	return b, a
}
func NewPropertyChange(name, kind string, before, after any) (PropertyChange, error) {
	b, err := PropertyValue(before)
	if err != nil {
		return PropertyChange{}, err
	}
	a, err := PropertyValue(after)
	if err != nil {
		return PropertyChange{}, err
	}
	if !ValidPropertyName(name) || !propertyTyped(b, kind) || !propertyTyped(a, kind) || kind != "string" && kind != "single_select" && kind != "multi_select" && kind != "true_false" || reflect.DeepEqual(b, a) {
		return PropertyChange{}, fmt.Errorf("invalid custom property comparison")
	}
	return PropertyChange{name, kind, b, a}, nil
}
func CompareProperties(o PropertyObservation, d PropertyDeclarations) ([]PropertyChange, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	values, err := o.Values()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(d))
	for name := range d {
		names = append(names, name)
	}
	sort.Strings(names)
	changes := []PropertyChange{}
	for _, name := range names {
		s, err := o.Schema(name)
		if err != nil {
			return nil, err
		}
		after, err := s.ValidateValue(d[name])
		if err != nil {
			return nil, err
		}
		before, err := s.ValidateValue(values[name])
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(before, after) {
			c, err := NewPropertyChange(name, s.Type, before, after)
			if err != nil {
				return nil, err
			}
			changes = append(changes, c)
		}
	}
	return changes, nil
}
