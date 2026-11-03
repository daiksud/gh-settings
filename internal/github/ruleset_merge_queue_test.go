package github

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func queueFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../tests/fixtures/ruleset-merge-queue.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRulesetMergeQueueRoundtripAndIncompleteReads(t *testing.T) {
	data := queueFixture(t)
	policy, err := decodeRulesetPolicyFromTest(data)
	if err != nil {
		t.Fatal(err)
	}
	obs, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 26024001, Policy: policy}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := obs.ExportDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	source := uiExportBase()
	source.Rulesets = &exported
	ui, err := UIFromExportDeclaration(source)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := workspace.EncodeConfig(ui)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "require_merge_queue:") || !strings.Contains(string(encoded), "merge_method: squash") {
		t.Fatal(string(encoded))
	}
	decoded, err := workspace.DecodeConfig(encoded)
	if err != nil {
		t.Fatal(err)
	}
	desired, err := DeclarationFromUI(decoded)
	if err != nil {
		t.Fatal(err)
	}
	changes, err := settings.ReconcileRulesets(obs, desired.Rulesets)
	if err != nil || len(changes) != 0 {
		t.Fatal("roundtrip drift", changes, err)
	}
	for _, key := range []string{"check_response_timeout_minutes", "grouping_strategy", "max_entries_to_build", "max_entries_to_merge", "merge_method", "min_entries_to_merge", "min_entries_to_merge_wait_minutes"} {
		for _, mode := range []string{"missing", "null", "wrong-type"} {
			t.Run(key+mode, func(t *testing.T) {
				var raw map[string]any
				json.Unmarshal(data, &raw)
				p := analysisParams(t, raw, "merge_queue")
				switch mode {
				case "missing":
					delete(p, key)
				case "null":
					p[key] = nil
				case "wrong-type":
					p[key] = true
				}
				body, _ := json.Marshal(raw)
				if _, err := decodeRulesetPolicyFromTest(body); err == nil {
					t.Fatal("accepted incomplete queue", string(body))
				}
			})
		}
	}
}
func TestRulesetMergeQueueMutationVerifiesResponseAndIndependentReadback(t *testing.T) {
	policy, err := decodeRulesetPolicyFromTest(queueFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"create", "update", "remove"} {
		for _, mode := range []string{"ok", "response-drift", "readback-drift", "readback403", "write403"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				after := policy
				if operation == "remove" {
					after.Rules = &[]settings.RulesetRule{{Type: "deletion"}}
				}
				var before *settings.RulesetDeclaration
				id, kind := int64(0), "create"
				if operation != "create" {
					before, id, kind = &policy, 26024001, "update"
				}
				if operation == "update" {
					rules := append([]settings.RulesetRule{}, *after.Rules...)
					for i := range rules {
						if rules[i].Type == "merge_queue" {
							p := *rules[i].Parameters
							method := "REBASE"
							p.MergeMethod = &method
							rules[i].Parameters = &p
						}
					}
					after.Rules = &rules
				}
				change, err := settings.NewRulesetChange(kind, id, before, &after)
				if err != nil {
					t.Fatal(err)
				}
				after = *change.After()
				calls := 0
				client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method == "GET" {
						if r.URL.Path != "/repos/owner/current/rulesets/26024001" || r.URL.Query().Get("includes_parents") != "false" {
							t.Fatal("unexpected readback target", r.URL)
						}
						if mode == "readback403" {
							return labelsResponse(r, 403, `{"message":"denied"}`), nil
						}
						body, _ := json.Marshal(after)
						if mode == "readback-drift" {
							body, _ = json.Marshal(policy)
							if operation == "create" {
								body = []byte(strings.Replace(string(body), `"merge_method":"SQUASH"`, `"merge_method":"MERGE"`, 1))
							}
						}
						return rulesetMutationResponse(t, r, 200, string(body), 26024001), nil
					}
					wantMethod, wantPath := "POST", "/repos/owner/current/rulesets"
					if kind == "update" {
						wantMethod, wantPath = "PUT", wantPath+"/26024001"
					}
					if r.Method != wantMethod || r.URL.Path != wantPath {
						t.Fatal(r.Method, r.URL)
					}
					body, _ := io.ReadAll(r.Body)
					want, _ := json.Marshal(after)
					var gotJSON, wantJSON any
					json.Unmarshal(body, &gotJSON)
					json.Unmarshal(want, &wantJSON)
					delete(wantJSON.(map[string]any), "id")
					if !reflect.DeepEqual(gotJSON, wantJSON) {
						t.Fatal("changed payload", string(body), string(want))
					}
					if mode == "write403" {
						return labelsResponse(r, 403, `{"message":"denied"}`), nil
					}
					if mode == "response-drift" {
						body = []byte(strings.Replace(string(body), `"enforcement":"active"`, `"enforcement":"disabled"`, 1))
					}
					return rulesetMutationResponse(t, r, 200, string(body), 26024001), nil
				}))
				_, err = client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
				if (err != nil) != (mode != "ok") {
					t.Fatal(calls, err)
				}
				wantCalls := 2
				if mode == "response-drift" || mode == "write403" {
					wantCalls = 1
				}
				if calls != wantCalls {
					t.Fatal("unexpected retry/readback", calls, err)
				}
				if mode == "readback403" || mode == "readback-drift" || mode == "response-drift" {
					if !strings.Contains(err.Error(), "writes may already have applied") {
						t.Fatal("uncertainty lost", err)
					}
				}
			})
		}
	}
}
