package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestArchivedCacheCopilotAndOIDCRequireMatchingEvidence(t *testing.T) {
	for _, scenario := range []string{"matching", "family differs", "copilot differs", "OIDC differs"} {
		t.Run(scenario, func(t *testing.T) {
			yes, no, days := true, false, 30
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			desiredOIDC := settings.OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}
			immutable := no
			if scenario == "OIDC differs" {
				immutable = yes
			}
			oidc, err := settings.NewOIDCSubjectObservation(settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &immutable})
			if err != nil {
				t.Fatal(err)
			}
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, OIDCSubject: &desiredOIDC, CopilotInstructions: &yes, Actions: settings.ActionsPolicyDeclaration{CacheRetentionDays: &days}}
			familyReads, copilotReads, oidcReads, writes := 0, 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
				},
				ReadArchivedOIDCSubject: func(repository.Repository, settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error) {
					oidcReads++
					return [2]settings.OIDCSubjectObservation{oidc, oidc}, nil
				},
				ReadArchivedCopilot: func(_ repository.Repository, instructions, mcp bool) (settings.ArchivedCopilotToggles, error) {
					copilotReads++
					if !instructions || mcp {
						t.Fatal("wrong selection")
					}
					value := true
					if scenario == "copilot differs" {
						value = false
					}
					return settings.ArchivedCopilotToggles{Instructions: &value}, nil
				},
				ReadCacheRetention: func(repository.Repository) (settings.CacheRetentionObservation, error) {
					familyReads++
					value := 30
					if scenario == "family differs" {
						value = 31
					}
					return settings.NewCacheRetentionObservation(value, "public")
				},
				CacheRetention:      func(repository.Repository, int) (bool, error) { writes++; return false, nil },
				OIDCSubject:         func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error) { writes++; return false, nil },
				CopilotInstructions: func(repository.Repository, bool) (bool, error) { writes++; return false, nil },
				Archive:             func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID:           func(string, string, int64) error { writes++; return nil },
			}
			_, err = Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, w)
			if (err == nil) != (scenario == "matching") || writes != 0 || oidcReads != 1 {
				t.Fatalf("err=%v writes=%d reads=%d/%d/%d", err, writes, oidcReads, copilotReads, familyReads)
			}
			if scenario == "matching" && (familyReads != 1 || copilotReads != 2) {
				t.Fatalf("incomplete matching reads %d/%d", familyReads, copilotReads)
			}
		})
	}
}
