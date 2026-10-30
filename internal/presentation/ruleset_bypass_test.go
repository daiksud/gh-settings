package presentation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

const bypassDisplayPolicy = `{"id":7,"name":"policy","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[{"actor_type":"DeployKey","bypass_mode":"always"},{"actor_type":"Team","actor_id":9,"bypass_mode":"always"},{"actor_type":"User","actor_id":9,"bypass_mode":"exempt"}],"rules":[]}`

func TestRulesetBypassPresentationKeepsActorIdentityAndRawEvidence(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, tc := range []struct{ native, after string }{
			{"always", `"always" (write required)`},
			{"pull_request", `"for_pull_requests_only" (write required)`},
			{"removed", `inactive (actor removed; write required)`},
		} {
			var before, after settings.RulesetDeclaration
			if err := json.Unmarshal([]byte(bypassDisplayPolicy), &before); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(bypassDisplayPolicy), &after); err != nil {
				t.Fatal(err)
			}
			if tc.native == "removed" {
				*after.BypassActors = append((*after.BypassActors)[:1], (*after.BypassActors)[2:]...)
			} else {
				*(*after.BypassActors)[1].BypassMode = tc.native
			}
			id := int64(9)
			records := settings.RulesetNormalizations{BypassModes: []settings.RulesetBypassModeNormalization{
				{ActorType: "Team", ActorID: &id, Normalization: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: "FUTURE\n\"TEAM<>", Fallback: "always"}},
				{ActorType: "DeployKey", Normalization: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: "KEY\tMODE", Fallback: "always"}},
			}}
			if reverse {
				records.BypassModes[0], records.BypassModes[1] = records.BypassModes[1], records.BypassModes[0]
			}
			change, err := settings.NewRulesetDisplayChangeWithNormalizations("update", &before, &after, records)
			if err != nil {
				t.Fatal(err)
			}
			keyWarning := `Warning: repository.settings.rulesets.bypass_list.bypass_mode: observed "KEY\tMODE" is outside UI choices; using "always" (projection only). Ruleset 7 "policy", actor_type="DeployKey".`
			keyDecision := `repository.settings.rulesets.bypass_list.bypass_mode (Ruleset 7 "policy", actor_type="DeployKey"): "always" (normalized observation) → "always" (write required)`
			teamWarning := `Warning: repository.settings.rulesets.bypass_list.bypass_mode: observed "FUTURE\n\"TEAM<>" is outside UI choices; using "always" (projection only). Ruleset 7 "policy", actor_type="Team", actor_id=9.`
			teamDecision := `repository.settings.rulesets.bypass_list.bypass_mode (Ruleset 7 "policy", actor_type="Team", actor_id=9): "always" (normalized observation) → ` + tc.after
			beforeJSON, _ := json.Marshal(change.Before())
			afterJSON, _ := json.Marshal(change.After())
			whole := `settings.rulesets: update "policy": ` + string(beforeJSON) + ` → ` + string(afterJSON)
			want := strings.Join([]string{keyWarning, keyDecision, teamWarning, teamDecision, "Effective policy: " + whole}, "\n")
			if got := RulesetLine(change); got != want || strings.Count(got, "Effective policy:") != 1 {
				t.Fatalf("actor-bound presentation differs:\ngot %s\nwant %s", got, want)
			}
			if tc.native != "always" {
				ordinary, err := settings.NewRulesetDisplayChange("update", &before, &after)
				if err != nil || RulesetLine(ordinary) != whole {
					t.Fatal("ordinary output changed", err)
				}
			}
		}
	}
}
