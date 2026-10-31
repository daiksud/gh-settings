package settings

// Comparison counts all declared fields and records only changed values.
type Comparison struct {
	Managed                 int
	Changes                 []Change
	CommitMessages          []NormalizedCommitMessageChange
	CommitMessageContexts   []CommitMessageContext
	Labels                  LabelChanges
	Rulesets                []RulesetChange
	Environments            []EnvironmentChange
	Actions                 []ActionsPolicyChange
	ActionsExecution        []ActionsExecutionChange
	ActionsExecutionContext *ActionsExecutionContext
	Security                []SecurityChange
	Pages                   []PagesChange
}

// HasChanges includes actionable normalized fallback matches, not only unequal
// effective values. Observation validation remains the producing operation's job.
func (comparison Comparison) HasChanges() bool {
	return len(comparison.Changes) > 0 || len(comparison.CommitMessages) > 0 || comparison.Labels.Len() > 0 || len(comparison.Rulesets) > 0 || len(comparison.Environments) > 0 || len(comparison.Actions) > 0 || len(comparison.ActionsExecution) > 0 || len(comparison.Security) > 0 || len(comparison.Pages) > 0
}

// Change contains a supported field identifier and a matching typed value pair.
type Change struct {
	field         FieldID
	values        changeValues
	normalization ObservationNormalization
}

// Field returns the validated field identity without exposing a mutable path.
func (change Change) Field() FieldID { return change.field }

func (change Change) Normalization() ObservationNormalization { return change.normalization }

// CompareRepository compares the REST-observed base fields without rendering them.
func CompareRepository(current RepositoryState, desired RepositoryDeclaration) Comparison {
	var result Comparison
	compareRepositoryFields(&result, repositoryStringFields[:], current, desired, stringChange)
	compareRepositoryFields(&result, repositoryBoolFields[:], current, desired, boolChange)
	return result
}

func compareRepositoryFields[S any, V bool | string](result *Comparison, fields []repositoryField[S, V], current S, desired RepositoryDeclaration, change func(FieldID, V, V) Change) {
	for _, field := range fields {
		after := field.desired(desired)
		if after == nil {
			continue
		}
		result.Managed++
		before := field.current(current)
		if before != *after {
			result.Changes = append(result.Changes, change(FieldID{path: field.name}, before, *after))
		}
	}
}
