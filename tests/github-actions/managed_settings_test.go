package githubactions

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
	"go.yaml.in/yaml/v4"
)

func readManagedDeclaration(t *testing.T) settings.Declaration {
	t.Helper()
	// The hosted workflows validate the original document with v0.47.0.
	// Check its pinned scalar before decoding the remaining policy with current source.
	data, err := os.ReadFile(filepath.Join(root(), ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var doc, extra yaml.Node
	if err := decoder.Decode(&doc); err != nil || len(doc.Content) != 1 {
		t.Fatal("expected an operational settings document", err)
	}
	if err := decoder.Decode(&extra); err != io.EOF {
		t.Fatal("expected only one settings document", err)
	}
	wrapPinnedToggle(t, doc.Content[0], false, "repository", "settings", "general", "features", "wikis")
	data, err = yaml.Marshal(&doc)
	if err != nil {
		t.Fatal(err)
	}
	ui, err := workspace.DecodeConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	if ui.Settings != nil && len(ui.Settings.IgnoredPullRequestLimitKeys) > 0 {
		t.Fatalf("omit inactive pull request limit keys from the operational declaration: %v", ui.Settings.IgnoredPullRequestLimitKeys)
	}
	declaration, err := github.DeclarationFromUI(ui)
	if err != nil {
		t.Fatal(err)
	}
	return declaration
}

func wrapPinnedToggle(t *testing.T, node *yaml.Node, want bool, path ...string) {
	t.Helper()
	for _, key := range path {
		fields, err := settingsYAMLMap(node)
		if err != nil || fields[key] == nil {
			t.Fatalf("missing v0.47.0 toggle %s: %v", strings.Join(path, "."), err)
		}
		node = fields[key]
	}
	var got bool
	if node.Kind != yaml.ScalarNode || node.Tag != "!!bool" || node.Decode(&got) != nil || got != want {
		t.Fatalf("expected v0.47.0 %s to be boolean %t", strings.Join(path, "."), want)
	}
	if err := node.Encode(map[string]bool{"enabled": got}); err != nil {
		t.Fatal(err)
	}
}

func TestAdditionalManagedSettingsMatchObservedPolicy(t *testing.T) {
	desired := readManagedDeclaration(t)
	// Independent observations recorded during adoption. These fixtures do not
	// query GitHub or establish that the live policy has remained unchanged.
	checks := []struct {
		name    string
		want    int
		compare func() (int, int, error)
	}{
		{"execution", 3, func() (int, int, error) {
			comparison, err := settings.CompareActionsPolicy(settings.ActionsPolicyState{Enabled: true, AllowedActions: "all", SHAPinningRequired: true}, desired.Actions)
			return comparison.Managed, len(comparison.Changes) + len(comparison.Execution), err
		}},
		{"workflow token", 2, func() (int, int, error) {
			n, changes, err := settings.CompareWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "read", CanApprovePullRequestReviews: true}, desired.Actions)
			return n, len(changes), err
		}},
		{"retention", 1, func() (int, int, error) {
			observation, err := settings.NewRetentionObservation(90, 90)
			if err != nil {
				return 0, 0, err
			}
			n, changes, err := settings.CompareRetention(observation, desired.Actions)
			return n, len(changes), err
		}},
		{"public fork approval", 1, func() (int, int, error) {
			observation, err := settings.NewForkApprovalObservation("first_time_contributors")
			if err != nil {
				return 0, 0, err
			}
			n, changes, err := settings.CompareForkApproval(observation, desired.Actions)
			return n, len(changes), err
		}},
		{"CodeQL default setup", 1, func() (int, int, error) {
			n, changes, err := settings.CompareCodeQLDefaultSetup(settings.NewCodeQLDefaultSetupObservation(true), desired.Security)
			return n, len(changes), err
		}},
		{"CodeQL scan configuration", 4, func() (int, int, error) {
			observed, err := settings.NewCodeQLConfigurationObservation(settings.CodeQLConfigurationState{
				Languages:   []string{"github_actions", "go", "javascript_typescript", "python"},
				Runner:      settings.CodeQLRunner{Type: "standard_github_runner"},
				QuerySuite:  "extended",
				ThreatModel: "remote_sources",
			})
			if err != nil {
				return 0, 0, err
			}
			n, changes, err := settings.CompareCodeQLConfiguration(observed, desired.Security.CodeQLConfiguration)
			return n, len(changes), err
		}},
		{"secret scanning", 2, func() (int, int, error) {
			n, changes, err := settings.CompareSecretScanning(settings.NewSecretScanningObservation(true, true), desired.Security)
			return n, len(changes), err
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			managed, changes, err := check.compare()
			if err != nil || managed != check.want || changes != 0 {
				t.Fatalf("managed=%d, changes=%d, error=%v", managed, changes, err)
			}
		})
	}
	if desired.Repository.Archived == nil || *desired.Repository.Archived || desired.Repository.HasSponsorships == nil || *desired.Repository.HasSponsorships {
		t.Fatal("observed false lifecycle/Sponsor values must be explicitly managed")
	}
	if desired.Security.ImmutableReleases == nil || !*desired.Security.ImmutableReleases {
		t.Fatal("retain the independently observed immutable-release policy")
	}
	if !desired.Actions.SelectedActions.Empty() || !desired.Actions.PrivateFork.Empty() {
		t.Fatal("inactive or inapplicable Actions controls must remain unmanaged")
	}
	if !desired.Pages.Empty() || desired.OIDCSubject != nil || !desired.Actions.CacheStorageEmpty() ||
		desired.CopilotInstructions != nil || desired.CopilotMCP != nil || desired.CopilotReviewEffort != nil ||
		desired.Security.AIScanPullRequests != nil || desired.Security.AIScanRepositories != nil {
		t.Fatal("new controls requiring additional or unconfirmed App grants must remain deferred")
	}
	if desired.Rulesets == nil || len(desired.Rulesets.Values()) != 1 {
		t.Fatal("manage only the existing main-branch Ruleset")
	}
	ruleset := desired.Rulesets.Values()[0]
	if ruleset.ID == nil || *ruleset.ID != 24911852 || ruleset.Name != "default" ||
		ruleset.BypassActors == nil || len(*ruleset.BypassActors) != 1 {
		t.Fatalf("preserve the existing ID-selected Ruleset and its sole administrator bypass: %+v", ruleset)
	}
	actor := (*ruleset.BypassActors)[0]
	if actor.ActorType != "RepositoryRole" || actor.ActorID == nil || *actor.ActorID != 5 || actor.BypassMode == nil || *actor.BypassMode != "pull_request" {
		t.Fatalf("retain only the observed administrator pull-request bypass: %+v", actor)
	}
	approval, signatureRequired, linear, nativeChecks, scanning, quality := false, false, false, false, false, false
	for _, rule := range *ruleset.Rules {
		switch rule.Type {
		case "pull_request":
			p := rule.Parameters
			approval = p != nil && p.RequiredApprovingReviewCount != nil &&
				*p.RequiredApprovingReviewCount == 1 && p.RequireCodeOwnerReview != nil &&
				*p.RequireCodeOwnerReview && p.DismissStaleReviewsOnPush != nil &&
				*p.DismissStaleReviewsOnPush && p.RequiredReviewThreadResolution != nil &&
				*p.RequiredReviewThreadResolution && p.RequireExtraApprovalForUnattributedChanges != nil && *p.RequireExtraApprovalForUnattributedChanges
		case "required_status_checks":
			p := rule.Parameters
			if p == nil || p.RequiredStatusChecks == nil || p.StrictRequiredStatusChecksPolicy == nil ||
				!*p.StrictRequiredStatusChecksPolicy || p.DoNotEnforceOnCreate == nil || *p.DoNotEnforceOnCreate {
				t.Fatal("native required checks must retain strict enforcement on creation")
			}
			want := map[string]bool{"screening": true, "commitlint": true, "settings-pr": true}
			for _, check := range *p.RequiredStatusChecks {
				if !want[check.Context] || check.IntegrationID == nil || *check.IntegrationID != 15368 {
					t.Fatalf("unexpected check or provider: %+v", check)
				}
				delete(want, check.Context)
			}
			nativeChecks = len(want) == 0
		case "required_signatures":
			signatureRequired = true
		case "required_linear_history":
			linear = true
		case "code_scanning":
			p := rule.Parameters
			scanning = p != nil && p.CodeScanningTools != nil && len(*p.CodeScanningTools) == 1 &&
				(*p.CodeScanningTools)[0] == (settings.CodeScanningTool{Tool: "CodeQL", AlertsThreshold: "errors", SecurityAlertsThreshold: "high_or_higher"})
		case "code_quality":
			p := rule.Parameters
			quality = p != nil && p.Severity != nil && *p.Severity == "errors"
		}
	}
	if !approval || signatureRequired || !linear || !nativeChecks || !scanning || !quality {
		t.Fatal("the main Ruleset must keep approvals, history, exactly three native checks, CodeQL and Code Quality, without signatures")
	}
	// Management is not just documentation: changed security-relevant state
	// produces a diff rather than silently becoming the new desired policy.
	managed, changes, err := settings.CompareWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "write", CanApprovePullRequestReviews: false}, desired.Actions)
	if err != nil || managed != 2 || len(changes) != 2 {
		t.Fatalf("workflow drift was not detected: managed=%d, changes=%v, error=%v", managed, changes, err)
	}
}
