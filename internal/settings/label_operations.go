package settings

import (
	"fmt"
	"slices"
	"strings"
)

// LabelChange is one closed create, update, or delete operation with owned values.
type LabelChange struct{ operation labelOperation }

type labelOperation interface {
	kind() string
	name() string
	visit(labelOperationVisitor)
}

type labelOperationVisitor struct {
	create func(LabelDeclaration)
	update func(LabelState, LabelDeclaration)
	remove func(LabelState)
}

type labelCreate struct{ after LabelDeclaration }
type labelUpdate struct {
	before LabelState
	after  LabelDeclaration
}
type labelDelete struct{ before LabelState }

func (labelCreate) kind() string                     { return "create" }
func (labelUpdate) kind() string                     { return "update" }
func (labelDelete) kind() string                     { return "delete" }
func (op labelCreate) name() string                  { return op.after.Name }
func (op labelUpdate) name() string                  { return op.before.Name }
func (op labelDelete) name() string                  { return op.before.Name }
func (op labelCreate) visit(v labelOperationVisitor) { v.create(copyLabelDeclaration(op.after)) }
func (op labelUpdate) visit(v labelOperationVisitor) {
	v.update(op.before, copyLabelDeclaration(op.after))
}
func (op labelDelete) visit(v labelOperationVisitor) { v.remove(op.before) }

func NewLabelCreate(after LabelDeclaration) (LabelChange, error) {
	if err := validateLabelAfter(after); err != nil {
		return LabelChange{}, err
	}
	return LabelChange{operation: labelCreate{after: copyLabelDeclaration(after)}}, nil
}

func NewLabelUpdate(before LabelState, after LabelDeclaration) (LabelChange, error) {
	if err := validateLabelBefore(before); err != nil {
		return LabelChange{}, err
	}
	if after.Name != before.Name {
		return LabelChange{}, fmt.Errorf("label operation cannot rename")
	}
	if err := validateLabelAfter(after); err != nil {
		return LabelChange{}, err
	}
	if (after.Color == nil && after.Description == nil) || (after.Color != nil && *after.Color == before.Color) || (after.Description != nil && *after.Description == before.Description) {
		return LabelChange{}, fmt.Errorf("invalid %q label operation", "update")
	}
	return LabelChange{operation: labelUpdate{before: before, after: copyLabelDeclaration(after)}}, nil
}

func NewLabelDelete(before LabelState) (LabelChange, error) {
	if err := validateLabelBefore(before); err != nil {
		return LabelChange{}, err
	}
	return LabelChange{operation: labelDelete{before: before}}, nil
}

func validateLabelBefore(before LabelState) error {
	if err := ValidateLabelName(before.Name); err != nil {
		return err
	}
	if before.Color != strings.ToLower(before.Color) {
		return fmt.Errorf("invalid label before value")
	}
	if err := ValidateLabelColor(before.Color); err != nil {
		return err
	}
	return ValidateLabelDescription(before.Description)
}

func validateLabelAfter(after LabelDeclaration) error {
	if err := ValidateLabels([]LabelDeclaration{after}); err != nil {
		return err
	}
	if after.Color != nil && *after.Color != strings.ToLower(*after.Color) {
		return fmt.Errorf("invalid label after color")
	}
	return nil
}

func (change LabelChange) Operation() string { return change.operation.kind() }
func (change LabelChange) Name() string      { return change.operation.name() }

// Visit calls exactly one callback with independent attribute values.
func (change LabelChange) Visit(create func(LabelDeclaration), update func(LabelState, LabelDeclaration), remove func(LabelState)) {
	change.operation.visit(labelOperationVisitor{create: create, update: update, remove: remove})
}

// LabelChanges retains operation order and at most one operation per identity.
// Its zero value is a valid empty sequence.
type LabelChanges struct{ entries []LabelChange }

func NewLabelChanges(changes []LabelChange) (LabelChanges, error) {
	seen := make(map[string]bool, len(changes))
	for _, change := range changes {
		if change.operation == nil {
			return LabelChanges{}, fmt.Errorf("missing label operation")
		}
		key := LabelIdentity(change.Name())
		if seen[key] {
			return LabelChanges{}, fmt.Errorf("duplicate label operation %q", change.Name())
		}
		seen[key] = true
	}
	return LabelChanges{entries: slices.Clone(changes)}, nil
}

func (changes LabelChanges) Len() int { return len(changes.entries) }
func (changes LabelChanges) Values() []LabelChange {
	return append([]LabelChange{}, changes.entries...)
}
