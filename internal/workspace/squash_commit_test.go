package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestSquashCommitCodec(t *testing.T) {
	for _, title := range []string{"PR_TITLE", "COMMIT_OR_PR_TITLE"} {
		for _, message := range []string{"", "PR_BODY", "COMMIT_MESSAGES", "BLANK"} {
			input := "schemaVersion: 1\nsettings:\n  repository:\n    squash_merge_commit_title: " + title + "\n"
			if message != "" {
				input += "    squash_merge_commit_message: " + message + "\n"
			}
			d, err := DecodeConfig([]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			if d.Repository.Empty() || !d.Repository.BaseEmpty() || d.Repository.MergeMethodsEmpty() || *d.Repository.SquashMergeCommitTitle != title {
				t.Fatalf("declaration=%+v", d)
			}
			if (d.Repository.SquashMergeCommitMessage == nil) != (message == "") {
				t.Fatal("omission changed")
			}
			data, err := EncodeConfig(d)
			if err != nil {
				t.Fatal(err)
			}
			round, err := DecodeConfig(data)
			if err != nil || !reflect.DeepEqual(d, round) {
				t.Fatalf("round=%+v err=%v", round, err)
			}
			if strings.Contains(string(data), "allow_squash_merge") || (message == "" && strings.Contains(string(data), "squash_merge_commit_message")) {
				t.Fatalf("injected default: %s", data)
			}
		}
	}
	for _, field := range []string{"squash_merge_commit_title", "squash_merge_commit_message"} {
		for _, value := range []string{"null", "", "''", "true", "0", "[]", "{}", "pr_title", "UNKNOWN", "' PR_TITLE'", "!!bool PR_TITLE"} {
			prefix := ""
			if field == "squash_merge_commit_message" {
				prefix = "squash_merge_commit_title: PR_TITLE, "
			}
			d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {" + prefix + field + ": " + value + "}}"))
			if err == nil || !d.Empty() {
				t.Fatalf("accepted %s=%s: %+v %v", field, value, d, err)
			}
		}
	}
	for _, message := range []string{"PR_BODY", "COMMIT_MESSAGES", "BLANK"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {squash_merge_commit_message: " + message + "}}")); err == nil || !strings.Contains(err.Error(), "requires squash_merge_commit_title") {
			t.Fatalf("message-only=%v", err)
		}
	}
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings:\n  repository:\n    <<: &defaults {squash_merge_commit_title: PR_TITLE}\n    squash_merge_commit_message: &message BLANK\n"))
	if err != nil || *d.Repository.SquashMergeCommitMessage != "BLANK" {
		t.Fatalf("merge=%+v %v", d, err)
	}
}
