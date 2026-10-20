package settings

import "fmt"

// RepositoryObservation retains acquired scalar values independently of omission
// in a desired declaration. A missing observation is never an observed zero value.
type RepositoryObservation struct{ values RepositoryDeclaration }

// NewRepositoryObservation copies only complete observations supplied by readers.
// A nil group means that group was not read.
func NewRepositoryObservation(base *RepositoryState, merge *MergeMethodsState, pr *PRAccessState) RepositoryObservation {
	var values RepositoryDeclaration
	if base != nil {
		values = base.Declaration()
	}
	if merge != nil {
		values = values.WithMergeMethods(*merge)
	}
	if pr != nil {
		values = values.WithPRAccess(*pr)
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
func RepositoryObservationGroups(fields []FieldID) (base, merge, pr bool, err error) {
	for _, field := range fields {
		switch {
		case containsRepositoryField(repositoryStringFields[:], field), containsRepositoryField(repositoryBoolFields[:], field):
			base = true
		case containsRepositoryField(mergeMethodsBoolFields[:], field), containsRepositoryField(mergeMethodsStringFields[:], field):
			merge = true
		case containsRepositoryField(prAccessBoolFields[:], field), containsRepositoryField(prAccessStringFields[:], field):
			pr = true
		default:
			return false, false, false, fmt.Errorf("field %q is not a supported repository observation", field.path)
		}
	}
	return base, merge, pr, nil
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
	if err := compareObservedFields(&result, prAccessBoolFields[:], observed.values, desired, boolChange); err != nil {
		return Comparison{}, err
	}
	if err := compareObservedFields(&result, prAccessStringFields[:], observed.values, desired, stringChange); err != nil {
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

// AllRepositoryFields requests every supported scalar in canonical order.
func AllRepositoryFields() []FieldID {
	fields := make([]FieldID, 0, RepositoryFieldCount())
	fields = appendRepositoryFieldIDs(fields, repositoryStringFields[:])
	fields = appendRepositoryFieldIDs(fields, repositoryBoolFields[:])
	fields = appendRepositoryFieldIDs(fields, mergeMethodsBoolFields[:])
	fields = appendRepositoryFieldIDs(fields, mergeMethodsStringFields[:])
	fields = appendRepositoryFieldIDs(fields, prAccessBoolFields[:])
	return appendRepositoryFieldIDs(fields, prAccessStringFields[:])
}

func appendRepositoryFieldIDs[S any, V bool | string](identities []FieldID, fields []repositoryField[S, V]) []FieldID {
	for _, field := range fields {
		identities = append(identities, FieldID{path: field.name})
	}
	return identities
}

// Declaration returns all supported values only after every field was acquired.
// Values are copied, preserving explicit false, empty strings and enum values.
func (observation RepositoryObservation) Declaration() (RepositoryDeclaration, error) {
	var result RepositoryDeclaration
	if err := copyObservedFields(&result, repositoryStringFields[:], observation.values); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := copyObservedFields(&result, repositoryBoolFields[:], observation.values); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := copyObservedFields(&result, mergeMethodsBoolFields[:], observation.values); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := copyObservedFields(&result, mergeMethodsStringFields[:], observation.values); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := copyObservedFields(&result, prAccessBoolFields[:], observation.values); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := copyObservedFields(&result, prAccessStringFields[:], observation.values); err != nil {
		return RepositoryDeclaration{}, err
	}
	return result, nil
}

func copyObservedFields[S any, V bool | string](result *RepositoryDeclaration, fields []repositoryField[S, V], observed RepositoryDeclaration) error {
	for _, field := range fields {
		value := field.desired(observed)
		if value == nil {
			return fmt.Errorf("repository observation is missing %s", field.name)
		}
		field.setDesired(result, *value)
	}
	return nil
}
