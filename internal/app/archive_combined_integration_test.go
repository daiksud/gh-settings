package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Each selected family must provide complete matching evidence before an
// archived apply can succeed. These fixtures do not imply an atomic snapshot.
func TestArchivedCombinedFamiliesRequireCompleteMatchingEvidence(t *testing.T) {
	for _, scenario := range []string{
		"matching", "workflow differs", "workflow unavailable", "approval differs",
		"cache retention differs", "cache retention unavailable", "cache storage differs", "cache storage unavailable",
		"instructions differ", "mcp differs", "copilot unavailable", "copilot fresh differs",
		"oidc differs", "oidc unavailable", "variable differs", "variable unavailable",
		"identity drift", "archive drift", "prune",
	} {
		t.Run(scenario, func(t *testing.T) {
			yes, no, permission, days, size := true, false, "read", 30, 10
			variables, err := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "BUILD_CHANNEL", Value: "stable"}})
			if err != nil {
				t.Fatal(err)
			}
			desired := settings.Declaration{
				Repository:          settings.RepositoryDeclaration{Archived: &yes},
				Actions:             settings.ActionsPolicyDeclaration{DefaultWorkflowPermissions: &permission, CanApprovePullRequestReviews: &no, CacheRetentionDays: &days, CacheSizeGB: &size},
				CopilotInstructions: &yes, CopilotMCP: &no,
				OIDCSubject: &settings.OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}, Variables: &variables,
			}
			oidcValue := settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &no}
			if scenario == "oidc differs" {
				oidcValue.UseImmutableSubjectClaim = &yes
			}
			oidc, err := settings.NewOIDCSubjectObservation(oidcValue)
			if err != nil {
				t.Fatal(err)
			}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			archiveReads, workflowReads, retentionReads, storageReads, copilotReads, oidcReads, variableReads, writes := 0, 0, 0, 0, 0, 0, 0, 0
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
				ReadWorkflowPermissions: func(repository.Repository) (settings.WorkflowPermissionsState, error) {
					workflowReads++
					if scenario == "workflow unavailable" {
						return settings.WorkflowPermissionsState{}, errors.New("denied")
					}
					p, approval := "read", false
					if scenario == "workflow differs" {
						p = "write"
					}
					if scenario == "approval differs" {
						approval = true
					}
					return settings.WorkflowPermissionsState{DefaultWorkflowPermissions: p, CanApprovePullRequestReviews: approval}, nil
				},
				ReadCacheRetention: func(repository.Repository) (settings.CacheRetentionObservation, error) {
					retentionReads++
					if scenario == "cache retention unavailable" {
						return settings.CacheRetentionObservation{}, nil
					}
					value := 30
					if scenario == "cache retention differs" {
						value = 31
					}
					return settings.NewCacheRetentionObservation(value, "public")
				},
				ReadCacheStorage: func(repository.Repository) (settings.CacheStorageObservation, error) {
					storageReads++
					if scenario == "cache storage unavailable" {
						return settings.CacheStorageObservation{}, nil
					}
					value := 10
					if scenario == "cache storage differs" {
						value = 11
					}
					return settings.NewCacheStorageObservation(value)
				},
				ReadArchivedCopilot: func(_ repository.Repository, instructions, mcp bool) (settings.ArchivedCopilotToggles, error) {
					copilotReads++
					if !instructions || !mcp {
						t.Fatal("selected Copilot controls not requested")
					}
					if scenario == "copilot unavailable" {
						return settings.ArchivedCopilotToggles{}, nil
					}
					instruction, useMCP := true, false
					if scenario == "instructions differ" || scenario == "copilot fresh differs" && copilotReads == 2 {
						instruction = false
					}
					if scenario == "mcp differs" {
						useMCP = true
					}
					return settings.ArchivedCopilotToggles{Instructions: &instruction, MCP: &useMCP}, nil
				},
				ReadArchivedOIDCSubject: func(_ repository.Repository, state settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error) {
					oidcReads++
					if state.ID != 1 || !state.Archived {
						t.Fatal("wrong OIDC lifecycle identity")
					}
					if scenario == "oidc unavailable" {
						return [2]settings.OIDCSubjectObservation{}, nil
					}
					return [2]settings.OIDCSubjectObservation{oidc, oidc}, nil
				},
				ReadArchivedVariable: func(_ repository.Repository, name string) (settings.VariableObservation, error) {
					variableReads++
					if name != "BUILD_CHANNEL" {
						t.Fatal("wrong variable selected")
					}
					if scenario == "variable unavailable" {
						return settings.VariableObservation{}, nil
					}
					value := "stable"
					if scenario == "variable differs" {
						value = "other"
					}
					return settings.NewVariableObservation(settings.RepositoryVariable{Name: name, Value: value})
				},
				Archive: func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				WorkflowPermissions: func(repository.Repository, settings.ActionsPolicyDeclaration) (bool, error) {
					writes++
					return false, nil
				},
				CacheRetention:      func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				CacheStorage:        func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				CopilotInstructions: func(repository.Repository, bool) (bool, error) { writes++; return false, nil },
				CopilotMCP:          func(repository.Repository, bool) (bool, error) { writes++; return false, nil },
				OIDCSubject:         func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error) { writes++; return false, nil },
				Variable:            func(repository.Repository, settings.RepositoryVariable) (bool, error) { writes++; return false, nil },
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
			_, err = Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) { return repo, nil }, writer, options...)
			if (err == nil) != (scenario == "matching") || writes != 0 {
				t.Fatalf("err=%v writes=%d reads archive=%d workflow=%d retention=%d storage=%d copilot=%d oidc=%d variable=%d", err, writes, archiveReads, workflowReads, retentionReads, storageReads, copilotReads, oidcReads, variableReads)
			}
			if scenario == "matching" && (archiveReads != 2 || workflowReads != 1 || retentionReads != 1 || storageReads != 1 || copilotReads != 2 || oidcReads != 1 || variableReads != 1) {
				t.Fatalf("incomplete matching evidence: archive=%d workflow=%d retention=%d storage=%d copilot=%d oidc=%d variable=%d", archiveReads, workflowReads, retentionReads, storageReads, copilotReads, oidcReads, variableReads)
			}
		})
	}
}
