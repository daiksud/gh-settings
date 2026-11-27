package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// An archived apply with both families requires each family's independent
// evidence, not just a successful comparison of either one.
func TestArchivedOIDCAndCopilotRequireMatchingEvidence(t *testing.T) {
	for _, scenario := range []string{
		"matching", "oidc differs", "oidc missing", "oidc denied", "oidc fresh differs",
		"copilot instructions differ", "copilot mcp differs", "copilot missing", "copilot denied", "copilot fresh differs", "identity drift", "archive drift",
	} {
		t.Run(scenario, func(t *testing.T) {
			yes, no := true, false
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			oidc := settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &no}
			desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, OIDCSubject: &oidc, CopilotInstructions: &yes, CopilotMCP: &no}
			observed, err := settings.NewOIDCSubjectObservation(oidc)
			if err != nil {
				t.Fatal(err)
			}
			changedOIDC := settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &yes}
			changed, err := settings.NewOIDCSubjectObservation(changedOIDC)
			if err != nil {
				t.Fatal(err)
			}
			archiveReads, oidcReads, copilotReads, writes := 0, 0, 0, 0
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archiveReads++
					id, archived := int64(1), true
					if scenario == "identity drift" && archiveReads == 2 {
						id = 2
					}
					if scenario == "archive drift" && archiveReads == 2 {
						archived = false
					}
					return settings.ArchiveState{ID: id, FullName: "owner/current", Archived: archived}, nil
				},
				ReadArchivedOIDCSubject: func(_ repository.Repository, state settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error) {
					oidcReads++
					if state.ID != 1 || !state.Archived {
						t.Fatal("wrong OIDC repository identity")
					}
					values := [2]settings.OIDCSubjectObservation{observed, observed}
					switch scenario {
					case "oidc differs":
						values[0] = changed
					case "oidc fresh differs":
						values[1] = changed
					case "oidc missing":
						values[0] = settings.OIDCSubjectObservation{}
					case "oidc denied":
						return values, errors.New("denied")
					}
					return values, nil
				},
				ReadArchivedCopilot: func(_ repository.Repository, selected settings.ArchivedCopilotToggles) (settings.ArchivedCopilotToggles, error) {
					instructions, mcp := selected.Instructions != nil, selected.MCP != nil
					copilotReads++
					if !instructions || !mcp {
						t.Fatal("selected Copilot control missing")
					}
					if scenario == "copilot denied" {
						return settings.ArchivedCopilotToggles{}, errors.New("denied")
					}
					instruction, useMCP := true, false
					if scenario == "copilot instructions differ" || scenario == "copilot fresh differs" && copilotReads == 2 {
						instruction = false
					}
					if scenario == "copilot mcp differs" {
						useMCP = true
					}
					result := settings.ArchivedCopilotToggles{Instructions: &instruction, MCP: &useMCP}
					if scenario == "copilot missing" {
						result.MCP = nil
					}
					return result, nil
				},
				OIDCSubject:         func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error) { writes++; return false, nil },
				CopilotInstructions: func(repository.Repository, bool) (bool, error) { writes++; return false, nil },
				CopilotMCP:          func(repository.Repository, bool) (bool, error) { writes++; return false, nil },
				Archive:             func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID:           func(string, string, int64) error { writes++; return nil },
				BindCollaboratorID:  func(string, int64) error { writes++; return nil },
			}
			_, err = Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) { return repo, nil }, writer)
			if (err == nil) != (scenario == "matching") || writes != 0 || oidcReads != 1 {
				t.Fatalf("err=%v archive=%d oidc=%d copilot=%d writes=%d", err, archiveReads, oidcReads, copilotReads, writes)
			}
			if (scenario == "oidc differs" || scenario == "oidc missing" || scenario == "oidc denied" || scenario == "oidc fresh differs") && copilotReads != 0 {
				t.Fatalf("continued after OIDC verification failed: copilot=%d", copilotReads)
			}
			if scenario == "matching" && (archiveReads != 2 || copilotReads != 2) {
				t.Fatalf("matching evidence incomplete: archive=%d copilot=%d", archiveReads, copilotReads)
			}
			if (scenario == "identity drift" || scenario == "archive drift") && (archiveReads != 2 || copilotReads != 2) {
				t.Fatalf("final identity check did not follow both family reads: archive=%d copilot=%d", archiveReads, copilotReads)
			}
		})
	}
}
