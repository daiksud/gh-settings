package settings

import "fmt"

// RepositoryPatchChanges compares acquired native values for the shared REST
// PATCH. Menu observations must be raw: projected fallback equality alone does
// not establish that GitHub stores the requested canonical pair.
func RepositoryPatchChanges(observed, desired RepositoryDeclaration) (RepositoryDeclaration, error) {
	allowed := desired.AllowMergeCommit
	if allowed == nil {
		allowed = observed.AllowMergeCommit
	}
	if desired.MergeCommitTitle != nil || desired.MergeCommitMessage != nil {
		if allowed == nil {
			return RepositoryDeclaration{}, fmt.Errorf("repository observation is missing repository.allow_merge_commit")
		}
		if err := desired.validateMergeCommitDefaults(*allowed); err != nil {
			return RepositoryDeclaration{}, err
		}
	}
	var changed RepositoryDeclaration
	if err := changedPatchFields(&changed, repositoryStringFields[:], observed, desired); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := changedPatchFields(&changed, repositoryBoolFields[:], observed, desired); err != nil {
		return RepositoryDeclaration{}, err
	}
	if err := changedPatchFields(&changed, mergeMethodsBoolFields[:], observed, desired); err != nil {
		return RepositoryDeclaration{}, err
	}
	for _, menu := range []struct {
		beforeTitle, beforeMessage, title, message *string
		changedTitle, changedMessage               **string
	}{
		{observed.SquashMergeCommitTitle, observed.SquashMergeCommitMessage, desired.SquashMergeCommitTitle, desired.SquashMergeCommitMessage, &changed.SquashMergeCommitTitle, &changed.SquashMergeCommitMessage},
		{observed.MergeCommitTitle, observed.MergeCommitMessage, desired.MergeCommitTitle, desired.MergeCommitMessage, &changed.MergeCommitTitle, &changed.MergeCommitMessage},
	} {
		if menu.title == nil && menu.message == nil {
			continue
		}
		if menu.beforeTitle == nil || menu.beforeMessage == nil {
			return RepositoryDeclaration{}, fmt.Errorf("repository observation is missing a complete commit message pair")
		}
		if menu.title != nil && *menu.title != *menu.beforeTitle || menu.message != nil && *menu.message != *menu.beforeMessage {
			// A whole UI menu owns both members even if only one differs. An
			// internal partial declaration still owns only its declared member.
			if menu.title != nil {
				value := *menu.title
				*menu.changedTitle = &value
			}
			if menu.message != nil {
				value := *menu.message
				*menu.changedMessage = &value
			}
		}
	}
	return changed, nil
}

func changedPatchFields[S any, V bool | string](changed *RepositoryDeclaration, fields []repositoryField[S, V], observed, desired RepositoryDeclaration) error {
	for _, field := range fields {
		after := field.desired(desired)
		if after == nil {
			continue
		}
		before := field.desired(observed)
		if before == nil {
			return fmt.Errorf("repository observation is missing %s", field.name)
		}
		if *before != *after {
			field.setDesired(changed, *after)
		}
	}
	return nil
}
