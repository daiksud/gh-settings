package githubactions

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v4"

	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// The operational Action is still pinned to v0.31. Keep its legacy protection
// intent intact. Only this test projects the remaining fields through today's
// codec; this is not a runtime migration or acceptance of retired YAML controls.
func readManagedDeclaration(t *testing.T) settings.Declaration {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(), ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := workspace.DecodeConfig(data); err == nil {
		t.Fatal("operational legacy controls unexpectedly became current YAML")
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	repository := raw["repository"].(map[string]any)
	configuration := repository["settings"].(map[string]any)
	rulesets := configuration["rulesets"].([]any)
	if len(rulesets) != 1 {
		t.Fatal("unexpected operational Rulesets")
	}
	rules := rulesets[0].(map[string]any)["rules"].(map[string]any)
	quality, ok := rules["require_code_quality_results"].(map[string]any)
	if !ok || len(quality) != 1 || quality["severity"] != "errors" {
		t.Fatal("pinned apply must retain Code Quality protection")
	}
	pr := rules["require_a_pull_request_before_merging"].(map[string]any)
	if pr["require_an_additional_approval_for_unattributed_copilot_pull_requests"] != true {
		t.Fatal("pinned apply must retain Preview approval true")
	}
	teams, ok := pr["require_review_from_specific_teams"].([]any)
	if !ok || len(teams) != 0 {
		t.Fatal("pinned empty team companion changed")
	}
	delete(rules, "require_code_quality_results")
	delete(pr, "require_an_additional_approval_for_unattributed_copilot_pull_requests")
	delete(pr, "require_review_from_specific_teams")
	// Test-only current view: never write this projection or feed it to apply.
	data, err = yaml.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	ui, err := workspace.DecodeConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := github.DeclarationFromUI(ui)
	if err != nil {
		t.Fatal(err)
	}
	return declaration
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
	if !desired.Pages.Empty() || !desired.Actions.SelectedActions.Empty() || !desired.Actions.PrivateFork.Empty() {
		t.Fatal("unadopted, unsupported or inactive resources must remain unmanaged")
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
				*p.RequiredReviewThreadResolution && p.RequireExtraApprovalForUnattributedChanges == nil
		case "required_status_checks":
			p := rule.Parameters
			if p == nil || p.RequiredStatusChecks == nil || p.StrictRequiredStatusChecksPolicy == nil ||
				!*p.StrictRequiredStatusChecksPolicy || p.DoNotEnforceOnCreate == nil || *p.DoNotEnforceOnCreate {
				t.Fatal("native required checks must retain strict enforcement on creation")
			}
			want := map[string]bool{"screening": true, "prlint": true, "settings-pr": true}
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
	if !approval || signatureRequired || !linear || !nativeChecks || !scanning || quality {
		t.Fatal("the main Ruleset must keep approvals, history, exactly three native checks, CodeQL, without signatures or unadopted Code Quality intent")
	}
	// Management is not just documentation: changed security-relevant state
	// produces a diff rather than silently becoming the new desired policy.
	managed, changes, err := settings.CompareWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "write", CanApprovePullRequestReviews: false}, desired.Actions)
	if err != nil || managed != 2 || len(changes) != 2 {
		t.Fatalf("workflow drift was not detected: managed=%d, changes=%v, error=%v", managed, changes, err)
	}
}
