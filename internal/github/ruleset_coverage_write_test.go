package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func coverageBody(t *testing.T, minimum, drop float64) map[string]any {
	t.Helper()
	body := scopeBody(t, false)
	body["rules"] = append(body["rules"].([]any),
		map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "errors"}},
		map[string]any{"type": "code_coverage", "parameters": map[string]any{"minimum_coverage": minimum, "max_coverage_drop": drop}})
	return body
}

func coveragePolicyFromBody(t *testing.T, body map[string]any) settings.RulesetDeclaration {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := decodeRulesetPolicyFromTest(data)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func withoutCoverage(body map[string]any) {
	var rules []any
	for _, rule := range body["rules"].([]any) {
		if rule.(map[string]any)["type"] != "code_coverage" {
			rules = append(rules, rule)
		}
	}
	body["rules"] = rules
}

func TestRulesetCoverageWritesConfirmCompletePolicy(t *testing.T) {
	for _, operation := range []string{"create", "update", "remove"} {
		for _, failure := range []string{"", "ack minimum", "ack missing", "readback drop", "readback missing", "readback absent", "readback denied"} {
			t.Run(operation+"/"+failure, func(t *testing.T) {
				beforeBody, afterBody := coverageBody(t, 80.5, 2.25), coverageBody(t, 90.25, 0)
				additionalApproval := operation == "update"
				scopedPR(t, beforeBody)["require_extra_approval_for_unattributed_changes"] = additionalApproval
				scopedPR(t, afterBody)["require_extra_approval_for_unattributed_changes"] = additionalApproval
				afterBody["name"] = "changed"
				if operation == "remove" {
					withoutCoverage(afterBody)
				}
				before, after := coveragePolicyFromBody(t, beforeBody), coveragePolicyFromBody(t, afterBody)
				kind, id, prior := "update", int64(7), &before
				if operation == "create" {
					kind, id, prior = "create", 0, nil
				}
				change, err := settings.NewRulesetChange(kind, id, prior, &after)
				if err != nil {
					t.Fatal(err)
				}
				writes, reads := 0, 0
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					body := afterBody
					if req.Method == "GET" {
						reads++
						if writes == 0 {
							body = beforeBody
						}
						if writes > 0 && failure == "readback denied" {
							return labelsResponse(req, 403, "{}"), nil
						}
					} else {
						writes++
						if (kind == "create" && req.Method != "POST") || (kind == "update" && req.Method != "PUT") {
							t.Fatal(req.Method)
						}
						var payload map[string]any
						if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
							t.Fatal(err)
						}
						if _, hasID := payload["id"]; hasID || payload["name"] != "changed" || payload["target"] != "branch" ||
							analysisParams(t, payload, "code_quality")["severity"] != "errors" ||
							scopedPR(t, payload)["require_extra_approval_for_unattributed_changes"] != additionalApproval {
							t.Fatal("lost owned identity or sibling policy", payload)
						}
						if operation != "remove" && !reflect.DeepEqual(analysisParams(t, payload, "code_coverage"), map[string]any{"minimum_coverage": 90.25, "max_coverage_drop": float64(0)}) {
							t.Fatal("incomplete requested thresholds", payload)
						}
						if operation == "remove" {
							for _, rule := range payload["rules"].([]any) {
								if rule.(map[string]any)["type"] == "code_coverage" {
									t.Fatal("coverage removal ignored")
								}
							}
						}
					}
					phaseFailure := writes > 0 && ((req.Method != "GET" && strings.HasPrefix(failure, "ack")) || (req.Method == "GET" && strings.HasPrefix(failure, "readback")))
					if phaseFailure {
						// Copy so truthful ACK and later GET are independent observations.
						data, _ := json.Marshal(body)
						json.Unmarshal(data, &body)
						if operation == "remove" {
							body["rules"] = append(body["rules"].([]any), map[string]any{"type": "code_coverage", "parameters": map[string]any{"minimum_coverage": 90.25, "max_coverage_drop": 0}})
						} else if strings.HasSuffix(failure, "absent") {
							withoutCoverage(body)
						} else {
							p := analysisParams(t, body, "code_coverage")
							if strings.HasSuffix(failure, "missing") {
								delete(p, "minimum_coverage")
							} else if strings.HasSuffix(failure, "minimum") {
								p["minimum_coverage"] = 90.5
							} else {
								p["max_coverage_drop"] = 1.0
							}
						}
					}
					data, _ := json.Marshal(body)
					return labelsResponse(req, 200, string(data)), nil
				}))
				gotID, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
				if (err != nil) != (failure != "") || writes != 1 || gotID != 7 {
					t.Fatal(gotID, writes, reads, err)
				}
				if failure != "" && !strings.Contains(err.Error(), "may already have applied") {
					t.Fatal("lost uncertainty", err)
				}
				wantReads := 1
				if kind == "update" {
					wantReads++
				}
				if strings.HasPrefix(failure, "ack") {
					wantReads--
				}
				if reads != wantReads {
					t.Fatal("ACK substituted for readback", reads, wantReads)
				}
			})
		}
	}
}

