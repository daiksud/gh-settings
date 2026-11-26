package workspace

import (
	"slices"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
)

func TestPullRequestLimitsYAML(t *testing.T) {
	prefix := "version: daiksud/gh-settings@v0\nrepository: {settings: {moderation: {interaction_limits: {pull_request_limits: "
	for _, body := range []string{`{limit_open_pull_requests_from_users_without_write_access: false}`, `{maximum_open_pull_requests_per_user: 1}`, `{maximum_open_pull_requests_per_user: 1000}`, `{count_draft_pull_requests_toward_the_limit: false}`, `{}`} {
		ui, err := DecodeConfig([]byte(prefix + body + "}}}}\n"))
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
		ui, err := DecodeConfig([]byte(prefix + body + "}}}}\n"))
		if err == nil {
			_, err = github.DeclarationFromUI(ui)
		}
		if err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestDisabledPullRequestLimitIgnoresOnlyPresentCompanions(t *testing.T) {
	const prefix = "version: daiksud/gh-settings@v0\nrepository: {settings: {moderation: {interaction_limits: {pull_request_limits: "
	const enabled = "limit_open_pull_requests_from_users_without_write_access"
	const maximum = "maximum_open_pull_requests_per_user"
	const drafts = "count_draft_pull_requests_toward_the_limit"
	const exemptions = "exemption_list"
	for _, tc := range []struct {
		name, fields string
		ignored      []string
	}{
		{"disabled alone", enabled + ": false", nil},
		{"wrong types and unknown login need no validation", exemptions + ": [private/login], " + drafts + ": null, " + maximum + ": nope, " + enabled + ": false", []string{maximum, drafts, exemptions}},
		{"empty exemption list", exemptions + ": [], " + enabled + ": false", []string{exemptions}},
		{"explicit switch overrides merged switch", "<<: {" + enabled + ": true, " + maximum + ": 0}, " + enabled + ": FALSE", []string{maximum}},
		{"aliased switch", drafts + ": &off false, " + enabled + ": *off", []string{drafts}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ui, err := DecodeConfig([]byte(prefix + "{" + tc.fields + "}}}}}"))
			if err != nil {
				t.Fatal(err)
			}
			d, err := github.DeclarationFromUI(ui)
			if err != nil {
				t.Fatal(err)
			}
			limits := d.PullRequestLimits
			if limits.Enabled == nil || *limits.Enabled || limits.Maximum != nil || limits.IncludeDrafts != nil || d.PullRequestExemptions != nil || !slices.Equal(ui.Settings.IgnoredPullRequestLimitKeys, tc.ignored) {
				t.Fatalf("unexpected ownership or ignored keys: %+v, exemptions %+v, ignored %v", limits, d.PullRequestExemptions, ui.Settings.IgnoredPullRequestLimitKeys)
			}
		})
	}
	for _, fields := range []string{
		enabled + ": false, typo: null",
		enabled + ": false, " + maximum + ": 1, " + maximum + ": 2",
		enabled + ": false, " + exemptions + ": {duplicate: 1, duplicate: 2}",
		"<<: {" + enabled + ": false, " + maximum + ": wrong}, " + enabled + ": true",
	} {
		if _, err := DecodeConfig([]byte(prefix + "{" + fields + "}}}}}")); err == nil {
			t.Errorf("accepted invalid fields: %s", fields)
		}
	}
}
