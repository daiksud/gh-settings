package workspace

import (
	"github.com/daiksud/gh-settings/internal/github"
	"strings"
	"testing"
)

func TestPullRequestLimitsYAML(t *testing.T) {
	prefix := "version: daiksud/gh-settings@v0\nrepository: {settings: {interaction_limits: {pull_request_limits: "
	for _, body := range []string{`{limit_open_pull_requests_from_users_without_write_access: false}`, `{maximum_open_pull_requests_per_user: 1}`, `{maximum_open_pull_requests_per_user: 1000}`, `{count_draft_pull_requests_toward_the_limit: false}`, `{}`} {
		ui, err := DecodeConfig([]byte(prefix + body + "}}}\n"))
		if err != nil {
			t.Fatal(body, err)
		}
		d, err := github.DeclarationFromUI(ui)
		if err != nil {
			t.Fatal(body, err)
		}
		if d.Empty() != (body == "{}") {
			t.Fatal(d)
		}
		encoded, err := EncodeConfig(ui)
		if err != nil || !strings.Contains(string(encoded), "pull_request_limits:") {
			t.Fatal(string(encoded), err)
		}
	}
	for _, body := range []string{`null`, `false`, `{enabled: false}`, `{max_open_pull_requests: 2}`, `{maximum_open_pull_requests_per_user: 0}`, `{maximum_open_pull_requests_per_user: 1001}`, `{maximum_open_pull_requests_per_user: '1'}`, `{maximum_open_pull_requests_per_user: 1.5}`, `{count_draft_pull_requests_toward_the_limit: null}`, `{limit_open_pull_requests_from_users_without_write_access: 'false'}`} {
		ui, err := DecodeConfig([]byte(prefix + body + "}}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("accepted", body)
		}
	}
}
