package settings

import "fmt"

// NewCreationPolicyChangeWithNormalization retains a required policy write when
// equal effective values cannot establish equality with the acquired raw token.
func NewCreationPolicyChangeWithNormalization(field FieldID, before, after string, normalization ObservationNormalization) (Change, error) {
	control, known := creationPolicyControlForField(field)
	if !known {
		return Change{}, fmt.Errorf("field %q cannot describe creation policy normalization", field.path)
	}
	fallback, valid := control.DefaultChoice()
	native, nativeValid := CreationPolicyValueForChoice(fallback)
	_, rawSupported := CreationPolicyChoiceForValue(normalization.RawValue)
	_, desiredSupported := CreationPolicyChoiceForValue(after)
	if !valid || !nativeValid || before != native || normalization.Path != control.Path() || normalization.Fallback != string(fallback) || rawSupported || !desiredSupported {
		return Change{}, fmt.Errorf("invalid creation policy normalization provenance")
	}
	return Change{field: field, values: stringPair{before: before, after: after}, normalization: normalization}, nil
}
