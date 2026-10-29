package github

import (
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestSquashObservationNormalizationRetainsTypedPair(t *testing.T) {
	for _, test := range []struct {
		title, message, choice string
		fallback               bool
	}{
		{"COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES", "default_message", false},
		{"PR_TITLE", "BLANK", "pull_request_title", false},
		{"PR_TITLE", "COMMIT_MESSAGES", "pull_request_title_and_commit_details", false},
		{"PR_TITLE", "PR_BODY", "pull_request_title_and_description", false},
		{"COMMIT_OR_PR_TITLE", "BLANK", "default_message", true},
		{"FUTURE_TITLE", "PR_BODY", "default_message", true},
		{"PR_TITLE", "FUTURE_MESSAGE", "default_message", true},
		{"", "", "default_message", true},
	} {
		title, message := test.title, test.message
		result, err := prepareCommitMessageObservation(&title, &message, settings.SquashCommitMenu)
		if err != nil || result.Choice == nil || string(*result.Choice) != test.choice || result.Pair == nil {
			t.Fatal("complete squash observation was not projected", test, result, err)
		}
		wantPair := settings.CommitMessagePair{Title: test.title, Message: test.message}
		if test.fallback {
			wantPair = settings.CommitMessagePair{Title: "COMMIT_OR_PR_TITLE", Message: "COMMIT_MESSAGES"}
			if result.Normalization.Path != "repository.settings.general.pull_requests.squash_merge.default_commit_message" || result.Normalization.RawTitle != test.title || result.Normalization.RawMessage != test.message || result.Normalization.Fallback != "default_message" {
				t.Fatal("raw pair was lost or collapsed", result.Normalization)
			}
		} else if result.Normalization != (settings.CommitMessageNormalization{}) {
			t.Fatal("supported pair acquired fallback provenance", result.Normalization)
		}
		title, message = "changed title", "changed message"
		if *result.Pair != wantPair || test.fallback && (result.Normalization.RawTitle != test.title || result.Normalization.RawMessage != test.message) {
			t.Fatal("projection aliases source pair", result)
		}
	}
	if result, err := prepareCommitMessageObservation(nil, nil, settings.SquashCommitMenu); err != nil || result != (preparedCommitMessage{}) {
		t.Fatal("unacquired pair became a default", result, err)
	}
	title, message := "COMMIT_OR_PR_TITLE", "BLANK"
	for _, pair := range [][2]*string{{&title, nil}, {nil, &message}} {
		if result, err := prepareCommitMessageObservation(pair[0], pair[1], settings.SquashCommitMenu); err == nil || result != (preparedCommitMessage{}) {
			t.Fatal("incomplete pair became fallback", result, err)
		}
	}
	if result, err := prepareCommitMessageObservation(&title, &message, settings.CommitMessageMenu("future_menu")); err == nil || result != (preparedCommitMessage{}) {
		t.Fatal("unknown menu kind acquired an invented fallback", result, err)
	}
}

func TestMergeObservationNormalizationRetainsTypedPair(t *testing.T) {
	for _, test := range []struct {
		title, message, choice string
		fallback               bool
	}{
		{"MERGE_MESSAGE", "PR_TITLE", "default_message", false},
		{"PR_TITLE", "BLANK", "pull_request_title", false},
		{"PR_TITLE", "PR_BODY", "pull_request_title_and_description", false},
		{"MERGE_MESSAGE", "BLANK", "default_message", true},
		{"FUTURE_TITLE", "PR_BODY", "default_message", true},
		{"PR_TITLE", "FUTURE_MESSAGE", "default_message", true},
		{"", "", "default_message", true},
	} {
		title, message := test.title, test.message
		result, err := prepareCommitMessageObservation(&title, &message, settings.MergeCommitMenu)
		if err != nil || result.Choice == nil || string(*result.Choice) != test.choice || result.Pair == nil {
			t.Fatal("complete merge observation was not projected", test, result, err)
		}
		wantPair := settings.CommitMessagePair{Title: test.title, Message: test.message}
		if test.fallback {
			wantPair = settings.CommitMessagePair{Title: "MERGE_MESSAGE", Message: "PR_TITLE"}
			if result.Normalization.Path != "repository.settings.general.pull_requests.merge_commit.default_commit_message" || result.Normalization.RawTitle != test.title || result.Normalization.RawMessage != test.message || result.Normalization.Fallback != "default_message" {
				t.Fatal("merge raw pair was lost or used squash fallback", result.Normalization)
			}
		} else if result.Normalization != (settings.CommitMessageNormalization{}) {
			t.Fatal("supported merge pair acquired fallback provenance", result.Normalization)
		}
		if *result.Pair != wantPair {
			t.Fatal("merge effective pair changed", result.Pair, wantPair)
		}
	}
}
