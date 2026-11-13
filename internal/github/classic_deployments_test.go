package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func deploymentGroup(enabled bool, names ...string) *settings.ClassicDeployments {
	g := &settings.ClassicDeployments{Enabled: &enabled}
	if enabled {
		v := append([]string{}, names...)
		g.Environments = &v
	}
	return g
}

func TestClassicDeploymentReconciliation(t *testing.T) {
	for _, mode := range []string{"enable", "change-list", "empty", "disable", "matching", "other-flag", "omitted", "omitted-drift", "replace-after", "missing-before", "foreign-before", "unknown-ack", "readback-drift", "other-companion-drift", "new-unselected", "create", "create-partial-error"} {
		t.Run(mode, func(t *testing.T) {
			rule := classicFixtureRule(7, "release/**")
			rule["requiresDeployments"] = false
			rule["requiredDeploymentEnvironments"] = []string{"staging", "production"}
			f := newClassicAPIFixture(t, rule)
			id, yes := int64(7), true
			d := settings.ClassicRule{ID: &id, BranchNamePattern: "release/**", RequireDeploymentsToSucceedBeforeMerging: deploymentGroup(true, "production")}
			switch mode {
			case "change-list", "disable", "matching", "other-flag":
				rule["requiresDeployments"] = true
			}
			if mode == "matching" || mode == "other-flag" {
				d.RequireDeploymentsToSucceedBeforeMerging = deploymentGroup(true, "production", "staging")
			}
			if mode == "disable" {
				f.mutate = func(rule map[string]any) { rule["requiredDeploymentEnvironments"] = nil }
				d.RequireDeploymentsToSucceedBeforeMerging = deploymentGroup(false)
			}
			if mode == "empty" {
				d.RequireDeploymentsToSucceedBeforeMerging = deploymentGroup(true)
			}
			if mode == "other-flag" || mode == "omitted" || mode == "omitted-drift" {
				d.RequireSignedCommits = &yes
			}
			if mode == "omitted" || mode == "omitted-drift" {
				d.RequireDeploymentsToSucceedBeforeMerging = nil
			}
			if strings.HasPrefix(mode, "create") {
				f.rules = nil
				d.ID = nil
			}
			selection := classicSelect(t, f, d)
			f.calls = nil
			f.hook = func(phase string, body map[string]any, _ map[string]any) {
				if phase == "environments" {
					target := body["repository"].(map[string]any)
					page := target["environments"].(map[string]any)
					if mode == "matching" || mode == "disable" || mode == "omitted" || mode == "omitted-drift" || mode == "empty" {
						t.Fatal("unnecessary Environment acquisition")
					}
					if mode == "missing-before" {
						page["nodes"], page["totalCount"] = []any{}, 0
					}
					if mode == "foreign-before" {
						target["id"] = "R_foreign"
					}
					if len(f.writes) > 0 && mode == "replace-after" {
						page["nodes"].([]any)[0].(map[string]any)["id"] = "E_replaced"
					}
					if len(f.writes) > 0 && mode == "new-unselected" {
						page["nodes"] = append(page["nodes"].([]any), map[string]any{"id": "E_new", "databaseId": nil, "name": "new"})
						page["totalCount"] = 3
					}
				}
				if phase == "write" {
					mutation := "updateBranchProtectionRule"
					if d.ID == nil {
						mutation = "createBranchProtectionRule"
					}
					ack := body[mutation].(map[string]any)["branchProtectionRule"].(map[string]any)
					if mode == "unknown-ack" {
						delete(ack, "requiresDeployments")
					}
					if mode == "create-partial-error" {
						f.graphqlErrors = []any{map[string]any{"message": "partial resolver error"}}
					}
				}
				if phase == "node" && len(f.writes) > 0 {
					node := body["node"].(map[string]any)
					if mode == "readback-drift" {
						node["requiredDeploymentEnvironments"] = []string{"staging"}
					}
					if mode == "omitted-drift" {
						node["requiredDeploymentEnvironments"] = nil
					}
					if mode == "other-companion-drift" {
						node["lockAllowsFetchAndMerge"] = true
					}
				}
			}
			result, err := f.client.Apply(classicRepo(), selection)
			bad := strings.Contains(mode, "drift") || strings.Contains(mode, "before") || mode == "replace-after" || mode == "unknown-ack" || mode == "create-partial-error"
			if (err != nil) != bad {
				t.Fatal(result, err, f.calls)
			}
			writes := 1
			if mode == "matching" || strings.Contains(mode, "before") {
				writes = 0
			}
			if len(f.writes) != writes {
				t.Fatal(f.writes, err)
			}
			if mode == "matching" && result.Wrote {
				t.Fatal(result)
			}
			if writes == 0 {
				return
			}
			want := map[string]any{"branchProtectionRuleId": "B_7"}
			if d.ID == nil {
				want = map[string]any{"repositoryId": "R_current", "pattern": "release/**"}
			}
			switch mode {
			case "disable":
				want["requiresDeployments"] = false
			case "other-flag", "omitted", "omitted-drift":
				want["requiresCommitSignatures"] = true
			default:
				if mode != "change-list" {
					want["requiresDeployments"] = true
				}
				names := []any{"production"}
				if mode == "empty" {
					names = []any{}
				}
				want["requiredDeploymentEnvironments"] = names
			}
			if !reflect.DeepEqual(f.writes[0], want) {
				t.Fatal("payload", f.writes[0], want)
			}
			if mode == "create-partial-error" && (result.ID <= 0 || result.NodeID == "" || result.Wrote) {
				t.Fatal("lost uncertain recovery", result)
			}
		})
	}
}