func TestRulesetCoveragePreflightRetainsSiblingDriftGuards(t *testing.T) {
	for _, mode := range []string{"coverage added", "coverage removed", "minimum", "drop", "quality", "additional approval"} {
		for _, removeIntent := range []bool{false, true} {
			beforeBody := coverageBody(t, 80.5, 2.25)
			if mode == "coverage added" {
				withoutCoverage(beforeBody)
			}
			before := coveragePolicyFromBody(t, beforeBody)
			afterBody := coverageBody(t, 80.5, 2.25)
			afterBody["name"] = "renamed"
			if removeIntent {
				withoutCoverage(afterBody)
			}
			after := coveragePolicyFromBody(t, afterBody)
			change, err := settings.NewRulesetChange("update", 7, &before, &after)
			if err != nil {
				t.Fatal(err)
			}
			current := coverageBody(t, 80.5, 2.25)
			switch mode {
			case "coverage removed":
				withoutCoverage(current)
			case "minimum":
				analysisParams(t, current, "code_coverage")["minimum_coverage"] = 0.0
			case "drop":
				analysisParams(t, current, "code_coverage")["max_coverage_drop"] = 0.0
			case "quality":
				analysisParams(t, current, "code_quality")["severity"] = "warnings"
			case "additional approval":
				scopedPR(t, current)["require_extra_approval_for_unattributed_changes"] = true
			}
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" {
					t.Fatal("drift authorized mutation", mode)
				}
				data, _ := json.Marshal(current)
				return labelsResponse(req, 200, string(data)), nil
			}))
			if id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change); err == nil || id != 0 || calls != 1 {
				t.Fatal(mode, removeIntent, id, calls, err)
			}
		}
	}
}

func TestCoverageRulesetPruneRequiresExactSnapshot(t *testing.T) {
	for _, mode := range []string{"matching", "changed", "missing"} {
		body := coverageBody(t, 80.5, 0)
		before := coveragePolicyFromBody(t, body)
		id := int64(7)
		before.ID = &id
		change, err := settings.NewRulesetChange("delete", id, &before, nil)
		if err != nil {
			t.Fatal(err)
		}
		if mode == "changed" {
			analysisParams(t, body, "code_coverage")["minimum_coverage"] = 81.0
		}
		if mode == "missing" {
			delete(analysisParams(t, body, "code_coverage"), "max_coverage_drop")
		}
		deletes, reads := 0, 0
		client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method == "DELETE" {
				deletes++
				return labelsResponse(req, 204, ""), nil
			}
			if req.Method != "GET" {
				t.Fatal(req.Method)
			}
			reads++
			if deletes > 0 {
				return labelsResponse(req, 200, "[]"), nil
			}
			data, _ := json.Marshal(body)
			return labelsResponse(req, 200, string(data)), nil
		}))
		got, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
		if mode == "matching" {
			if err != nil || got != 7 || deletes != 1 || reads != 2 {
				t.Fatal(got, deletes, reads, err)
			}
		} else if err == nil || got != 0 || deletes != 0 || reads != 1 {
			t.Fatal(got, deletes, reads, err)
		}
	}
}
