package settings

import (
	"reflect"
	"testing"
)

func propertyTestSchema(name, kind string) PropertySchema {
	f := false
	a := []string{"a", "b"}
	return PropertySchema{Name: name, Type: kind, Source: "organization", Editable: "org_and_repo_actors", Required: &f, Explicit: &f, RegexObserved: true, DefaultPresent: true, Allowed: &a}
}
func TestPropertiesExplicitUnsetSchemaAndCopies(t *testing.T) {
	schemas := []PropertySchema{propertyTestSchema("text", "string"), propertyTestSchema("tags", "multi_select"), propertyTestSchema("flag", "true_false")}
	values := PropertyDeclarations{"text": "", "tags": []string{"b", "a"}, "flag": "true"}
	o, err := NewPropertyObservation(schemas, values)
	if err != nil {
		t.Fatal(err)
	}
	values["tags"].([]string)[0] = "changed"
	*schemas[0].Required = true
	d, err := o.Declaration()
	if err != nil {
		t.Fatal(err)
	}
	changes, err := CompareProperties(o, d)
	if err != nil || len(changes) != 0 {
		t.Fatal(changes, err)
	}
	d["tags"].([]string)[0] = "mutated"
	v, _ := o.Values()
	if !reflect.DeepEqual(v["tags"], []string{"a", "b"}) {
		t.Fatal("mutable observation", v)
	}
	changes, err = CompareProperties(o, PropertyDeclarations{"text": nil, "tags": []string{}, "flag": "false"})
	if err != nil || len(changes) != 3 {
		t.Fatal(changes, err)
	}
	for _, c := range changes {
		b, a := c.Values()
		if c.Name() == "text" && (b != "" || a != nil) {
			t.Fatal("empty and unset conflated", b, a)
		}
	}
	if _, err := CompareProperties(PropertyObservation{}, PropertyDeclarations{"text": nil}); err == nil {
		t.Fatal("unknown became unset")
	}
	for _, mutate := range []func(*PropertySchema){func(s *PropertySchema) { s.RegexObserved = false }, func(s *PropertySchema) { s.Required = nil }, func(s *PropertySchema) { s.Explicit = nil }, func(s *PropertySchema) { s.DefaultPresent = false }, func(s *PropertySchema) { s.Default = "default" }, func(s *PropertySchema) { v := true; s.Required = &v }, func(s *PropertySchema) { v := true; s.Explicit = &v }, func(s *PropertySchema) { s.Editable = "org_actors" }, func(s *PropertySchema) { s.Source = "enterprise" }, func(s *PropertySchema) { s.Regex = ".*" }, func(s *PropertySchema) { s.Type = "url" }} {
		s := propertyTestSchema("text", "string")
		mutate(&s)
		o, _ := NewPropertyObservation([]PropertySchema{s}, PropertyDeclarations{})
		if _, err := CompareProperties(o, PropertyDeclarations{"text": "x"}); err == nil {
			t.Fatal("unsupported schema accepted", s)
		}
		if _, err := o.Declaration(); err == nil {
			t.Fatal("unsupported export")
		}
	}
	for _, v := range []any{true, 2, []any{false}, []string{"a", "a"}, "bad\nvalue", "bad\"value"} {
		if _, err := PropertyValue(v); err == nil {
			t.Fatal("invalid value", v)
		}
	}
	s := propertyTestSchema("choice", "single_select")
	if _, err := s.ValidateValue("other"); err == nil {
		t.Fatal("outside allowed set")
	}
	if _, err := s.ValidateValue([]string{}); err == nil {
		t.Fatal("wrong type")
	}
	s = propertyTestSchema("flag", "true_false")
	if _, err := s.ValidateValue("False"); err == nil {
		t.Fatal("noncanonical boolean string")
	}
}
