package app

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedOIDCAndVariablesRequireAllSelectedMatches(t *testing.T) {
	for _, scenario := range []string{"matching", "OIDC differs", "Actions variable differs", "Agents variable differs"} {
		t.Run(scenario, func(t *testing.T) {
			yes, no := true, false
			desiredOIDC := settings.OIDCSubjectTemplate{UseImmutableSubjectClaim: &no}
			observedImmutable := no
			if scenario == "OIDC differs" {
				observedImmutable = yes
			}
			oidc, err := settings.NewOIDCSubjectObservation(settings.OIDCSubjectTemplate{UseDefaultTemplate: &yes, UseImmutableSubjectClaim: &observedImmutable})
			if err != nil {
				t.Fatal(err)
			}
			actions, err := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "BUILD_CHANNEL", Value: "stable"}})
			if err != nil {
				t.Fatal(err)
			}
			agents, err := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "MODEL_FLAG", Value: "enabled"}})
			if err != nil {
				t.Fatal(err)
			}
			declaration := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, OIDCSubject: &desiredOIDC, Variables: &actions, AgentsVariables: &agents}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			writes, oidcReads, variableReads := 0, 0, 0
			readVariable := func(name, value string) (settings.VariableObservation, error) {
				variableReads++
				return settings.NewVariableObservation(settings.RepositoryVariable{Name: name, Value: value})
			}
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
				},
				ReadArchivedOIDCSubject: func(repository.Repository, settings.ArchiveState) ([2]settings.OIDCSubjectObservation, error) {
					oidcReads++
					return [2]settings.OIDCSubjectObservation{oidc, oidc}, nil
				},
				ReadArchivedVariable: func(_ repository.Repository, name string) (settings.VariableObservation, error) {
					value := "stable"
					if scenario == "Actions variable differs" {
						value = "other"
					}
					return readVariable(name, value)
				},
				ReadArchivedAgentVariable: func(_ repository.Repository, name string) (settings.VariableObservation, error) {
					value := "enabled"
					if scenario == "Agents variable differs" {
						value = "other"
					}
					return readVariable(name, value)
				},
				OIDCSubject:   func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error) { writes++; return false, nil },
				Variable:      func(repository.Repository, settings.RepositoryVariable) (bool, error) { writes++; return false, nil },
				AgentVariable: func(repository.Repository, settings.RepositoryVariable) (bool, error) { writes++; return false, nil },
			}
			_, err = Apply(func() (settings.Declaration, error) { return declaration, nil }, func() (repository.Repository, error) { return repo, nil }, writer)
			if (err == nil) != (scenario == "matching") || writes != 0 || oidcReads != 1 {
				t.Fatalf("scenario %q: err=%v writes=%d oidc reads=%d", scenario, err, writes, oidcReads)
			}
			if scenario == "matching" && variableReads != 2 {
				t.Fatalf("expected both variable families, got %d reads", variableReads)
			}
		})
	}
}
