package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedVariablesMatchRequiresCompleteSelectedObservations(t *testing.T) {
	const value = "fixture-value-canary"
	const other = "fixture-other-canary"
	actions, err := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "BUILD_CHANNEL", Value: value}})
	if err != nil {
		t.Fatal(err)
	}
	agents, err := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "MODEL_FLAG", Value: value}})
	if err != nil {
		t.Fatal(err)
	}
	valid := func(name, value string) settings.VariableObservation {
		o, err := settings.NewVariableObservation(settings.RepositoryVariable{Name: name, Value: value})
		if err != nil {
			t.Fatal(err)
		}
		return o
	}
	repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "repo"}
	for _, scenario := range []string{"matching", "matching ruleset", "matching environment", "environment differs", "variable differs with environment", "actions differs", "agents differs", "absent", "wrong name", "unacquired", "denied", "no reader", "prune", "identity drift"} {
		t.Run(scenario, func(t *testing.T) {
			archived := true
			d := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, Variables: &actions, AgentsVariables: &agents}
			identityReads, variableReads, writes, local := 0, 0, 0, 0
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					identityReads++
					id := int64(7)
					if scenario == "identity drift" && identityReads == 2 {
						id = 8
					}
					return settings.ArchiveState{ID: id, FullName: "owner/repo", Archived: true}, nil
				},
				ReadArchivedVariable: func(_ repository.Repository, name string) (settings.VariableObservation, error) {
					variableReads++
					if name != "BUILD_CHANNEL" {
						t.Error("unexpected Actions name")
					}
					switch scenario {
					case "actions differs", "variable differs with environment":
						return valid(name, other), nil
					case "absent":
						return settings.NewAbsentVariableObservation(), nil
					case "wrong name":
						return valid("OTHER", value), nil
					case "unacquired":
						return settings.VariableObservation{}, nil
					case "denied":
						return settings.VariableObservation{}, errors.New(value)
					}
					return valid(name, value), nil
				},
				ReadArchivedAgentVariable: func(_ repository.Repository, name string) (settings.VariableObservation, error) {
					variableReads++
					if name != "MODEL_FLAG" {
						t.Error("unexpected Agents name")
					}
					if scenario == "agents differs" {
						return valid(name, other), nil
					}
					return valid(name, value), nil
				},
				Variable:           func(repository.Repository, settings.RepositoryVariable) (bool, error) { writes++; return true, nil },
				AgentVariable:      func(repository.Repository, settings.RepositoryVariable) (bool, error) { writes++; return true, nil },
				Archive:            func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID:          func(string, string, int64) error { local++; return nil },
				BindCollaboratorID: func(string, int64) error { local++; return nil },
			}
			if scenario == "matching ruleset" {
				state := appRulesetState(t)
				id := state.ID
				policy := state.Policy
				policy.ID = &id
				policies, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{policy})
				if err != nil {
					t.Fatal(err)
				}
				d.Rulesets = &policies
				writer.ReadRulesets = func(repository.Repository) (settings.RulesetObservation, error) {
					return settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
				}
			}
			if scenario == "matching environment" || scenario == "environment differs" || scenario == "variable differs with environment" {
				environments, err := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "production"}})
				if err != nil {
					t.Fatal(err)
				}
				d.Environments = &environments
				writer.ReadEnvironments = func(_ repository.Repository, _ []string) (settings.EnvironmentObservation, error) {
					if scenario == "environment differs" {
						return settings.NewEnvironmentObservation(nil)
					}
					zero, off := 0, false
					reviewers := []settings.EnvironmentReviewer{}
					return settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 71, CustomRules: []settings.EnvironmentCustomRule{}, Policy: settings.EnvironmentDeclaration{Name: "production", WaitTimer: &zero, PreventSelfReview: &off, CanAdminsBypass: &off, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}}})
				}
			}
			if scenario == "no reader" {
				writer.ReadArchivedVariable = nil
			}
			var prune settings.PruneScopes
			if scenario == "prune" {
				prune, err = settings.ParsePruneScopes("labels")
				if err != nil {
					t.Fatal(err)
				}
			}
			result, applyErr := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) { return repo, nil }, writer, prune)
			matching := scenario == "matching" || scenario == "matching ruleset" || scenario == "matching environment"
			if (applyErr == nil) != matching || writes != 0 || local != 0 {
				t.Fatal("unexpected outcome or side effect", result.Outcome, writes, local)
			}
			if applyErr != nil && (strings.Contains(applyErr.Error(), value) || strings.Contains(applyErr.Error(), other)) {
				t.Fatal("value leaked in diagnostic")
			}
			if matching && (identityReads != 2 || variableReads != 2) {
				t.Fatal("selected observations or identity checks missing", identityReads, variableReads)
			}
		})
	}
}
