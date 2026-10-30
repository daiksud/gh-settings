package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func exportBypassState(t *testing.T, id int64, name string, normalized bool) settings.RulesetState {
	t.Helper()
	state := exportEnforcementState(t, id, name, "evaluate", normalized && name == "z")
	always, exempt, team2, team10 := "always", "exempt", int64(2), int64(10)
	state.Policy.BypassActors = &[]settings.BypassActor{
		{ActorType: "Team", ActorID: &team10, BypassMode: &always},
		{ActorType: "DeployKey", BypassMode: &always},
		{ActorType: "Team", ActorID: &team2, BypassMode: &always},
		{ActorType: "User", ActorID: &team2, BypassMode: &exempt},
	}
	if normalized {
		for _, actor := range (*state.Policy.BypassActors)[:3] {
			state.BypassModeNormalizations = append(state.BypassModeNormalizations, settings.RulesetBypassModeNormalization{
				ActorType: actor.ActorType, ActorID: actor.ActorID,
				Normalization: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: name + "_" + actor.ActorType + "_MODE", Fallback: "always"},
			})
		}
	}
	return state
}

func TestRulesetBypassExportWarnsOnlyFromCompleteRequestedAcquisition(t *testing.T) {
	type warning struct {
		id         int64
		name, kind string
		actorID    int64
		record     settings.ObservationNormalization
	}
	for _, tc := range []struct {
		name                string
		include, normalized bool
		sinks               int
		failure             string
	}{
		{"bypass-only sink", true, true, 2, ""}, {"scalar-only sink", true, true, 1, ""}, {"both sinks", true, true, 3, ""}, {"nil sinks", true, true, 0, ""},
		{"unrequested", false, true, 3, ""}, {"canonical", true, false, 3, ""}, {"read failure", true, true, 3, "read"}, {"unacquired", true, true, 3, "unacquired"}, {"later UI failure", true, true, 3, "ui"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			z, a := exportBypassState(t, 7, "z", tc.normalized), exportBypassState(t, 9, "a", tc.normalized)
			canonical := exportEnforcementState(t, 8, "canonical", "", false)
			if tc.failure == "ui" {
				always := "always"
				canonical.Policy.BypassActors = &[]settings.BypassActor{{ActorType: "OrganizationAdmin", BypassMode: &always}}
			}
			observed, err := settings.NewRulesetObservation([]settings.RulesetState{z, canonical, a}, 0)
			if err != nil {
				t.Fatal(err)
			}
			original, _ := observed.Values()
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			cause := errors.New("bypass acquisition failed")
			reads := 0
			reader.Rulesets = func(repository.Repository) (settings.RulesetObservation, error) {
				reads++
				if tc.failure == "read" {
					return observed, cause
				}
				if tc.failure == "unacquired" {
					return settings.RulesetObservation{}, nil
				}
				return observed, nil
			}
			var got, want []warning
			saved := false
			checkout := ExportCheckout{Repository: checkoutRepository, IncludeRulesets: tc.include, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					ui, err := github.UIFromExportDeclaration(d)
					if err != nil {
						return err
					}
					if tc.include {
						entries := *ui.Settings.Rulesets
						if len(entries) != 3 || entries[0].RulesetName != "a" || entries[1].RulesetName != "canonical" || entries[2].RulesetName != "z" {
							t.Fatal("sorted complete UI lost", entries)
						}
						for _, index := range []int{0, 2} {
							actors := *entries[index].BypassList
							if len(actors) != 4 {
								t.Fatal(actors)
							}
							for _, actor := range actors {
								if actor.Team != nil && (*actor.Team != 2 && *actor.Team != 10 || *actor.BypassMode != "always") {
									t.Fatal("sink corrupted Team", actor)
								}
								if actor.DeployKeys != nil && (!*actor.DeployKeys || *actor.BypassMode != "always") {
									t.Fatal("nil-ID actor lost", actor)
								}
								if actor.User != nil && (*actor.User != 2 || *actor.BypassMode != "exempt") {
									t.Fatal("canonical companion lost", actor)
								}
							}
						}
					}
					saved = true
					return nil
				}, nil
			}}
			if tc.sinks&1 != 0 {
				checkout.RulesetNormalizationWarning = func(id int64, name string, n settings.ObservationNormalization) {
					got = append(got, warning{id: id, name: name, kind: "scalar", record: n})
				}
			}
			if tc.sinks&2 != 0 {
				checkout.RulesetBypassModeNormalizationWarning = func(id int64, name string, n settings.RulesetBypassModeNormalization) {
					e := warning{id: id, name: name, kind: n.ActorType, record: n.Normalization}
					if n.ActorID != nil {
						e.actorID = *n.ActorID
						*n.ActorID = 99
					}
					got = append(got, e)
				}
			}
			err = Export(checkout, reader)
			if tc.include && tc.normalized && tc.failure != "read" && tc.failure != "unacquired" {
				for _, x := range []struct {
					id   int64
					name string
				}{{9, "a"}, {7, "z"}} {
					if x.name == "z" && tc.sinks&1 != 0 {
						want = append(want, warning{id: x.id, name: x.name, kind: "scalar", record: z.EnforcementNormalization})
					}
					if tc.sinks&2 != 0 {
						for _, actor := range []struct {
							kind string
							id   int64
						}{{"DeployKey", 0}, {"Team", 2}, {"Team", 10}} {
							want = append(want, warning{id: x.id, name: x.name, kind: actor.kind, actorID: actor.id, record: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: x.name + "_" + actor.kind + "_MODE", Fallback: "always"}})
						}
					}
				}
			}
			wantReads := 0
			if tc.include {
				wantReads = 1
			}
			after, _ := observed.Values()
			if (err != nil) != (tc.failure != "") || saved != (tc.failure == "") || reads != wantReads || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(original, after) {
				t.Fatal(saved, reads, got, want, err)
			}
			if tc.failure == "read" && !errors.Is(err, cause) {
				t.Fatal("cause lost", err)
			}
		})
	}
}
