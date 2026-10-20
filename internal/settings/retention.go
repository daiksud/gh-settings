package settings

import "fmt"

func ValidateRetentionDays(days int) error {
	if days < 1 || days > 400 {
		return fmt.Errorf("actions.artifact_and_log_retention_days must be an integer from 1 through 400")
	}
	return nil
}
func (d ActionsPolicyDeclaration) RetentionEmpty() bool { return d.RetentionDays == nil }

// RetentionObservation separates acquired configured days from the unowned cap.
// A managing policy may change independently; do not invent days<=maximum as a
// read invariant. Desired/exported values must pass the current cap separately.
type RetentionObservation struct {
	acquired      bool
	days, maximum int
}

func NewRetentionObservation(days, maximum int) (RetentionObservation, error) {
	if err := ValidateRetentionDays(days); err != nil {
		return RetentionObservation{}, err
	}
	if maximum < 1 {
		return RetentionObservation{}, fmt.Errorf("maximum_allowed_days must be a positive observed integer")
	}
	return RetentionObservation{acquired: true, days: days, maximum: maximum}, nil
}
func (o RetentionObservation) Values() (int, int, error) {
	if !o.acquired {
		return 0, 0, fmt.Errorf("retention observation was not acquired")
	}
	return o.days, o.maximum, nil
}
func PrepareRetention(o RetentionObservation, days int) error {
	if err := ValidateRetentionDays(days); err != nil {
		return err
	}
	_, maximum, err := o.Values()
	if err != nil {
		return err
	}
	if days > maximum {
		return fmt.Errorf("actions.artifact_and_log_retention_days %d exceeds repository maximum_allowed_days %d", days, maximum)
	}
	return nil
}
func (o RetentionObservation) Declaration() (ActionsPolicyDeclaration, error) {
	days, _, err := o.Values()
	if err != nil {
		return ActionsPolicyDeclaration{}, err
	}
	if err := PrepareRetention(o, days); err != nil {
		return ActionsPolicyDeclaration{}, err
	}
	return ActionsPolicyDeclaration{RetentionDays: &days}, nil
}
func CompareRetention(o RetentionObservation, d ActionsPolicyDeclaration) (int, []ActionsPolicyChange, error) {
	if err := d.Validate(); err != nil {
		return 0, nil, err
	}
	before, _, err := o.Values()
	if err != nil {
		return 0, nil, err
	}
	changes := []ActionsPolicyChange{}
	if d.RetentionDays == nil {
		return 0, changes, nil
	}
	if err := PrepareRetention(o, *d.RetentionDays); err != nil {
		return 0, nil, err
	}
	if before != *d.RetentionDays {
		c, _ := NewActionsPolicyChange("actions.artifact_and_log_retention_days", before, *d.RetentionDays)
		changes = append(changes, c)
	}
	return 1, changes, nil
}
