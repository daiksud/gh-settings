package app

import (
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestClassicForkSyncNonforkStopsFamilyAndReportsEarlierWrites(t *testing.T) {
	for _, earlier := range []bool{false, true} {
		yes := true
		description := "desired"
		observed := appClassicObserved()
		rules := []settings.ClassicRule{{ID: observed.Rules[0].Rule.ID, BranchNamePattern: "main", LockBranch: &yes, AllowForkSyncing: &yes}}
		called := false
		result, err := Apply(func() (settings.Declaration, error) {
			return settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}, BranchProtectionRules: &rules}, nil
		}, checkoutRepository, Writer{
			Repository:                func(repository.Repository, settings.RepositoryDeclaration) (bool, error) { return earlier, nil },
			ReadBranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) { return observed, nil },
			BranchProtectionRule: func(repository.Repository, settings.ClassicSelection) (settings.ClassicWriteResult, error) {
				called = true
				return settings.ClassicWriteResult{}, nil
			},
		})
		if err == nil || called || (result.Outcome == ApplyPartialFailure) != earlier || strings.Contains(err.Error(), "earlier successful writes") != earlier {
			t.Fatal(result, err, called)
		}
	}
}

func TestClassicForkSyncIncompleteProjectionRetainsPriorExport(t *testing.T) {
	yes := true
	observed := appClassicObserved()
	observed.IsFork, observed.Rules[0].IsFork = true, true
	observed.Rules[0].Rule.LockBranch = &yes
	prior := []settings.ClassicRule{{BranchNamePattern: "retained"}}
	saved, infos := false, 0
	err := Export(ExportCheckout{Repository: checkoutRepository, Existing: settings.Declaration{BranchProtectionRules: &prior}, Info: func(path string, _ error) error {
		if path == settings.ClassicProtectionPath {
			infos++
		}
		return nil
	}, PrepareSave: func() (func(settings.Declaration) error, error) {
		return func(d settings.Declaration) error {
			saved = true
			if d.BranchProtectionRules == nil || (*d.BranchProtectionRules)[0].BranchNamePattern != "retained" {
				t.Fatal(d)
			}
			return nil
		}, nil
	}}, Reader{BranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) { return observed, nil }})
	if err != nil || !saved || infos != 1 {
		t.Fatal(err, saved, infos)
	}
}
