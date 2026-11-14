package app

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// The two independently read families and the Copilot toggle must all match
// in the same invocation; no prior result authorizes a later mutation.
func TestArchivedRetentionLimitsAndCopilotMustAllMatch(t *testing.T) {
	for _, scenario := range []string{"match", "retention differs", "retention missing", "limits differ", "limits missing", "copilot differs", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			yes, days := true, 30
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, Actions: settings.ActionsPolicyDeclaration{RetentionDays: &days}, PullRequestLimits: settings.PullRequestLimitsDeclaration{Enabled: &yes}, CopilotInstructions: &yes}
			archiveReads, retentionReads, limitsReads, copilotReads, writes := 0, 0, 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archiveReads++
					id := int64(1)
					if scenario == "identity drift" && archiveReads == 2 {
						id = 2
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
				},
				ReadRetention: func(repository.Repository) (settings.RetentionObservation, error) {
					retentionReads++
					if scenario == "retention missing" {
						return settings.RetentionObservation{}, nil
					}
					value := 30
					if scenario == "retention differs" {
						value = 31
					}
					return settings.NewRetentionObservation(value, 90)
				},
				ReadPullRequestLimits: func(repository.Repository) (settings.PullRequestLimitsObservation, error) {
					limitsReads++
					if scenario == "limits missing" {
						return settings.PullRequestLimitsObservation{}, nil
					}
					return settings.NewPullRequestLimitsObservation(settings.PullRequestLimitsValues{Enabled: scenario != "limits differ", Maximum: 5})
				},
				ReadArchivedCopilot: func(_ repository.Repository, instructions, mcp bool) (settings.ArchivedCopilotToggles, error) {
					copilotReads++
					if !instructions || mcp {
						t.Fatal("wrong Copilot selection")
					}
					value := scenario != "copilot differs"
					return settings.ArchivedCopilotToggles{Instructions: &value}, nil
				},
				Retention: func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				PullRequestLimits: func(repository.Repository, settings.PullRequestLimitsDeclaration) (bool, error) {
					writes++
					return false, nil
				},
				CopilotInstructions: func(repository.Repository, bool) (bool, error) { writes++; return false, nil },
				Archive:             func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID:           func(string, string, int64) error { writes++; return nil },
				BindCollaboratorID:  func(string, int64) error { writes++; return nil },
			}
			_, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (scenario == "match") || writes != 0 {
				t.Fatalf("err=%v writes=%d", err, writes)
			}
			if scenario == "match" && (archiveReads != 2 || retentionReads != 1 || limitsReads != 1 || copilotReads != 2) {
				t.Fatalf("reads archive=%d retention=%d limits=%d copilot=%d", archiveReads, retentionReads, limitsReads, copilotReads)
			}
		})
	}
}
