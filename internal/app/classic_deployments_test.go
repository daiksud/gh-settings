package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"strings"
	"testing"
)

func TestClassicDeploymentPruneConflictBeforeAnyWrite(t *testing.T) {
	for _, name := range []string{"production", "PRODUCTION"} {
		for _, archived := range []bool{false, true} {
			no, yes, description, id := false, true, "changed", int64(7)
			names := []string{name}
			rules := []settings.ClassicRule{{ID: &id, BranchNamePattern: "main", RequireDeploymentsToSucceedBeforeMerging: &settings.ClassicDeployments{Enabled: &yes, Environments: &names}}}
			environments, _ := settings.NewEnvironmentDeclarations(nil)
			desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description, Archived: &no}, BranchProtectionRules: &rules, Environments: &environments}
			prune, _ := settings.ParsePruneScopes("environments")
			candidate := environmentPruneFixture(t, archived, "production")
			result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, checkoutRepository, Writer{
				ReadEnvironmentPrune: func(repository.Repository, []string) (settings.EnvironmentPruneObservation, error) {
					return candidate, nil
				},
				ReadBranchProtectionRules: func(repository.Repository) (settings.ClassicObservation, error) {
					t.Fatal("read Classic after conflicting global prune")
					return settings.ClassicObservation{}, nil
				},
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					t.Fatal("unarchive reached")
					return settings.ArchiveState{}, nil
				},
				Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
					t.Fatal("unrelated write reached")
					return false, nil
				},
			}, prune)
			if err == nil || !strings.Contains(err.Error(), "conflicts") || result.Outcome != ApplyFailed || len(result.Operations) != 0 {
				t.Fatal(result, err)
			}
		}
	}
}

func TestClassicDeploymentPruneOmissionAndDisabledDoNotRetain(t *testing.T) {
	for _, mode := range []string{"omitted", "empty", "disabled", "different"} {
		no, yes := false, true
		group := (*settings.ClassicDeployments)(nil)
		switch mode {
		case "empty":
			group = &settings.ClassicDeployments{}
		case "disabled":
			group = &settings.ClassicDeployments{Enabled: &no}
		case "different":
			names := []string{"kept"}
			group = &settings.ClassicDeployments{Enabled: &yes, Environments: &names}
		}
		rules := []settings.ClassicRule{{BranchNamePattern: "main", RequireDeploymentsToSucceedBeforeMerging: group}}
		environments, _ := settings.NewEnvironmentDeclarations(nil)
		desired := settings.Declaration{BranchProtectionRules: &rules, Environments: &environments}
		prune, _ := settings.ParsePruneScopes("environments")
		prepared, err := preparePrune(repository.Repository{Owner: "owner", Name: "current"}, desired, prune, Writer{ReadEnvironmentPrune: func(_ repository.Repository, keep []string) (settings.EnvironmentPruneObservation, error) {
			if len(keep) != 0 {
				t.Fatal("silently retained refs", keep)
			}
			return environmentPruneFixture(t, false, "production"), nil
		}})
		if err != nil {
			t.Fatal(mode, err)
		}
		candidates, err := prepared.environments.Candidates()
		if err != nil || len(candidates) != 1 {
			t.Fatal(candidates, err)
		}
	}
}
