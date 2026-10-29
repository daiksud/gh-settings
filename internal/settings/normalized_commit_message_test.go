package settings

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestNormalizedCommitMenuDecisionOwnsBothNativeFields(t *testing.T) {
	for _, test := range []struct {
		menu                       CommitMessageMenu
		path, rawTitle, rawMessage string
		titleField, messageField   string
	}{
		{SquashCommitMenu, "repository.settings.general.pull_requests.squash_merge.default_commit_message", "COMMIT_OR_PR_TITLE", "BLANK", "repository.squash_merge_commit_title", "repository.squash_merge_commit_message"},
		{MergeCommitMenu, "repository.settings.general.pull_requests.merge_commit.default_commit_message", "MERGE_MESSAGE", "BLANK", "repository.merge_commit_title", "repository.merge_commit_message"},
	} {
		for _, desired := range []uimodel.CommitMessageChoice{uimodel.DefaultMessage, uimodel.PullRequestTitle} {
			normalization := CommitMessageNormalization{Path: test.path, RawTitle: test.rawTitle, RawMessage: test.rawMessage, Fallback: uimodel.DefaultMessage}
			decision, err := NewNormalizedCommitMessageChange(test.menu, normalization, desired)
			fields := decision.NativeFields()
			if err != nil || decision.Menu() != test.menu || decision.Normalization() != normalization || len(fields) != 2 || fields[0].Path() != test.titleField || fields[1].Path() != test.messageField {
				t.Fatal("one normalized menu lost paired ownership/provenance", decision, fields, err)
			}
			before, after := decision.Values()
			if before != uimodel.DefaultMessage || after != desired {
				t.Fatal("menu-level comparison changed UI meaning", before, after)
			}
			normalization.RawTitle = "changed source"
			if decision.Normalization().RawTitle != test.rawTitle {
				t.Fatal("decision aliases incoming provenance")
			}
			fields[0] = FieldID{}
			if decision.NativeFields()[0].Path() != test.titleField {
				t.Fatal("caller mutation changed native field ownership")
			}
		}
	}
	empty := CommitMessageNormalization{Path: "repository.settings.general.pull_requests.squash_merge.default_commit_message", Fallback: uimodel.DefaultMessage}
	if decision, err := NewNormalizedCommitMessageChange(SquashCommitMenu, empty, uimodel.DefaultMessage); err != nil || decision.Normalization() != empty {
		t.Fatal("acquired empty pair treated as missing", decision, err)
	}
}

func TestNormalizedCommitMenuDecisionRejectsForgedProvenance(t *testing.T) {
	valid := CommitMessageNormalization{Path: "repository.settings.general.pull_requests.squash_merge.default_commit_message", RawTitle: "COMMIT_OR_PR_TITLE", RawMessage: "BLANK", Fallback: uimodel.DefaultMessage}
	for _, test := range []struct {
		menu          CommitMessageMenu
		normalization CommitMessageNormalization
		desired       uimodel.CommitMessageChoice
	}{
		{CommitMessageMenu("future_menu"), valid, uimodel.DefaultMessage},
		{MergeCommitMenu, valid, uimodel.DefaultMessage},
		{SquashCommitMenu, CommitMessageNormalization{}, uimodel.DefaultMessage},
		{SquashCommitMenu, CommitMessageNormalization{Path: "unmanaged", RawTitle: valid.RawTitle, RawMessage: valid.RawMessage, Fallback: valid.Fallback}, uimodel.DefaultMessage},
		{SquashCommitMenu, CommitMessageNormalization{Path: valid.Path, RawTitle: valid.RawTitle, RawMessage: valid.RawMessage, Fallback: uimodel.PullRequestTitle}, uimodel.DefaultMessage},
		{SquashCommitMenu, CommitMessageNormalization{Path: valid.Path, RawTitle: "COMMIT_OR_PR_TITLE", RawMessage: "COMMIT_MESSAGES", Fallback: valid.Fallback}, uimodel.DefaultMessage},
		{SquashCommitMenu, CommitMessageNormalization{Path: valid.Path, RawTitle: "PR_TITLE", RawMessage: "BLANK", Fallback: valid.Fallback}, uimodel.DefaultMessage},
		{SquashCommitMenu, valid, uimodel.CommitMessageChoice("FUTURE_CHOICE")},
		{MergeCommitMenu, CommitMessageNormalization{Path: "repository.settings.general.pull_requests.merge_commit.default_commit_message", RawTitle: "MERGE_MESSAGE", RawMessage: "BLANK", Fallback: uimodel.DefaultMessage}, uimodel.PullRequestTitleAndCommitDetails},
	} {
		if decision, err := NewNormalizedCommitMessageChange(test.menu, test.normalization, test.desired); err == nil {
			t.Fatalf("forged normalized menu accepted: %+v %+v", test, decision)
		}
	}
}
