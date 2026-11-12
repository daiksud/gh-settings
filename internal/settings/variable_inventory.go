package settings

import (
	"fmt"
	"slices"
	"strings"
)

// VariableInventory distinguishes a complete empty collection from no acquisition.
// Environment identity binds its values to the existing parent, never a secret.
type VariableInventory struct {
	entries       []RepositoryVariable
	environment   string
	environmentID int64
}

func NewVariableInventory(values []RepositoryVariable) (VariableInventory, error) {
	names := make([]string, len(values))
	for i, v := range values {
		names[i] = v.Name
	}
	canonical, err := CanonicalVariableNames(names)
	if err != nil {
		return VariableInventory{}, err
	}
	owned := make([]RepositoryVariable, len(values))
	for i, v := range values {
		owned[i] = RepositoryVariable{Name: canonical[i], Value: v.Value}
	}
	slices.SortFunc(owned, func(a, b RepositoryVariable) int { return strings.Compare(a.Name, b.Name) })
	return VariableInventory{entries: owned}, nil
}
func NewEnvironmentVariableInventory(environment string, id int64, values []RepositoryVariable) (VariableInventory, error) {
	if id <= 0 {
		return VariableInventory{}, fmt.Errorf("missing Environment inventory identity")
	}
	if _, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: environment}}); err != nil {
		return VariableInventory{}, err
	}
	result, err := NewVariableInventory(values)
	if err != nil {
		return VariableInventory{}, err
	}
	result.environment = environment
	result.environmentID = id
	return result, nil
}
func (o VariableInventory) Values() ([]RepositoryVariable, error) {
	if o.entries == nil {
		return nil, fmt.Errorf("missing complete variables inventory")
	}
	return slices.Clone(o.entries), nil
}
func (o VariableInventory) EnvironmentID(environment string) (int64, error) {
	if o.environmentID <= 0 || EnvironmentIdentity(environment) != EnvironmentIdentity(o.environment) {
		return 0, fmt.Errorf("Environment variable inventory identity mismatch")
	}
	return o.environmentID, nil
}

func PrunedVariableChanges(inventory VariableInventory, desired *VariableDeclarations) ([]VariableChange, error) {
	if desired == nil {
		return nil, nil
	}
	values, err := inventory.Values()
	if err != nil {
		return nil, err
	}
	keep := map[string]bool{}
	for _, v := range desired.Values() {
		keep[v.Name] = true
	}
	var result []VariableChange
	for _, v := range values {
		if !keep[v.Name] {
			c, err := NewVariableDelete(v)
			if err != nil {
				return nil, err
			}
			result = append(result, c)
		}
	}
	return result, nil
}
