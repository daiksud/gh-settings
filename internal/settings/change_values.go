package settings

import (
	"fmt"
	"slices"
)

type changeVisitor struct {
	boolean func(bool, bool)
	text    func(string, string)
	topics  func([]string, []string)
}

type changeValues interface{ visit(changeVisitor) }

type boolPair struct{ before, after bool }
type stringPair struct{ before, after string }
type topicsPair struct{ before, after []string }

func (pair boolPair) visit(visitor changeVisitor)   { visitor.boolean(pair.before, pair.after) }
func (pair stringPair) visit(visitor changeVisitor) { visitor.text(pair.before, pair.after) }
func (pair topicsPair) visit(visitor changeVisitor) {
	visitor.topics(slices.Clone(pair.before), slices.Clone(pair.after))
}

// NewBoolChange requires a boolean field and a native boolean pair.
func NewBoolChange(field FieldID, before, after bool) (Change, error) {
	if _, ok := RepositoryBoolSetter(field.path); !ok {
		return Change{}, fmt.Errorf("comparison field %q does not accept boolean values", field.path)
	}
	return boolChange(field, before, after), nil
}

func boolChange(field FieldID, before, after bool) Change {
	return Change{field: field, values: boolPair{before: before, after: after}}
}

// NewStringChange requires a string field and values allowed by its definition.
func NewStringChange(field FieldID, before, after string) (Change, error) {
	if !ValidRepositoryFieldValue(field.path, before) || !ValidRepositoryFieldValue(field.path, after) {
		return Change{}, fmt.Errorf("invalid string comparison for %q", field.path)
	}
	return stringChange(field, before, after), nil
}

func stringChange(field FieldID, before, after string) Change {
	return Change{field: field, values: stringPair{before: before, after: after}}
}

// NewTopicsChange validates and copies both topic sets into canonical order.
func NewTopicsChange(before, after []string) (Change, error) {
	if err := ValidateTopics(before); err != nil {
		return Change{}, err
	}
	if err := ValidateTopics(after); err != nil {
		return Change{}, err
	}
	return Change{field: topicsField, values: topicsPair{before: SortedTopics(before), after: SortedTopics(after)}}, nil
}

// Visit calls exactly one callback with a native before/after pair.
// Topic slices are independent copies, including explicitly empty lists.
func (change Change) Visit(boolean func(bool, bool), text func(string, string), topics func([]string, []string)) {
	change.values.visit(changeVisitor{boolean: boolean, text: text, topics: topics})
}
