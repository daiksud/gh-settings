package settings

import (
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestRepositoryObservationRetainsCommitMenuSidecars(t *testing.T) {
	squash := CommitMessageNormalization{Path: "repository.settings.general.pull_requests.squash_merge.default_commit_message", RawTitle: "RAW_SQUASH_TITLE", RawMessage: "RAW_SQUASH_MESSAGE", Fallback: uimodel.DefaultMessage}
	merge := CommitMessageNormalization{Path: "repository.settings.general.pull_requests.merge_commit.default_commit_message", RawTitle: "RAW_MERGE_TITLE", RawMessage: "RAW_MERGE_MESSAGE", Fallback: uimodel.DefaultMessage}
	source := MergeMethodsState{AllowMergeCommit: true, AllowSquashMerge: true, MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE", SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", SquashCommitNormalization: squash, MergeCommitNormalization: merge}
	observed := NewRepositoryObservation(&RepositoryState{DefaultBranch: "main"}, &source, &PRAccessState{PullRequestCreationPolicy: "ALL"}).WithIssueCreationPolicy(IssueCreationPolicyState{Policy: "ALL"}).WithSponsorships(SponsorshipsState{Enabled: false})
	source.SquashCommitNormalization.RawTitle = "changed source"
	source.MergeCommitNormalization.RawMessage = "changed source"
	source.MergeCommitTitle = "changed effective source"
	records := observed.CommitMessageNormalizations()
	if len(records) != 2 || records[0] != squash || records[1] != merge {
		t.Fatal("paired raw provenance lost or duplicated", records)
	}
	records[0].RawTitle = "changed caller record"
	if again := observed.CommitMessageNormalizations(); len(again) != 2 || again[0] != squash || again[1] != merge {
		t.Fatal("caller mutation changed observation provenance", again)
	}
	declaration, err := observed.Declaration()
	if err != nil || declaration.SquashMergeCommitTitle == nil || *declaration.SquashMergeCommitTitle != "COMMIT_OR_PR_TITLE" || declaration.SquashMergeCommitMessage == nil || *declaration.SquashMergeCommitMessage != "COMMIT_MESSAGES" || declaration.MergeCommitTitle == nil || *declaration.MergeCommitTitle != "MERGE_MESSAGE" || declaration.MergeCommitMessage == nil || *declaration.MergeCommitMessage != "PR_TITLE" {
		t.Fatal("effective native declaration changed or borrowed raw provenance", declaration, err)
	}
	err = VisitDeclaredRepositoryFields(declaration, func(field, value string) error {
		if strings.HasPrefix(value, "RAW_") {
			t.Fatal("raw provenance became a declared request member", field, value)
		}
		return nil
	}, func(string, bool) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestRepositoryObservationWithoutCommitMenuProvenance(t *testing.T) {
	for _, source := range []*MergeMethodsState{nil, {}} {
		if got := NewRepositoryObservation(nil, source, nil).CommitMessageNormalizations(); len(got) != 0 {
			t.Fatal("ordinary or unacquired group acquired provenance", got)
		}
	}
	empty := CommitMessageNormalization{Path: "repository.settings.general.pull_requests.squash_merge.default_commit_message", Fallback: uimodel.DefaultMessage}
	observed := NewRepositoryObservation(nil, &MergeMethodsState{SquashCommitNormalization: empty}, nil)
	if got := observed.CommitMessageNormalizations(); len(got) != 1 || got[0] != empty {
		t.Fatal("acquired empty raw pair was dropped", got)
	}
}
