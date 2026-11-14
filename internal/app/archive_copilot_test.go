package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedCopilotMatchingRequiresEverySelectedValue(t *testing.T) {
	for _, scenario := range []string{"both", "false", "instructions only", "mcp only", "instructions mismatch", "mcp mismatch", "missing", "denied", "second drift", "identity drift", "prune", "unsupported"} {
		t.Run(scenario, func(t *testing.T) {
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			yes, no := true, false
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, CopilotInstructions: &yes, CopilotMCP: &no}
			if scenario == "false" {
				d.CopilotInstructions = &no
			}
			if scenario == "instructions only" {
				d.CopilotMCP = nil
			}
			if scenario == "mcp only" {
				d.CopilotInstructions = nil
			}
			if scenario == "unsupported" {
				d.Actions.Enabled = &yes
				if archivedMatchingScope(d, settings.PruneScopes{}) {
					t.Fatal("unsupported execution policy passed the archived scope guard")
				}
			}
			archives, reads, writes := 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archives++
					id := int64(1)
					if scenario == "identity drift" && archives == 2 {
						id = 2
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: true}, nil
				},
				ReadArchivedCopilot: func(_ repository.Repository, instructions, mcp bool) (settings.ArchivedCopilotToggles, error) {
					reads++
					if instructions != (d.CopilotInstructions != nil) || mcp != (d.CopilotMCP != nil) {
						t.Fatal("incorrect selections")
					}
					if scenario == "denied" {
						return settings.ArchivedCopilotToggles{}, errors.New("denied")
					}
					v, m := true, false
					if scenario == "false" || scenario == "instructions mismatch" || scenario == "second drift" && reads == 2 {
						v = false
					}
					if scenario == "mcp mismatch" {
						m = true
					}
					o := settings.ArchivedCopilotToggles{Instructions: &v, MCP: &m}
					if scenario == "missing" {
						o.Instructions = nil
					}
					return o, nil
				},
				Archive:             func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CopilotInstructions: func(repository.Repository, bool) (bool, error) { writes++; return false, nil },
				CopilotMCP:          func(repository.Repository, bool) (bool, error) { writes++; return false, nil },
				CreatedID:           func(string, string, int64) error { writes++; return nil },
				BindCollaboratorID:  func(string, int64) error { writes++; return nil },
			}
			var options []settings.PruneScopes
			if scenario == "prune" {
				prune, err := settings.ParsePruneScopes("labels")
				if err != nil {
					t.Fatal(err)
				}
				options = append(options, prune)
			}
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w, options...)
			success := scenario == "both" || scenario == "false" || scenario == "instructions only" || scenario == "mcp only"
			if (err == nil) != success || writes != 0 || success && (reads != 2 || archives != 2) {
				t.Fatalf("outcome=%v err=%v reads=%d archives=%d writes=%d", result.Outcome, err, reads, archives, writes)
			}
			if scenario == "unsupported" && reads != 0 {
				t.Fatal("unsupported intent reached Copilot reads")
			}
		})
	}
}
