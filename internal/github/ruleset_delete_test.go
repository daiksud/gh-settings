package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func deletionChange(t *testing.T) settings.RulesetChange {
	t.Helper()
	var raw map[string]json.RawMessage
	json.Unmarshal([]byte(basicRulesetJSON), &raw)
	before, err := decodeRulesetPolicy(raw)
	if err != nil {
		t.Fatal(err)
	}
	id := int64(7)
	before.ID = &id
	c, err := settings.NewRulesetChange("delete", id, &before, nil)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRulesetDeleteRequiresStrictPreflightAndConfirmedInventoryAbsence(t *testing.T) {
	for _, mode := range []string{"success", "owner changed", "id changed", "name changed", "rule changed", "hidden bypass", "unknown rule", "unknown parameter", "delete denied", "delete not found", "unexpected status", "retained", "retained inherited", "retained push", "inventory denied", "inventory null", "inventory missing id", "pagination denied"} {
		t.Run(mode, func(t *testing.T) {
			calls := []string{}
			deletes := 0
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				calls = append(calls, r.Method+" "+r.URL.RequestURI())
				if r.Header.Get("X-GitHub-Api-Version") == "" {
					t.Error("missing pinned API version")
				}
				if r.Method == "DELETE" {
					deletes++
					if r.URL.Path != "/repos/owner/current/rulesets/7" || r.URL.RawQuery != "" {
						t.Error("wrong delete identity", r.URL)
					}
					if r.Body != nil {
						body, _ := io.ReadAll(r.Body)
						if len(body) != 0 {
							t.Error("DELETE has payload", string(body))
						}
					}
					status := 204
					if mode == "delete denied" {
						status = 403
					}
					if mode == "delete not found" {
						status = 404
					}
					if mode == "unexpected status" {
						status = 200
					}
					return labelsResponse(r, status, `{}`), nil
				}
				if r.Method != "GET" || r.URL.Query().Get("includes_parents") != "false" {
					t.Error("unsafe read", r.Method, r.URL)
				}
				if r.URL.Path == "/repos/owner/current/rulesets/7" {
					body := basicRulesetJSON
					if deletes > 0 {
						if mode == "retained push" {
							body = strings.Replace(body, `"target":"branch"`, `"target":"push"`, 1)
						}
						return labelsResponse(r, 200, body), nil
					}
					switch mode {
					case "owner changed":
						body = strings.Replace(body, `"source_type":"Repository"`, `"source_type":"Organization"`, 1)
					case "id changed":
						body = strings.Replace(body, `"id":7`, `"id":8`, 1)
					case "name changed":
						body = strings.Replace(body, `"name":"basic"`, `"name":"other"`, 1)
					case "rule changed":
						body = strings.Replace(body, `"type":"deletion"`, `"type":"required_signatures"`, 1)
					case "hidden bypass":
						body = strings.Replace(body, `"bypass_actors":[],`, ``, 1)
					case "unknown rule":
						body = strings.Replace(body, `"type":"deletion"`, `"type":"future_rule"`, 1)
					case "unknown parameter":
						body = strings.Replace(body, `"type":"deletion"`, `"type":"deletion","parameters":{"future":true}`, 1)
					}
					return labelsResponse(r, 200, body), nil
				}
				if r.URL.Path != "/repos/owner/current/rulesets" {
					t.Fatal(r.URL)
				}
				if deletes != 1 {
					t.Error("verification before deletion", calls)
				}
				if mode == "inventory denied" {
					return labelsResponse(r, 403, `{"message":"denied"}`), nil
				}
				if mode == "inventory null" {
					return labelsResponse(r, 200, `null`), nil
				}
				if mode == "inventory missing id" {
					return labelsResponse(r, 200, `[{"source_type":"Repository","source":"owner/current"}]`), nil
				}
				if mode == "pagination denied" {
					if r.URL.Query().Get("page") == "2" {
						return labelsResponse(r, 403, `{"message":"denied"}`), nil
					}
					items := []string{}
					for i := 1; i <= 100; i++ {
						items = append(items, fmt.Sprintf(`{"id":%d,"source_type":"Organization","source":"owner"}`, 100+i))
					}
					return labelsResponse(r, 200, "["+strings.Join(items, ",")+"]"), nil
				}
				if mode == "retained" || mode == "retained push" {
					return labelsResponse(r, 200, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`), nil
				}
				if mode == "retained inherited" {
					return labelsResponse(r, 200, `[{"id":7,"source_type":"Organization","source":"owner"}]`), nil
				}
				return labelsResponse(r, 200, `[]`), nil
			}))
			_, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, deletionChange(t))
			if (err == nil) != (mode == "success") {
				t.Fatal("outcome", err, calls)
			}
			prewrite := mode == "owner changed" || mode == "id changed" || mode == "name changed" || mode == "rule changed" || mode == "hidden bypass" || mode == "unknown rule" || mode == "unknown parameter"
			if prewrite && deletes != 0 {
				t.Fatal("invalid acquisition deleted", calls)
			}
			if !prewrite && deletes != 1 {
				t.Fatal("unexpected delete attempts", calls)
			}
			if err != nil && !prewrite && !strings.Contains(err.Error(), "write may have applied") {
				t.Fatal("uncertain write not reported", err)
			}
		})
	}
}

func TestRulesetAdditionalApprovalPruneKeepsFreshPolicyGate(t *testing.T) {
	for _, value := range []bool{false, true} {
		for _, drift := range []bool{false, true} {
			t.Run(fmt.Sprintf("%t/drift=%t", value, drift), func(t *testing.T) {
				body := scopeBody(t, value)
				data, _ := json.Marshal(body)
				before, err := decodeRulesetPolicyFromTest(data)
				if err != nil {
					t.Fatal(err)
				}
				before.ID = rulesetPtr(int64(7))
				change, err := settings.NewRulesetChange("delete", 7, &before, nil)
				if err != nil {
					t.Fatal(err)
				}
				calls := []string{}
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					calls = append(calls, req.Method)
					switch len(calls) {
					case 1:
						if req.Method != "GET" || req.URL.Path != "/repos/owner/current/rulesets/7" || req.URL.Query().Get("includes_parents") != "false" {
							t.Fatal(req.Method, req.URL)
						}
						if drift {
							scopedPR(t, body)["require_extra_approval_for_unattributed_changes"] = !value
						}
						data, _ := json.Marshal(body)
						return labelsResponse(req, 200, string(data)), nil
					case 2:
						if req.Method != "DELETE" || req.URL.Path != "/repos/owner/current/rulesets/7" {
							t.Fatal(req.Method, req.URL)
						}
						return labelsResponse(req, 204, ""), nil
					case 3:
						if req.Method != "GET" || req.URL.Path != "/repos/owner/current/rulesets" || req.URL.Query().Get("includes_parents") != "false" {
							t.Fatal(req.Method, req.URL)
						}
						return labelsResponse(req, 200, `[]`), nil
					default:
						t.Fatal("unexpected request", req.Method, req.URL)
						return nil, nil
					}
				}))
				_, err = client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
				if (err != nil) != drift || len(calls) != map[bool]int{false: 3, true: 1}[drift] {
					t.Fatal(calls, err)
				}
			})
		}
	}
}
