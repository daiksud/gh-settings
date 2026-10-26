package githubactions

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func TestAdditionalManagedSettingsMatchObservedPolicy(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(root(), ".github/settings.yml"))
	if err != nil {
		t.Fatal(err)
	}
	desired, err := workspace.DecodeConfig(data)
	if err != nil {
		t.Fatal(err)
	}
	if desired.Labels != nil {
		t.Fatal("the package transition must omit labels, never declare an empty managed list")
	}
	// Independent observations recorded during adoption. These fixtures do not
	// query GitHub or establish that the live policy has remained unchanged.
	checks := []struct {
		name    string
		want    int
		compare func() (int, int, error)
	}{
		{"execution", 3, func() (int, int, error) {
			n, changes, err := settings.CompareActionsPolicy(settings.ActionsPolicyState{Enabled: true, AllowedActions: "all", SHAPinningRequired: true}, desired.Actions)
			return n, len(changes), err
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
	if desired.Rulesets != nil || desired.Environments != nil || !desired.Pages.Empty() || !desired.Actions.SelectedActions.Empty() || !desired.Actions.PrivateFork.Empty() {
		t.Fatal("unadopted, unsupported or inactive resources must remain unmanaged")
	}
	// Management is not just documentation: changed security-relevant state
	// produces a diff rather than silently becoming the new desired policy.
	managed, changes, err := settings.CompareWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "write", CanApprovePullRequestReviews: false}, desired.Actions)
	if err != nil || managed != 2 || len(changes) != 2 {
		t.Fatalf("workflow drift was not detected: managed=%d, changes=%v, error=%v", managed, changes, err)
	}
}
