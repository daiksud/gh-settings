package settings

// RepositoryAccessChanges compares only declared access controls. Observed
// policies must retain raw acquired strings: a projected fallback is not evidence
// that GitHub stores the requested canonical value.
func RepositoryAccessChanges(observed, desired RepositoryDeclaration) (RepositoryDeclaration, error) {
	var changed RepositoryDeclaration
	if err := changedPatchFields(&changed, prAccessBoolFields[:], observed, desired); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := changedPatchFields(&changed, prAccessStringFields[:], observed, desired); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := changedPatchFields(&changed, issuePolicyStringFields[:], observed, desired); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := changedPatchFields(&changed, sponsorshipBoolFields[:], observed, desired); err != nil {
		return RepositoryDeclaration{}, err
	}
	return changed, nil
}
