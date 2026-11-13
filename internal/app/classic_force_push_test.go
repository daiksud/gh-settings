package app

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestClassicForcePushTeamExportRetainsPriorAndApplyPreserves(t *testing.T) {
	observed := appClassicObserved()
	state := &observed.Rules[0]
	state.Companions.ForcePushActors = []settings.ClassicActor{{Kind: "Team", ID: "T_PRIVATE"}}
	var err error
	state.Rule.AllowForcePushes, err = settings.ClassicForcePushProjection(false, state.Companions.ForcePushActors)
	if err != nil {
		t.Fatal(err)
	}
	prior := settings.CloneClassicRule(state.Rule)
	prior.AllowForcePushes = nil
	rules := []settings.ClassicRule{prior}
	existing := settings.Declaration{BranchProtectionRules: &rules}
	infos, saved := 0, false
	err = Export(ExportCheckout{Repository: checkoutRepository, Existing: existing, Info: func(path string, _ error) error {
		if path == settings.ClassicProtectionPath {
			infos++
		}
		return nil
	}, PrepareSave: func() (func(settings.Declaration) error, error) {
		return func(actual settings.Declaration) error {
			saved = true
			if actual.BranchProtectionRules == nil || (*actual.BranchProtectionRules)[0].AllowForcePushes != nil {
				t.Fatal(actual)
			}
			return nil
		}, nil
	}}, Reader{BranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) { return observed, nil }})
	if err != nil || !saved || infos != 1 {
		t.Fatal(saved, infos, err)
	}
	yes := true
	rules[0].RequireSignedCommits = &yes
	called := false
	result, err := Apply(func() (settings.Declaration, error) { return existing, nil }, checkoutRepository, Writer{ReadBranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) { return observed, nil }, BranchProtectionRule: func(_ repository.Repository, selected settings.ClassicSelection) (settings.ClassicWriteResult, error) {
		called = true
		if selected.Before.Companions.ForcePushActors[0].Kind != "Team" || selected.Desired.AllowForcePushes != nil {
			t.Fatal(selected)
		}
		return settings.ClassicWriteResult{ID: *selected.Before.Rule.ID, NodeID: selected.Before.NodeID, Wrote: true}, nil
	}})
	if err != nil || !called || result.Outcome != ApplyCompleted {
		t.Fatal(result, called, err)
	}
}
