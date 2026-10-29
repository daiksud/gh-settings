package settings

import "fmt"

func compareObservedCreationPolicy[S any](result *Comparison, fields []repositoryField[S, string], observed, desired RepositoryDeclaration, normalization ObservationNormalization) error {
	if normalization == (ObservationNormalization{}) {
		return compareObservedFields(result, fields, observed, desired, stringChange)
	}
	for _, field := range fields {
		after := field.desired(desired)
		if after == nil {
			continue
		}
		before := field.desired(observed)
		if before == nil {
			return fmt.Errorf("repository observation is missing %s", field.name)
		}
		change, err := NewCreationPolicyChangeWithNormalization(FieldID{path: field.name}, *before, *after, normalization)
		if err != nil {
			return err
		}
		result.Managed++
		result.Changes = append(result.Changes, change)
	}
	return nil
}
