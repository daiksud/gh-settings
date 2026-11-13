package github

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestClassicAdminCreationPolarityAndVerification(t *testing.T) {
	for _, value := range []bool{false, true} {
		for _, mode := range []string{"verified", "missing-ack", "readback-drift"} {
			t.Run(fmt.Sprintf("%t/%s", value, mode), func(t *testing.T) {
				f := newClassicAPIFixture(t)
				selection := classicSelect(t, f, settings.ClassicRule{BranchNamePattern: "new/**", DoNotAllowBypassingTheAboveSettings: &value})
				f.hook = func(phase string, body map[string]any, _ map[string]any) {
					if mode == "missing-ack" && phase == "write" {
						ack := body["createBranchProtectionRule"].(map[string]any)["branchProtectionRule"].(map[string]any)
						delete(ack, "isAdminEnforced")
					}
					if mode == "readback-drift" && phase == "node" && len(f.writes) > 0 {
						body["node"].(map[string]any)["isAdminEnforced"] = !value
					}
				}
				result, err := f.client.Apply(classicRepo(), selection)
				if (err == nil) != (mode == "verified") || !result.Wrote || result.ID <= 0 {
					t.Fatal(result, err)
				}
				want := map[string]any{"repositoryId": "R_current", "pattern": "new/**", "isAdminEnforced": value}
				if len(f.writes) != 1 || !reflect.DeepEqual(f.writes[0], want) {
					t.Fatal("incorrect administrator polarity or implicit companion", f.writes)
				}
			})
		}
	}
}

func TestClassicAdminPreservesOmissionAndCompanions(t *testing.T) {
	for _, scenario := range []string{"omitted", "fork-sync-preserved", "fork-sync-drift", "omitted-admin-drift", "admin-readback-drift", "unknown-admin"} {
		t.Run(scenario, func(t *testing.T) {
			rule := classicFixtureRule(7, "release/**")
			rule["isAdminEnforced"], rule["lockAllowsFetchAndMerge"] = true, true
			f := newClassicAPIFixture(t, rule)
			id, no, yes := int64(7), false, true
			d := settings.ClassicRule{ID: &id, BranchNamePattern: "release/**", DoNotAllowBypassingTheAboveSettings: &no}
			if scenario == "omitted" || scenario == "omitted-admin-drift" {
				d.DoNotAllowBypassingTheAboveSettings, d.RequireSignedCommits = nil, &yes
			}
			selection := classicSelect(t, f, d)
			f.hook = func(phase string, body map[string]any, _ map[string]any) {
				if phase != "node" {
					return
				}
				node := body["node"].(map[string]any)
				if scenario == "unknown-admin" {
					delete(node, "isAdminEnforced")
				}
				if len(f.writes) == 0 {
					return
				}
				switch scenario {
				case "fork-sync-drift":
					node["lockAllowsFetchAndMerge"] = false
				case "omitted-admin-drift":
					node["isAdminEnforced"] = false
				case "admin-readback-drift":
					node["isAdminEnforced"] = true
				}
			}
			result, err := f.client.Apply(classicRepo(), selection)
			good := scenario == "omitted" || scenario == "fork-sync-preserved"
			if (err == nil) != good {
				t.Fatal(result, err)
			}
			if scenario == "unknown-admin" {
				if len(f.writes) != 0 || result.Wrote {
					t.Fatal("unacquired administrator flag mutated", result, f.writes)
				}
				return
			}
			want := map[string]any{"branchProtectionRuleId": "B_7", "isAdminEnforced": false}
			if d.DoNotAllowBypassingTheAboveSettings == nil {
				want = map[string]any{"branchProtectionRuleId": "B_7", "requiresCommitSignatures": true}
			}
			if !result.Wrote || len(f.writes) != 1 || !reflect.DeepEqual(f.writes[0], want) {
				t.Fatal(result, f.writes)
			}
		})
	}
}
