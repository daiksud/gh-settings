package github

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func analysisFixture(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../tests/fixtures/ruleset-code-analysis.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func analysisParams(t *testing.T, policy map[string]any, kind string) map[string]any {
	t.Helper()
	for _, value := range policy["rules"].([]any) {
		rule := value.(map[string]any)
		if rule["type"] == kind {
			return rule["parameters"].(map[string]any)
		}
	}
	t.Fatal("missing rule", kind)
	return nil
}

func TestRulesetCodeAnalysisExportRetainsCompleteProtection(t *testing.T) {
	data := analysisFixture(t)
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
	yaml, err := workspace.EncodeConfig(ui)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(yaml), "version: daiksud/gh-settings@v0") || !strings.Contains(string(yaml), "require_code_scanning_results:") || !strings.Contains(string(yaml), "ruleset_name: default") || strings.Contains(string(yaml), "schemaVersion") {
		t.Fatalf("export did not use the UI configuration boundary: %s", yaml)
	}
	decoded, err := workspace.DecodeConfig(yaml)
	if err != nil {
		t.Fatal(string(yaml), err)
	}
	d, err := DeclarationFromUI(decoded)
	if err != nil {
		t.Fatal(err)
	}
	changes, err := settings.ReconcileRulesets(obs, d.Rulesets)
	if err != nil || len(changes) != 0 {
		t.Fatal("export lost protection", changes, err)
	}
	var expected, actual map[string]any
	json.Unmarshal(data, &expected)
	encoded, _ := json.Marshal(d.Rulesets.Values()[0])
	json.Unmarshal(encoded, &actual)
	delete(expected, "source_type")
	delete(expected, "source")
	for _, value := range []map[string]any{expected, actual} {
		slices.SortFunc(value["rules"].([]any), func(a, b any) int {
			return strings.Compare(a.(map[string]any)["type"].(string), b.(map[string]any)["type"].(string))
		})
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("existing protections lost:\nwant %v\ngot %v", expected, actual)
	}
}

func TestRulesetCodeAnalysisReadsRejectUnknownIncompleteAndMixed(t *testing.T) {
	for _, mode := range []string{"missing-tools", "null-tools", "missing-threshold", "null-threshold", "wrong-threshold", "unknown-tool-field", "case-alias", "null-severity", "missing-severity", "unknown-severity", "wrong-type", "mixed"} {
		t.Run(mode, func(t *testing.T) {
			var raw map[string]any
			json.Unmarshal(analysisFixture(t), &raw)
			scan, quality := analysisParams(t, raw, "code_scanning"), analysisParams(t, raw, "code_quality")
			tool := scan["code_scanning_tools"].([]any)[0].(map[string]any)
			switch mode {
			case "missing-tools":
				delete(scan, "code_scanning_tools")
			case "null-tools":
				scan["code_scanning_tools"] = nil
			case "missing-threshold":
				delete(tool, "security_alerts_threshold")
			case "null-threshold":
				tool["alerts_threshold"] = nil
			case "wrong-threshold":
				tool["security_alerts_threshold"] = "high"
			case "unknown-tool-field":
				tool["future"] = true
			case "case-alias":
				tool["Tool"] = tool["tool"]
				delete(tool, "tool")
			case "null-severity":
				quality["severity"] = nil
			case "missing-severity":
				delete(quality, "severity")
			case "unknown-severity":
				quality["severity"] = "none"
			case "wrong-type":
				quality["severity"] = true
			case "mixed":
				scan["severity"] = "errors"
			}
			data, _ := json.Marshal(raw)
			if _, err := decodeRulesetPolicyFromTest(data); err == nil {
				t.Fatal("accepted lossy analysis observation", mode)
			}
		})
	}
}

func TestRulesetCodeAnalysisMutationVerifiesResponseAndIndependentReadback(t *testing.T) {
	policy, err := decodeRulesetPolicyFromTest(analysisFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	// No PR rule: analysis rules themselves must trigger independent readback,
	// including removal of the last analysis rule.
	rules := []settings.RulesetRule{}
	for _, rule := range *policy.Rules {
		if rule.Type == "code_scanning" || rule.Type == "code_quality" {
			rules = append(rules, rule)
		}
	}
	policy.Rules = &rules
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
					// An independent quality threshold change leaves scanning intact.
					rules := append([]settings.RulesetRule{}, *after.Rules...)
					for i := range rules {
						if rules[i].Type == "code_quality" {
							p := *rules[i].Parameters
							severity := "warnings"
							p.Severity = &severity
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
								body = []byte(strings.Replace(string(body), `"severity":"errors"`, `"severity":"all"`, 1))
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
