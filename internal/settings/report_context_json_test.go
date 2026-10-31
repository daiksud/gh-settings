package settings

import (
	"encoding/json"
	"testing"
)

func TestReportContextJSONRequiresClosedCompleteValues(t *testing.T) {
	for _, data := range []string{
		`{"menu":"squash_merge","before":{"title":"COMMIT_OR_PR_TITLE","message":"COMMIT_MESSAGES"},"after":{"title":"PR_TITLE"}}`,
		`{"menu":"squash_merge","before":{"title":"COMMIT_OR_PR_TITLE","message":"COMMIT_MESSAGES","extra":true},"after":{"title":"PR_TITLE","message":"BLANK"}}`,
		`{"menu":"squash_merge","before":{"title":"COMMIT_OR_PR_TITLE","message":"COMMIT_MESSAGES"},"after":null}`,
		`{"menu":"squash_merge","before":{"title":"COMMIT_OR_PR_TITLE","message":"COMMIT_MESSAGES"},"after":{"title":"PR_TITLE","message":"BLANK"},"extra":true}`,
	} {
		var c CommitMessageContext
		if json.Unmarshal([]byte(data), &c) == nil {
			t.Fatalf("accepted %s", data)
		}
	}
	for _, data := range []string{`{"before":"disable_actions"}`, `{"before":"disable_actions","after":null}`, `{"before":"disable_actions","after":"allow_all_actions_and_reusable_workflows","observed_allowed":"all","extra":true}`} {
		var c ActionsExecutionContext
		if json.Unmarshal([]byte(data), &c) == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}
