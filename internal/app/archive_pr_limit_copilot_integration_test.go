package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestArchivedPullRequestLimitsAndCopilotIntegration(t *testing.T) {
	for _, scenario := range []string{"matching", "family differs", "family unavailable", "copilot differs"} {
		t.Run(scenario, func(t *testing.T) {
			yes, no := true, false
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, CopilotInstructions: &yes, CopilotMCP: &no}
			d.PullRequestLimits.Enabled = &yes
			archives, familyReads, copilotReads, writes := 0, 0, 0, 0
			w := Writer{ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
				archives++
				return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
			}, ReadArchivedCopilot: func(repository.Repository, settings.ArchivedCopilotToggles) (settings.ArchivedCopilotToggles, error) {
				copilotReads++
				v := true
				if scenario == "copilot differs" {
					v = false
				}
				return settings.ArchivedCopilotToggles{Instructions: &v, MCP: &no}, nil
			}, Archive: func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil }, CopilotInstructions: func(repository.Repository, bool) (bool, error) { writes++; return false, nil }, CopilotMCP: func(repository.Repository, bool) (bool, error) { writes++; return false, nil }, CreatedID: func(string, string, int64) error { writes++; return nil }, BindCollaboratorID: func(string, int64) error { writes++; return nil }}
			w.ReadPullRequestLimits = func(repository.Repository) (settings.PullRequestLimitsObservation, error) {
				familyReads++
				if scenario == "family unavailable" {
					return settings.PullRequestLimitsObservation{}, nil
				}
				v := true
				if scenario == "family differs" {
					v = false
				}
				return settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: v, Maximum: 5})
			}
			_, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (scenario == "matching") || writes != 0 || familyReads == 0 {
				t.Fatalf("err=%v archive=%d family=%d copilot=%d writes=%d", err, archives, familyReads, copilotReads, writes)
			}
			if scenario == "matching" && (archives != 2 || copilotReads != 2 || familyReads != 1) {
				t.Fatalf("missing matching evidence: archive=%d family=%d copilot=%d", archives, familyReads, copilotReads)
			}
		})
	}
}
