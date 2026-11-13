package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestClassicLockPreservesOmittedAndForkSync(t *testing.T) {
	for _, scenario := range []string{"omitted", "fork-sync-preserved", "fork-sync-drift", "omitted-lock-drift", "lock-readback-drift", "unknown-lock"} {
		t.Run(scenario, func(t *testing.T) {
			rule := classicFixtureRule(7, "release/**")
			rule["lockBranch"], rule["lockAllowsFetchAndMerge"] = true, true
			f := newClassicAPIFixture(t, rule)
			id, no, yes := int64(7), false, true
			d := settings.ClassicRule{ID: &id, BranchNamePattern: "release/**", LockBranch: &no}
			if scenario == "omitted" || scenario == "omitted-lock-drift" {
				d.LockBranch, d.RequireSignedCommits = nil, &yes
			}
			selection := classicSelect(t, f, d)
			f.hook = func(phase string, body map[string]any, _ map[string]any) {
				if phase != "node" {
					return
				}
				node := body["node"].(map[string]any)
				if scenario == "unknown-lock" {
					delete(node, "lockBranch")
				}
				if len(f.writes) == 0 {
					return
				}
				switch scenario {
				case "fork-sync-drift":
					node["lockAllowsFetchAndMerge"] = false
				case "omitted-lock-drift":
					node["lockBranch"] = false
				case "lock-readback-drift":
					node["lockBranch"] = true
				}
			}
			result, err := f.client.Apply(classicRepo(), selection)
			good := scenario == "omitted" || scenario == "fork-sync-preserved"
			if (err == nil) != good {
				t.Fatal(result, err)
			}
			if scenario == "unknown-lock" {
				if len(f.writes) != 0 || result.Wrote {
					t.Fatal("unacquired lock mutated", result, f.writes)
				}
				return
			}
			want := map[string]any{"branchProtectionRuleId": "B_7", "lockBranch": false}
			if d.LockBranch == nil {
				want = map[string]any{"branchProtectionRuleId": "B_7", "requiresCommitSignatures": true}
			}
			if !result.Wrote || len(f.writes) != 1 || !reflect.DeepEqual(f.writes[0], want) {
				t.Fatal(result, f.writes)
			}
		})
	}
}
