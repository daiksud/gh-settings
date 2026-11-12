package settings

import (
	"fmt"
	"slices"
	"strings"
)

// RepositoryVariable is an explicitly selected plaintext, non-sensitive value.
type RepositoryVariable struct {
	Name  string
	Value string
}

// VariableDeclarations owns canonical, validated names and their complete values.
// An omitted pointer and an empty collection both select no remote objects.
type VariableDeclarations struct{ entries []RepositoryVariable }

func VariableName(name string) (string, error) {
	canonical := strings.ToUpper(name)
	if name == "" || strings.HasPrefix(canonical, "GITHUB_") || name[0] >= '0' && name[0] <= '9' {
		return "", fmt.Errorf("invalid repository variable name %q", name)
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return "", fmt.Errorf("invalid repository variable name %q", name)
		}
	}
	return canonical, nil
}

// CanonicalVariableNames validates name selection independently of values.
func CanonicalVariableNames(names []string) ([]string, error) {
	result := make([]string, len(names))
	seen := map[string]bool{}
	for i, name := range names {
		canonical, err := VariableName(name)
		if err != nil {
			return nil, err
		}
		if seen[canonical] {
			return nil, fmt.Errorf("duplicate repository variable name %q", canonical)
		}
		seen[canonical] = true
		result[i] = canonical
	}
	return result, nil
}
func NewVariableDeclarations(values []RepositoryVariable) (VariableDeclarations, error) {
	names := make([]string, len(values))
	for i, value := range values {
		names[i] = value.Name
	}
	names, err := CanonicalVariableNames(names)
	if err != nil {
		return VariableDeclarations{}, err
	}
	entries := slices.Clone(values)
	for i := range entries {
		if entries[i].Value == "" {
			return VariableDeclarations{}, fmt.Errorf("repository variable %q value must not be empty", names[i])
		}
		entries[i].Name = names[i]
	}
	return VariableDeclarations{entries: entries}, nil
}
func (d VariableDeclarations) Len() int                     { return len(d.entries) }
func (d VariableDeclarations) Values() []RepositoryVariable { return slices.Clone(d.entries) }

// VariableObservation distinguishes acquired absence from an incomplete read.
type VariableObservation struct {
	acquired bool
	value    *RepositoryVariable
}

func NewAbsentVariableObservation() VariableObservation { return VariableObservation{acquired: true} }
func NewVariableObservation(value RepositoryVariable) (VariableObservation, error) {
	name, err := VariableName(value.Name)
	if err != nil {
		return VariableObservation{}, err
	}
	value.Name = name
	return VariableObservation{acquired: true, value: &value}, nil
}
func (o VariableObservation) Value() (*RepositoryVariable, error) {
	if !o.acquired {
		return nil, fmt.Errorf("missing repository variable observation")
	}
	if o.value == nil {
		return nil, nil
	}
	result := *o.value
	return &result, nil
}

// VariableChange is a create (nil before) or value-only update of one name.
type VariableChange struct {
	name    string
	before  *string
	after   string
	deleted bool
}

func NewVariableChange(name string, before *string, after string) (VariableChange, error) {
	if after == "" {
		return VariableChange{}, fmt.Errorf("repository variable value must not be empty")
	}
	canonical, err := VariableName(name)
	if err != nil {
		return VariableChange{}, err
	}
	if before != nil && *before == after {
		return VariableChange{}, fmt.Errorf("unchanged repository variable")
	}
	change := VariableChange{name: canonical, after: after}
	if before != nil {
		value := *before
		change.before = &value
	}
	return change, nil
}
func (c VariableChange) Name() string { return c.name }
func (c VariableChange) Values() (*string, string) {
	if c.before == nil {
		return nil, c.after
	}
	before := *c.before
	return &before, c.after
}
func (c VariableChange) Operation() string {
	if c.deleted {
		return "delete"
	}
	if c.before == nil {
		return "create"
	}
	return "update"
}
func CompareVariable(observed VariableObservation, desired RepositoryVariable) (*VariableChange, error) {
	if desired.Value == "" {
		return nil, fmt.Errorf("repository variable value must not be empty")
	}
	before, err := observed.Value()
	if err != nil {
		return nil, err
	}
	name, err := VariableName(desired.Name)
	if err != nil {
		return nil, err
	}
	var value *string
	if before != nil {
		if before.Name != name {
			return nil, fmt.Errorf("repository variable observation identity mismatch")
		}
		if before.Value == desired.Value {
			return nil, nil
		}
		value = &before.Value
	}
	change, err := NewVariableChange(name, value, desired.Value)
	if err != nil {
		return nil, err
	}
	return &change, nil
}
func SortedVariables(d VariableDeclarations) []RepositoryVariable {
	values := d.Values()
	slices.SortFunc(values, func(a, b RepositoryVariable) int { return strings.Compare(a.Name, b.Name) })
	return values
}

// NewVariableDelete retains the acquired value for exact pre-delete drift checks.
// An empty observed value is still a complete existing object, not absence.
func NewVariableDelete(before RepositoryVariable) (VariableChange, error) {
	name, err := VariableName(before.Name)
	if err != nil {
		return VariableChange{}, err
	}
	value := before.Value
	return VariableChange{name: name, before: &value, deleted: true}, nil
}
