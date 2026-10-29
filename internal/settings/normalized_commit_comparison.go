package settings

import "fmt"

// compareNormalizedCommitMessages removes only validated whole-menu ownership
// from a local intent copy. Ordinary component comparisons keep their contract.
func compareNormalizedCommitMessages(observed RepositoryObservation, desired RepositoryDeclaration) (Comparison, RepositoryDeclaration, error) {
	var result Comparison
	remaining := desired
	for _, entry := range []struct {
		menu                       CommitMessageMenu
		normalization              CommitMessageNormalization
		beforeTitle, beforeMessage *string
		afterTitle, afterMessage   *string
	}{
		{SquashCommitMenu, observed.squashCommitNormalization, observed.values.SquashMergeCommitTitle, observed.values.SquashMergeCommitMessage, desired.SquashMergeCommitTitle, desired.SquashMergeCommitMessage},
		{MergeCommitMenu, observed.mergeCommitNormalization, observed.values.MergeCommitTitle, observed.values.MergeCommitMessage, desired.MergeCommitTitle, desired.MergeCommitMessage},
	} {
		if entry.normalization == (CommitMessageNormalization{}) || entry.afterTitle == nil && entry.afterMessage == nil {
			continue
		}
		if entry.beforeTitle == nil || entry.beforeMessage == nil || entry.afterTitle == nil || entry.afterMessage == nil {
			return Comparison{}, RepositoryDeclaration{}, fmt.Errorf("normalized %s comparison requires complete observed and desired pairs", entry.menu)
		}
		choice, valid := CommitMessageChoiceForPair(entry.menu, CommitMessagePair{Title: *entry.afterTitle, Message: *entry.afterMessage})
		if !valid {
			return Comparison{}, RepositoryDeclaration{}, fmt.Errorf("desired %s pair is outside UI choices", entry.menu)
		}
		decision, err := NewNormalizedCommitMessageChange(entry.menu, entry.normalization, choice)
		if err != nil {
			return Comparison{}, RepositoryDeclaration{}, err
		}
		fallback, valid := CommitMessagePairForChoice(entry.menu, entry.normalization.Fallback)
		if !valid || *entry.beforeTitle != fallback.Title || *entry.beforeMessage != fallback.Message {
			return Comparison{}, RepositoryDeclaration{}, fmt.Errorf("observed %s pair does not match its recorded UI fallback", entry.menu)
		}
		result.Managed += len(decision.NativeFields())
		result.CommitMessages = append(result.CommitMessages, decision)
		if entry.menu == SquashCommitMenu {
			remaining.SquashMergeCommitTitle, remaining.SquashMergeCommitMessage = nil, nil
		} else {
			remaining.MergeCommitTitle, remaining.MergeCommitMessage = nil, nil
		}
	}
	return result, remaining, nil
}
