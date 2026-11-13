package github

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func classicPruneSelect(t *testing.T, f *classicAPIFixture) settings.ClassicPruneCandidate {
	t.Helper()
	observed, err := f.client.Read(classicRepo())
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := settings.PrepareClassicPrune(observed, []settings.ClassicRule{})
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := prepared.Candidates()
	if err != nil || len(candidates) == 0 {
		t.Fatal(candidates, err)
	}
	f.calls = nil
	return candidates[0]
}
func TestClassicPruneDeleteExactIdentityACKAndAbsence(t *testing.T) {
	f := newClassicAPIFixture(t, classicFixtureRule(7, "main"), classicFixtureRule(8, "main"))
	candidate := classicPruneSelect(t, f)
	if err := f.client.Delete(classicRepo(), candidate); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.writes, []map[string]any{{"branchProtectionRuleId": "B_7"}}) || len(f.rules) != 1 || f.rules[0]["id"] != "B_8" {
		t.Fatal(f.writes, f.rules)
	}
	if got := strings.Join(f.calls, ","); got != "node,actors,actors,actors,actors,delete,inventory,actors,actors,actors,actors" {
		t.Fatal(got)
	}
}
func TestClassicPruneDeletionFailureBoundaries(t *testing.T) {
	for _, mode := range []string{"node-absent", "target", "archived", "identity", "pattern", "raw-null", "raw-actor", "ack-null", "ack-missing", "ack-wrong", "ack-extra", "partial-error", "inventory-missing", "same-node", "same-database-id", "readback-archived"} {
		t.Run(mode, func(t *testing.T) {
			f := newClassicAPIFixture(t, classicFixtureRule(7, "main"))
			candidate := classicPruneSelect(t, f)
			pre := mode == "node-absent" || mode == "target" || mode == "archived" || mode == "identity" || mode == "pattern" || mode == "raw-null" || mode == "raw-actor"
			f.hook = func(phase string, body, vars map[string]any) {
				if phase == "node" {
					n := body["node"].(map[string]any)
					switch mode {
					case "node-absent":
						body["node"] = nil
					case "target":
						n["repository"].(map[string]any)["id"] = "replacement"
					case "archived":
						n["repository"].(map[string]any)["isArchived"] = true
					case "identity":
						n["databaseId"] = 90
					case "pattern":
						n["pattern"] = "renamed"
					case "raw-null":
						n["requiredStatusCheckContexts"] = nil
					}
				}
				if phase == "actors" && mode == "raw-actor" {
					n := body["node"].(map[string]any)
					n["pushAllowances"] = classicFixturePage([]any{map[string]any{"actor": map[string]any{"__typename": "User", "id": "PRIVATE_ACTOR"}}}, 1, false, nil)
				}
				if phase == "delete" {
					switch mode {
					case "ack-null":
						body["deleteBranchProtectionRule"] = nil
					case "ack-missing":
						delete(body, "deleteBranchProtectionRule")
					case "ack-wrong":
						body["deleteBranchProtectionRule"].(map[string]any)["__typename"] = "Other"
					case "ack-extra":
						body["deleteBranchProtectionRule"].(map[string]any)["extra"] = true
					case "partial-error":
						f.graphqlErrors = []any{map[string]any{"message": "resolver failed", "type": "NOT_FOUND", "path": []any{"repository"}}}
					}
				}
				if phase == "inventory" {
					target := body["repository"].(map[string]any)
					switch mode {
					case "inventory-missing":
						delete(target, "branchProtectionRules")
					case "readback-archived":
						target["isArchived"] = true
					case "same-node", "same-database-id":
						rule := classicFixtureRule(7, "other")
						if mode == "same-node" {
							rule["databaseId"] = 99
						} else {
							rule["id"] = "B_other"
						}
						f.rules = append(f.rules, rule)
						target["branchProtectionRules"] = classicFixturePage([]any{rule}, 1, false, nil)
					}
				}
			}
			err := f.client.Delete(classicRepo(), candidate)
			var marker *classicPreWriteFailure
			if err == nil || errors.As(err, &marker) != pre || len(f.writes) != map[bool]int{true: 0, false: 1}[pre] {
				t.Fatal(f.calls, f.writes, err)
			}
			if !pre && !strings.Contains(err.Error(), "write may have applied") {
				t.Fatal("uncertain effects hidden", err)
			}
			if strings.HasPrefix(mode, "ack-") || mode == "partial-error" {
				if strings.Contains(strings.Join(f.calls, ","), "inventory") {
					t.Fatal("continued after uncertain ACK", f.calls)
				}
			}
		})
	}
}
