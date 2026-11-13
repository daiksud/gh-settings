package settings

import "fmt"

// RepositoryExportGroup is one complete, independently readable observation.
// Export can retain another group's declaration without treating it as observed.
type RepositoryExportGroup uint8

const (
	RepositoryExportBase RepositoryExportGroup = iota
	RepositoryExportMergeMethods
	RepositoryExportPullRequests
	RepositoryExportIssuePolicy
	RepositoryExportSponsorships
)

// RepositoryExportGroups follows the repository adapter's acquisition order.
func RepositoryExportGroups() []RepositoryExportGroup {
	return []RepositoryExportGroup{RepositoryExportBase, RepositoryExportMergeMethods, RepositoryExportPullRequests, RepositoryExportIssuePolicy, RepositoryExportSponsorships}
}

// Fields requests every field owned by this group, without archive lifecycle.
func (group RepositoryExportGroup) Fields() []FieldID {
	var fields []FieldID
	switch group {
	case RepositoryExportBase:
		fields = appendRepositoryFieldIDs(fields, repositoryStringFields[:])
		return appendRepositoryFieldIDs(fields, repositoryBoolFields[:])
	case RepositoryExportMergeMethods:
		fields = appendRepositoryFieldIDs(fields, mergeMethodsBoolFields[:])
		return appendRepositoryFieldIDs(fields, mergeMethodsStringFields[:])
	case RepositoryExportPullRequests:
		fields = appendRepositoryFieldIDs(fields, prAccessBoolFields[:])
		return appendRepositoryFieldIDs(fields, prAccessStringFields[:])
	case RepositoryExportIssuePolicy:
		return appendRepositoryFieldIDs(fields, issuePolicyStringFields[:])
	case RepositoryExportSponsorships:
		return appendRepositoryFieldIDs(fields, sponsorshipBoolFields[:])
	default:
		return nil
	}
}

// ExportDeclaration overlays only a completely acquired group's native values.
// Unlike Declaration, other groups may be absent. It never manufactures defaults
// or imports extra values returned outside the requested group.
func (observation RepositoryObservation) ExportDeclaration(group RepositoryExportGroup, previous RepositoryDeclaration) (RepositoryDeclaration, error) {
	result := previous
	var err error
	switch group {
	case RepositoryExportBase:
		err = copyObservedFields(&result, repositoryStringFields[:], observation.values)
		if err == nil {
			err = copyObservedFields(&result, repositoryBoolFields[:], observation.values)
		}
	case RepositoryExportMergeMethods:
		err = copyObservedFields(&result, mergeMethodsBoolFields[:], observation.values)
		if err == nil {
			err = copyObservedFields(&result, mergeMethodsStringFields[:], observation.values)
		}
	case RepositoryExportPullRequests:
		err = copyObservedFields(&result, prAccessBoolFields[:], observation.values)
		if err == nil {
			err = copyObservedFields(&result, prAccessStringFields[:], observation.values)
		}
	case RepositoryExportIssuePolicy:
		err = copyObservedFields(&result, issuePolicyStringFields[:], observation.values)
	case RepositoryExportSponsorships:
		err = copyObservedFields(&result, sponsorshipBoolFields[:], observation.values)
	default:
		err = fmt.Errorf("unknown repository export group")
	}
	if err != nil {
		return RepositoryDeclaration{}, err
	}
	return result, nil
}
