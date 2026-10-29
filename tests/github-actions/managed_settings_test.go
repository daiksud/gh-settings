package githubactions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"go.yaml.in/yaml/v4"
)

// pinnedManagedPolicy describes only the policy assertions for the tracked
// v0.23 Action consumer. It is not a decoder for the current public YAML schema.
type pinnedManagedPolicy struct {
	SchemaVersion yaml.Node `yaml:"schemaVersion"`
	Settings      struct {
		Repository struct {
			Archived        *bool `yaml:"archived"`
			HasSponsorships *bool `yaml:"has_sponsorships"`
		} `yaml:"repository"`
		Labels   []settings.LabelDeclaration       `yaml:"labels"`
		Rulesets []settings.RulesetDeclaration     `yaml:"rulesets"`
		Actions  settings.ActionsPolicyDeclaration `yaml:"actions"`
		Security settings.SecurityDeclaration      `yaml:"security"`
		Pages    yaml.Node                         `yaml:"pages"`
	} `yaml:"settings"`
}

func readPinnedManagedPolicy(t *testing.T) pinnedManagedPolicy {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root(), ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var document pinnedManagedPolicy
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if document.SchemaVersion.Kind != yaml.ScalarNode || document.SchemaVersion.Tag != "!!int" || document.SchemaVersion.Value != "1" {
		t.Fatal("pinned released Action configuration marker changed; adoption is Issue #382")
	}
	labels, err := settings.NewLabelDeclarations(document.Settings.Labels)
	if err != nil {
		t.Fatal(err)
	}
	document.Settings.Labels = labels.Values()
	rulesets, err := settings.NewRulesetDeclarations(document.Settings.Rulesets)
	if err != nil {
		t.Fatal(err)
	}
	document.Settings.Rulesets = rulesets.Values()
	if err := document.Settings.Actions.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := document.Settings.Security.Validate(); err != nil {
		t.Fatal(err)
	}
	return document
}

func TestManagedLabelsPreserveExistingIDsNamesAndColors(t *testing.T) {
	desired := readPinnedManagedPolicy(t)
	// Preserve the independent pre-unification API observation. The intended
	// policy change updates descriptions, not existing label identity or color.
	data, err := os.ReadFile("testdata/managed-label-baseline.json")
	if err != nil {
		t.Fatal(err)
	}
	var labels []settings.LabelState
	if err := json.Unmarshal(data, &labels); err != nil {
		t.Fatal(err)
	}
	if _, err := settings.NewLabelObservation(labels); err != nil {
		t.Fatal(err)
	}
	if len(labels) != len(desired.Settings.Labels) {
		t.Fatalf("managed labels = %d, observed labels = %d", len(desired.Settings.Labels), len(labels))
	}
	observedByID := map[int64]settings.LabelState{}
	for _, label := range labels {
		observedByID[label.ID] = label
	}
	for _, label := range desired.Settings.Labels {
		if label.ID == nil || label.Color == nil {
			t.Fatalf("managed label lacks ID or color: %+v", label)
		}
		original, exists := observedByID[*label.ID]
		if !exists || original.Name != label.Name || original.Color != *label.Color {
			t.Fatalf("label identity or color changed: desired=%+v original=%+v", label, original)
		}
	}
}

