package workspace

import (
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestModerationCanonicalGroupingPreservesControlsAndUIOrder(t *testing.T) {
	input := `version: daiksud/gh-settings@v0
repository:
  settings:
    rulesets: []
    moderation:
      interaction_limits:
        pull_request_limits:
          limit_open_pull_requests_from_users_without_write_access: true
          maximum_open_pull_requests_per_user: 1
          count_draft_pull_requests_toward_the_limit: false
          exemption_list: []
    collaborators: []
    general: {template_repository: false}
`
	ui, err := DecodeConfig([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	d, err := github.DeclarationFromUI(ui)
	if err != nil {
		t.Fatal(err)
	}
	o, _ := settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: true, Maximum: 1, IncludeDrafts: false})
	managed, changes, err := settings.ComparePullRequestLimits(o, d.PullRequestLimits)
	if err != nil || managed != 3 || len(changes) != 0 || d.PullRequestExemptions == nil || len(d.PullRequestExemptions.Logins()) != 0 {
		t.Fatalf("changed ownership: %#v %d %v %v", d, managed, changes, err)
	}
	encoded, err := EncodeConfig(ui)
	if err != nil {
		t.Fatal(err)
	}
	previous := -1
	for _, key := range []string{"general:", "collaborators:", "moderation:", "rulesets:"} {
		index := strings.Index(string(encoded), "\n        "+key)
		if index <= previous {
			t.Fatalf("incorrect UI order for %s: %s", key, encoded)
		}
		previous = index
	}
	if !strings.Contains(string(encoded), "\n            interaction_limits:\n") {
		t.Fatal(string(encoded))
	}
	again, err := DecodeConfig(encoded)
	if err != nil || !reflect.DeepEqual(ui, again) {
		t.Fatalf("roundtrip changed controls: %s %v", encoded, err)
	}
}

func TestModerationRejectsObsoleteConflictingAndUnsupportedShapes(t *testing.T) {
	for _, test := range []struct{ body, diagnostic string }{
		{"ignored_pull_request_limit_keys: []", "unknown repository.settings key"},
		{"interaction_limits: {}", "is no longer supported"},
		{"interaction_limits: {}, moderation: {interaction_limits: {}}", "cannot both be declared"},
		{"moderation: {interaction_limits: {}}, interaction_limits: {}", "cannot both be declared"},
		{"moderation: null", "repository.settings.moderation must not be null"},
		{"moderation: []", "repository.settings.moderation must be a mapping"},
		{"moderation: {interaction_limits: null}", "repository.settings.moderation.interaction_limits must not be null"},
		{"moderation: {code_review_limits: {}}", "unknown repository.settings.moderation key"},
		{"moderation: {interaction_limits: {temporary_restrictions: {}}}", "unknown repository.settings.moderation.interaction_limits key"},
	} {
		input := "version: daiksud/gh-settings@v0\nrepository: {settings: {" + test.body + "}}\n"
		got, err := DecodeConfig([]byte(input))
		if err == nil || !strings.Contains(err.Error(), test.diagnostic) || !reflect.DeepEqual(got, uimodel.Repository{}) {
			t.Fatalf("%s: %#v %v", test.body, got, err)
		}
	}
}

func TestEmptyModerationDoesNotManageAnySettings(t *testing.T) {
	for _, body := range []string{"{}", "{interaction_limits: {}}", "{interaction_limits: {pull_request_limits: {}}}"} {
		ui, err := DecodeConfig([]byte("version: daiksud/gh-settings@v0\nrepository: {settings: {moderation: " + body + "}}\n"))
		if err != nil {
			t.Fatal(err)
		}
		d, err := github.DeclarationFromUI(ui)
		if err != nil || !d.Empty() {
			t.Fatalf("%s: %#v %v", body, d, err)
		}
	}
}