func TestClassicDeploymentReferencePaginationAndMalformed(t *testing.T) {
	for _, mode := range []string{"complete", "duplicate", "null-id", "missing-database-id", "wrong-database-id", "partial", "cursor-loop", "null-page", "wrong-spelling"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t)
			f.hook = func(phase string, body map[string]any, vars map[string]any) {
				if phase != "environments" {
					return
				}
				target := body["repository"].(map[string]any)
				start, end := 0, 100
				if vars["cursor"] != nil {
					start, end = 100, 101
				}
				nodes := []any{}
				for i := start; i < end; i++ {
					nodes = append(nodes, map[string]any{"id": fmt.Sprintf("E_%d", i), "databaseId": nil, "name": fmt.Sprintf("env-%d", i)})
				}
				next := end == 100
				var cursor any
				if next {
					cursor = "next"
				}
				switch mode {
				case "duplicate":
					if start > 0 {
						nodes[0].(map[string]any)["id"] = "E_0"
					}
				case "null-id":
					nodes[0].(map[string]any)["id"] = nil
				case "missing-database-id":
					delete(nodes[0].(map[string]any), "databaseId")
				case "wrong-database-id":
					nodes[0].(map[string]any)["databaseId"] = "1"
				case "partial":
					next = false
				case "cursor-loop":
					next = true
					cursor = "next"
				}
				target["environments"] = classicFixturePage(nodes, 101, next, cursor)
				if mode == "null-page" {
					target["environments"] = nil
				}
			}
			names := []string{"env-100"}
			if mode == "wrong-spelling" {
				names = []string{"ENV-100"}
			}
			refs, err := f.client.classicDeploymentReferences(classicRepo(), classicTarget{id: "R_current"}, names)
			if (err == nil) != (mode == "complete") {
				t.Fatal(refs, err)
			}
			if mode == "complete" && refs["env-100"] != "E_100" {
				t.Fatal(refs)
			}
		})
	}
}

func TestClassicDeploymentAcquisitionRequiresTypedRawState(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, tc := range []struct {
			name  string
			value any
			bad   bool
		}{
			{"null", nil, enabled}, {"empty", []any{}, false}, {"list", []any{"production"}, false},
			{"null-member", []any{nil}, true}, {"duplicate", []any{"production", "production"}, true},
			{"blank", []any{" "}, true}, {"wrong-type", false, true},
		} {
			rule := classicFixtureRule(7, "main")
			rule["requiresDeployments"] = enabled
			rule["requiredDeploymentEnvironments"] = tc.value
			raw, _ := json.Marshal(rule)
			_, _, err := decodeClassicRule(raw, classicRepo(), true)
			if (err != nil) != tc.bad {
				t.Fatal(enabled, tc.name, err)
			}
		}
	}
}

func TestClassicDeploymentRejectedMutationRemainsUncertain(t *testing.T) {
	f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
	id := int64(7)
	selection := classicSelect(t, f, settings.ClassicRule{ID: &id, BranchNamePattern: "main", RequireDeploymentsToSucceedBeforeMerging: deploymentGroup(false)})
	f.rejectWrites = true
	f.calls = nil
	result, err := f.client.Apply(classicRepo(), selection)
	var pre *classicPreWriteFailure
	if err == nil || errors.As(err, &pre) || result.Wrote || len(f.writes) != 0 || f.calls[len(f.calls)-1] != "write-denied" {
		t.Fatal(result, err, f.calls)
	}
}
