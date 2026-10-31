package settings

import "fmt"

// RepositoryObservation retains acquired scalar values independently of omission
// in a desired declaration. A missing observation is never an observed zero value.
type RepositoryObservation struct {
	values                     RepositoryDeclaration
	squashCommitNormalization  CommitMessageNormalization
	mergeCommitNormalization   CommitMessageNormalization
	prCreationNormalization    ObservationNormalization
	issueCreationNormalization ObservationNormalization
}

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
	observation := RepositoryObservation{values: values}
	if merge != nil {
		observation.squashCommitNormalization = merge.SquashCommitNormalization
		observation.mergeCommitNormalization = merge.MergeCommitNormalization
	}
	if pr != nil {
		observation.prCreationNormalization = pr.Normalization
	}
	return observation
}

// CommitMessageNormalizations returns acquired provenance in scalar-field order.
// It remains separate from declarations and owns the returned records.
func (observation RepositoryObservation) CommitMessageNormalizations() []CommitMessageNormalization {
	var records []CommitMessageNormalization
	for _, normalization := range [...]CommitMessageNormalization{observation.squashCommitNormalization, observation.mergeCommitNormalization} {
		if normalization != (CommitMessageNormalization{}) {
			records = append(records, normalization)
		}
	}
	return records
}

// CreationPolicyNormalizations returns owned records in native field order.
func (observation RepositoryObservation) CreationPolicyNormalizations() []ObservationNormalization {
	var records []ObservationNormalization
	for _, normalization := range [...]ObservationNormalization{observation.prCreationNormalization, observation.issueCreationNormalization} {
		if normalization != (ObservationNormalization{}) {
			records = append(records, normalization)
		}
	}
	return records
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
func RepositoryObservationGroups(fields []FieldID) (base, merge, pr, issue, sponsorship bool, err error) {
	for _, field := range fields {
		switch {
		case containsRepositoryField(repositoryStringFields[:], field), containsRepositoryField(repositoryBoolFields[:], field):
			base = true
		case containsRepositoryField(mergeMethodsBoolFields[:], field), containsRepositoryField(mergeMethodsStringFields[:], field):
			merge = true
		case containsRepositoryField(prAccessBoolFields[:], field), containsRepositoryField(prAccessStringFields[:], field):
			pr = true
		case containsRepositoryField(issuePolicyStringFields[:], field):
			issue = true
		case containsRepositoryField(sponsorshipBoolFields[:], field):
			sponsorship = true
		default:
			return false, false, false, false, false, fmt.Errorf("field %q is not a supported repository observation", field.path)
		}
	}
	return base, merge, pr, issue, sponsorship, nil
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
	allowMergeCommit := desired.AllowMergeCommit
	if allowMergeCommit == nil {
		allowMergeCommit = observed.values.AllowMergeCommit
	}
	if allowMergeCommit != nil {
		if err := desired.validateMergeCommitDefaults(*allowMergeCommit); err != nil {
			return Comparison{}, err
		}
	}
	result, remaining, err := compareNormalizedCommitMessages(observed, desired)
	if err != nil {
		return Comparison{}, err
	}
	desired = remaining
	result.CommitMessageContexts, err = commitMessageContexts(observed.values, desired)
	if err != nil {
		return Comparison{}, err
	}
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
	if err := compareObservedCreationPolicy(&result, prAccessStringFields[:], observed.values, desired, observed.prCreationNormalization); err != nil {
		return Comparison{}, err
	}
	if err := compareObservedCreationPolicy(&result, issuePolicyStringFields[:], observed.values, desired, observed.issueCreationNormalization); err != nil {
		return Comparison{}, err
	}
	if err := compareObservedFields(&result, sponsorshipBoolFields[:], observed.values, desired, boolChange); err != nil {
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
	fields = appendRepositoryFieldIDs(fields, prAccessStringFields[:])
	fields = appendRepositoryFieldIDs(fields, issuePolicyStringFields[:])
	return appendRepositoryFieldIDs(fields, sponsorshipBoolFields[:])
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
	if result.AllowMergeCommit != nil && !*result.AllowMergeCommit {
		result.MergeCommitTitle = nil
		result.MergeCommitMessage = nil
	}
	if err := copyObservedFields(&result, prAccessBoolFields[:], observation.values); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := copyObservedFields(&result, prAccessStringFields[:], observation.values); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := copyObservedFields(&result, issuePolicyStringFields[:], observation.values); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := copyObservedFields(&result, sponsorshipBoolFields[:], observation.values); err != nil {
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

// WithIssueCreationPolicy returns a copied observation with the acquired policy.
func (observation RepositoryObservation) WithIssueCreationPolicy(state IssueCreationPolicyState) RepositoryObservation {
	declareRepositoryFields(issuePolicyStringFields[:], state, &observation.values)
	observation.issueCreationNormalization = state.Normalization
	return observation
}
