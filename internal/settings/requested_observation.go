package settings

import "fmt"

// RepositoryObservation retains acquired scalar values independently of omission
// in a desired declaration. A missing observation is never an observed zero value.
type RepositoryObservation struct{ values RepositoryDeclaration }

// NewRepositoryObservation copies only complete observations supplied by readers.
// A nil group means that group was not read.
func NewRepositoryObservation(base *RepositoryState, merge *MergeMethodsState) RepositoryObservation {
	var values RepositoryDeclaration
	if base != nil {
		values = base.Declaration()
	}
	if merge != nil {
		values = values.WithMergeMethods(*merge)
	}
	return RepositoryObservation{values: values}
}

// Fields returns the acquired field identities in stable comparison order.
func (observation RepositoryObservation) Fields() []FieldID {
	return DeclaredRepositoryFields(observation.values)
}

// DeclaredRepositoryFields requests exactly the managed scalar fields.
func DeclaredRepositoryFields(desired RepositoryDeclaration) []FieldID {
	var fields []FieldID
	add := func(name string) { fields = append(fields, FieldID{path: name}) }
	_ = VisitDeclaredRepositoryFields(desired,
		func(name, _ string) error { add(name); return nil },
		func(name string, _ bool) error { add(name); return nil },
	)
	return fields
}

// RepositoryObservationGroups identifies the complete model observations needed
// for a field request. The adapter selects the API for each observation group.
func RepositoryObservationGroups(fields []FieldID) (base, merge bool, err error) {
	for _, field := range fields {
		switch {
		case containsRepositoryField(repositoryStringFields[:], field), containsRepositoryField(repositoryBoolFields[:], field):
			base = true
		case containsRepositoryField(mergeMethodsBoolFields[:], field), containsRepositoryField(mergeMethodsStringFields[:], field):
			merge = true
		default:
			return false, false, fmt.Errorf("field %q is not a supported repository observation", field.path)
		}
	}
	return base, merge, nil
}

func containsRepositoryField[S any, V bool | string](fields []repositoryField[S, V], identity FieldID) bool {
	for _, field := range fields {
		if field.name == identity.path {
			return true
		}
	}
	return false
}

// CompareRepositoryObservation rejects a missing required value before exposing
// any partial comparison, and preserves native values and canonical field order.
func CompareRepositoryObservation(observed RepositoryObservation, desired RepositoryDeclaration) (Comparison, error) {
	var result Comparison
	if err := compareObservedFields(&result, repositoryStringFields[:], observed.values, desired, stringChange); err != nil {
		return Comparison{}, err
	}
	if err := compareObservedFields(&result, repositoryBoolFields[:], observed.values, desired, boolChange); err != nil {
		return Comparison{}, err
	}
	if err := compareObservedFields(&result, mergeMethodsBoolFields[:], observed.values, desired, boolChange); err != nil {
		return Comparison{}, err
	}
	if err := compareObservedFields(&result, mergeMethodsStringFields[:], observed.values, desired, stringChange); err != nil {
		return Comparison{}, err
	}
	return result, nil
}

func compareObservedFields[S any, V bool | string](result *Comparison, fields []repositoryField[S, V], observed, desired RepositoryDeclaration, change func(FieldID, V, V) Change) error {
	for _, field := range fields {
		after := field.desired(desired)
		if after == nil {
			continue
		}
		before := field.desired(observed)
		if before == nil {
			return fmt.Errorf("repository observation is missing %s", field.name)
		}
		result.Managed++
		if *before != *after {
			result.Changes = append(result.Changes, change(FieldID{path: field.name}, *before, *after))
		}
	}
	return nil
}