func TestAdditionalManagedSettingsMatchObservedPolicy(t *testing.T) {
	desired := readPinnedManagedPolicy(t)
	if desired.Settings.Labels == nil || len(desired.Settings.Labels) != 20 {
		t.Fatal("restore exactly the twenty existing ID-owned labels")
	}
	ids := map[int64]bool{}
	for _, label := range desired.Settings.Labels {
		if label.ID == nil || *label.ID <= 0 || ids[*label.ID] || label.Color == nil || label.Description == nil {
			t.Fatalf("all labels must retain unique existing IDs and complete values: %+v", label)
		}
		ids[*label.ID] = true
		if label.Name == "ci" && *label.ID != 12791426962 {
			t.Fatal("ci must select its observed existing object, not create or adopt by name")
		}
	}
	// Independent observations recorded during adoption. These fixtures do not
	// query GitHub or establish that the live policy has remained unchanged.
	checks := []struct {
		name    string
		want    int
		compare func() (int, int, error)
	}{
		{"execution", 3, func() (int, int, error) {
			n, changes, err := settings.CompareActionsPolicy(settings.ActionsPolicyState{Enabled: true, AllowedActions: "all", SHAPinningRequired: true}, desired.Settings.Actions)
			return n, len(changes), err
		}},
		{"workflow token", 2, func() (int, int, error) {
			n, changes, err := settings.CompareWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "read", CanApprovePullRequestReviews: true}, desired.Settings.Actions)
			return n, len(changes), err
		}},
		{"retention", 1, func() (int, int, error) {
			observation, err := settings.NewRetentionObservation(90, 90)
			if err != nil {
				return 0, 0, err
			}
			n, changes, err := settings.CompareRetention(observation, desired.Settings.Actions)
			return n, len(changes), err
		}},
		{"public fork approval", 1, func() (int, int, error) {
			observation, err := settings.NewForkApprovalObservation("first_time_contributors")
			if err != nil {
				return 0, 0, err
			}
			n, changes, err := settings.CompareForkApproval(observation, desired.Settings.Actions)
			return n, len(changes), err
		}},
		{"secret scanning", 2, func() (int, int, error) {
			n, changes, err := settings.CompareSecretScanning(settings.NewSecretScanningObservation(true, true), desired.Settings.Security)
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
	if desired.Settings.Repository.Archived == nil || *desired.Settings.Repository.Archived || desired.Settings.Repository.HasSponsorships == nil || *desired.Settings.Repository.HasSponsorships {
		t.Fatal("observed false lifecycle/Sponsor values must be explicitly managed")
	}
	if (desired.Settings.Pages.Kind != 0 && (desired.Settings.Pages.Kind != yaml.MappingNode || desired.Settings.Pages.Tag != "!!map" || len(desired.Settings.Pages.Content) != 0)) || !desired.Settings.Actions.SelectedActions.Empty() || !desired.Settings.Actions.PrivateFork.Empty() {
		t.Fatal("unadopted, unsupported or inactive resources must remain unmanaged")
	}
	if desired.Settings.Rulesets == nil || len(desired.Settings.Rulesets) != 1 {
		t.Fatal("manage only the existing main-branch Ruleset")
	}
	ruleset := desired.Settings.Rulesets[0]
	if ruleset.ID == nil || *ruleset.ID != 24911852 || ruleset.Name != "default" ||
		ruleset.BypassActors == nil || len(*ruleset.BypassActors) != 0 {
		t.Fatalf("select the existing Ruleset and remove every bypass actor: %+v", ruleset)
	}
	approval, signed, linear, nativeChecks := false, false, false, false
	for _, rule := range *ruleset.Rules {
		switch rule.Type {
		case "pull_request":
			p := rule.Parameters
			approval = p != nil && p.RequiredApprovingReviewCount != nil &&
				*p.RequiredApprovingReviewCount == 1 && p.RequireCodeOwnerReview != nil &&
				*p.RequireCodeOwnerReview && p.DismissStaleReviewsOnPush != nil &&
				*p.DismissStaleReviewsOnPush && p.RequiredReviewThreadResolution != nil &&
				*p.RequiredReviewThreadResolution && p.RequireExtraApprovalForUnattributedChanges != nil &&
				*p.RequireExtraApprovalForUnattributedChanges
		case "required_status_checks":
			p := rule.Parameters
			if p == nil || p.RequiredStatusChecks == nil || p.StrictRequiredStatusChecksPolicy == nil ||
				!*p.StrictRequiredStatusChecksPolicy || p.DoNotEnforceOnCreate == nil || *p.DoNotEnforceOnCreate {
				t.Fatal("native required checks must retain strict enforcement on creation")
			}
			want := map[string]bool{"ci-pr": true, "prlint": true, "settings-plan": true, "settings-env": true}
			for _, check := range *p.RequiredStatusChecks {
				if !want[check.Context] || check.IntegrationID == nil || *check.IntegrationID != 15368 {
					t.Fatalf("unexpected check or provider: %+v", check)
				}
				delete(want, check.Context)
			}
			nativeChecks = len(want) == 0
		case "required_signatures":
			signed = true
		case "required_linear_history":
			linear = true
		}
	}
	if !approval || !signed || !linear || !nativeChecks {
		t.Fatal("the main Ruleset must keep approvals, signatures, history, and all four native checks")
	}
	// Management is not just documentation: changed security-relevant state
	// produces a diff rather than silently becoming the new desired policy.
	managed, changes, err := settings.CompareWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "write", CanApprovePullRequestReviews: false}, desired.Settings.Actions)
	if err != nil || managed != 2 || len(changes) != 2 {
		t.Fatalf("workflow drift was not detected: managed=%d, changes=%v, error=%v", managed, changes, err)
	}
}
