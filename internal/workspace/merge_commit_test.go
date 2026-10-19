package workspace

import (
	"reflect"
	"strings"
	"testing"
)

func TestMergeCommitCodec(t *testing.T) {
	for _, title := range []string{"PR_TITLE", "MERGE_MESSAGE"} {
		for _, message := range []string{"", "PR_BODY", "PR_TITLE", "BLANK"} {
			input := "schemaVersion: 1\nsettings:\n  repository:\n    merge_commit_title: " + title + "\n"
			if message != "" {
				input += "    merge_commit_message: " + message + "\n"
			}
			d, err := DecodeConfig([]byte(input))
			if err != nil {
				t.Fatal(err)
			}
			if d.Repository.Empty() || !d.Repository.BaseEmpty() || d.Repository.MergeMethodsEmpty() || *d.Repository.MergeCommitTitle != title {
				t.Fatalf("declaration=%+v", d)
			}
			if (d.Repository.MergeCommitMessage == nil) != (message == "") {
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
			if strings.Contains(string(data), "allow_squash_merge") || (message == "" && strings.Contains(string(data), "merge_commit_message")) {
				t.Fatalf("injected default: %s", data)
			}
		}
	}
	for _, field := range []string{"merge_commit_title", "merge_commit_message"} {
		for _, value := range []string{"null", "", "''", "true", "0", "[]", "{}", "pr_title", "UNKNOWN", "COMMIT_OR_PR_TITLE", "' PR_TITLE'", "!!bool PR_TITLE"} {
			prefix := ""
			if field == "merge_commit_message" {
				prefix = "merge_commit_title: PR_TITLE, "
			}
			d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {" + prefix + field + ": " + value + "}}"))
			if err == nil || !d.Empty() {
				t.Fatalf("accepted %s=%s: %+v %v", field, value, d, err)
			}
		}
	}
	for _, message := range []string{"PR_BODY", "PR_TITLE", "BLANK"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {merge_commit_message: " + message + "}}")); err == nil || !strings.Contains(err.Error(), "requires merge_commit_title") {
			t.Fatalf("message-only=%v", err)
		}
	}
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings:\n  repository:\n    <<: &defaults {merge_commit_title: PR_TITLE}\n    merge_commit_message: &message BLANK\n"))
	if err != nil || *d.Repository.MergeCommitMessage != "BLANK" {
		t.Fatalf("merge=%+v %v", d, err)
	}
}
