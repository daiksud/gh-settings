package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedRulesetsMatchRequiresCompleteIdentityAndPolicy(t *testing.T) {
	for _, scenario := range []string{"matching", "matching with extra object", "owned default differs", "policy differs", "missing ID", "create", "wrong ID", "missing object", "name collision", "normalized", "read denied", "unavailable reader", "incomplete observation", "prune", "identity changed"} {
		t.Run(scenario, func(t *testing.T) {
			state := appRulesetState(t)
			policy := state.Policy
			id := int64(7)
			policy.ID = &id
			switch scenario {
			case "missing ID", "create":
				policy.ID = nil
				if scenario == "create" {
					policy.Name = "new"
				}
			case "wrong ID":
				other := int64(8)
				policy.ID = &other
			case "policy differs":
				value := "disabled"
				policy.Enforcement = &value
			case "owned default differs":
				policy.Rules = nil // Normalizes to empty, which differs from the observed deletion rule.
			case "normalized":
				value := "disabled"
				policy.Enforcement = &value
				state.Policy.Enforcement = &value
				state.EnforcementNormalization = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "FUTURE", Fallback: "disabled"}
			}
			policies, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{policy})
			if err != nil {
				t.Fatal(err)
			}
			archived := true
			desired := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, Rulesets: &policies}
			repo := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			reads, writes, local := 0, 0, 0
			writer := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					reads++
					identity := int64(1)
					if scenario == "identity changed" && reads == 2 {
						identity = 2
					}
					return settings.ArchiveState{ID: identity, FullName: "owner/current", Archived: true}, nil
				},
				ReadRulesets: func(repository.Repository) (settings.RulesetObservation, error) {
					switch scenario {
					case "read denied":
						return settings.RulesetObservation{}, errors.New("denied")
					case "incomplete observation":
						return settings.RulesetObservation{}, nil
					case "missing object":
						return settings.NewRulesetObservation(nil, nil)
					case "name collision":
						other := state
						other.ID, other.Policy.Name = 8, "other"
						state.Policy.Name = "different"
						other.Policy.Name = "basic"
						return settings.NewRulesetObservation([]settings.RulesetState{state, other}, nil)
					case "matching with extra object":
						other := state
						other.ID, other.Policy.Name = 8, "unmanaged"
						return settings.NewRulesetObservation([]settings.RulesetState{state, other}, nil)
					default:
						return settings.NewRulesetObservation([]settings.RulesetState{state}, nil)
					}
				},
				Ruleset:   func(repository.Repository, settings.RulesetChange) (int64, error) { writes++; return 0, nil },
				Archive:   func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
				CreatedID: func(string, string, int64) error { local++; return nil },
			}
			if scenario == "unavailable reader" {
				writer.ReadRulesets = nil
			}
			var prune settings.PruneScopes
			if scenario == "prune" {
				prune, err = settings.ParsePruneScopes("rulesets")
				if err != nil {
					t.Fatal(err)
				}
			}
			result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) { return repo, nil }, writer, prune)
			matching := scenario == "matching" || scenario == "matching with extra object"
			if (err == nil) != matching || writes != 0 || local != 0 {
				t.Fatalf("outcome=%v err=%v writes=%d local=%d", result.Outcome, err, writes, local)
			}
			if matching && (reads != 2 || result.Outcome != ApplyMatchingLabels) {
				t.Fatalf("outcome=%v identity reads=%d", result.Outcome, reads)
			}
		})
	}
}
