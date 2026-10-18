package workspace

import (
	"strings"
	"testing"
)

func TestMergeMethodBooleanCodec(t *testing.T) {
	for _, field := range []string{"allow_merge_commit", "allow_squash_merge", "allow_rebase_merge", "delete_branch_on_merge", "allow_auto_merge", "allow_update_branch"} {
		for _, value := range []string{"true", "false", "TRUE", "!!bool false", "&value false"} {
			t.Run(field+value, func(t *testing.T) {
				d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings:\n  repository:\n    " + field + ": " + value + "\n"))
				if err != nil || d.Repository.Empty() || !d.Repository.BaseEmpty() {
					t.Fatalf("desired=%+v err=%v", d, err)
				}
				data, err := EncodeConfig(d)
				if err != nil || !strings.Contains(string(data), field+": ") {
					t.Fatalf("data=%s err=%v", data, err)
				}
				round, err := DecodeConfig(data)
				if err != nil || round.Repository.MergeMethodsEmpty() {
					t.Fatalf("round=%+v err=%v", round, err)
				}
				for _, other := range []string{"allow_merge_commit", "allow_squash_merge", "allow_rebase_merge", "delete_branch_on_merge", "allow_auto_merge", "allow_update_branch"} {
					if other != field && strings.Contains(string(data), other+":") {
						t.Fatalf("injected default: %s", data)
					}
				}
			})
		}
		for _, value := range []string{"null", "", "'false'", "0", "[]", "{}", "yes", "!!bool yes"} {
			t.Run(field+value, func(t *testing.T) {
				d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {" + field + ": " + value + "}}"))
				if err == nil || !d.Empty() {
					t.Fatalf("accepted %s: %+v %v", value, d, err)
				}
			})
		}
	}
	for _, field := range []string{"allow_forking", "allow_merge_queue", "squash_merge_commit_title", "merge_commit_message"} {
		if _, err := DecodeConfig([]byte("schemaVersion: 1\nsettings: {repository: {" + field + ": true}}")); err == nil {
			t.Fatalf("expanded scope: %s", field)
		}
	}
	d, err := DecodeConfig([]byte("schemaVersion: 1\nsettings:\n  repository:\n    <<: &methods {allow_merge_commit: false, allow_rebase_merge: true}\n    allow_squash_merge: false\n"))
	if err != nil || *d.Repository.AllowMergeCommit || *d.Repository.AllowSquashMerge || !*d.Repository.AllowRebaseMerge {
		t.Fatalf("alias merge=%+v err=%v", d, err)
	}
}
