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

const qualityRulesetJSON = `{"id":7,"name":"quality","source_type":"Repository","source":"owner/current","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"code_quality","parameters":{"severity":"errors"}}]}`

func rulesetQualityChange(t *testing.T, desired string, both bool) settings.RulesetChange {
	t.Helper()
	before, err := decodeRulesetPolicyFromTest([]byte(qualityRulesetJSON))
	if err != nil {
		t.Fatal(err)
	}
	after, err := decodeRulesetPolicyFromTest([]byte(qualityRulesetJSON))
	if err != nil {
		t.Fatal(err)
	}
	idBefore, idAfter := int64(7), int64(7)
	before.ID, after.ID = &idBefore, &idAfter
	records := settings.RulesetNormalizations{QualitySeverity: settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "FUTURE_SEVERITY", Fallback: "errors"}}
	if desired == "removed" {
		after.Rules = &[]settings.RulesetRule{}
	} else {
		(*after.Rules)[0].Parameters.Severity = &desired
	}
	if both {
		before.Enforcement, after.Enforcement = actionsPointer("disabled"), actionsPointer("disabled")
		records.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
	}
	change, err := settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func TestNormalizedRulesetQualityHTTPVerificationRequiresExactSeverityOrRemoval(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, test := range []struct {
			name, desired, response, readback string
			wantErr                           bool
		}{
			{"equal errors", "errors", "canonical", "canonical", false}, {"warnings", "warnings", "canonical", "canonical", false}, {"notes", "notes", "canonical", "canonical", false}, {"all", "all", "canonical", "canonical", false}, {"remove", "removed", "canonical", "canonical", false},
			{"ignored equal", "errors", "canonical", "future", true}, {"ignored different", "warnings", "canonical", "future", true}, {"ignored removal unknown", "removed", "canonical", "future", true}, {"ignored removal canonical", "removed", "canonical", "retained rule", true},
			{"bad acknowledgment", "errors", "future", "canonical", true}, {"ack retained rule", "removed", "retained rule", "canonical", true},
			{"null readback", "errors", "canonical", "null", true}, {"ID readback", "errors", "canonical", "wrong ID", true}, {"foreign readback", "errors", "canonical", "foreign", true}, {"denied readback", "errors", "canonical", "denied", true},
		} {
			t.Run(test.name, func(t *testing.T) {
				var operations []string
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					operations = append(operations, req.Method)
					if req.URL.Path != "/repos/owner/current/rulesets/7" {
						t.Fatal("wrong selected ID", req.URL)
					}
					mode := test.readback
					if req.Method == "PUT" {
						mode = test.response
						var got map[string]any
						if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
							t.Fatal(err)
						}
						enforcement := "active"
						if both {
							enforcement = "disabled"
						}
						rules := []any{}
						if test.desired != "removed" {
							rules = append(rules, map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": test.desired}})
						}
						want := map[string]any{"name": "quality", "target": "branch", "enforcement": enforcement, "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}}, "bypass_actors": []any{}, "rules": rules}
						if !reflect.DeepEqual(got, want) {
							t.Fatal("payload differs from independent intent", got, want)
						}
					} else if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
						t.Fatal(req.Method, req.URL)
					}
					if mode == "denied" {
						return labelsResponse(req, 403, `{"message":"denied"}`), nil
					}
					var body map[string]any
					json.Unmarshal([]byte(qualityRulesetJSON), &body)
					if both {
						body["enforcement"] = "disabled"
					}
					if test.desired == "removed" {
						body["rules"] = []any{}
					} else {
						body["rules"] = []any{map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": test.desired}}}
					}
					switch mode {
					case "future":
						body["rules"] = []any{map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "FUTURE_SEVERITY"}}}
					case "retained rule":
						body["rules"] = []any{map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "errors"}}}
					case "null":
						body["rules"] = []any{map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": nil}}}
					case "wrong ID":
						body["id"] = 8
					case "foreign":
						body["source"] = "owner/other"
					}
					encoded, _ := json.Marshal(body)
					return labelsResponse(req, 200, string(encoded)), nil
				}))
				id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, rulesetQualityChange(t, test.desired, both))
				wantOperations := []string{"PUT", "GET"}
				if test.response != "canonical" {
					wantOperations = []string{"PUT"}
				}
				if (err != nil) != test.wantErr || id != 7 || !reflect.DeepEqual(operations, wantOperations) {
					t.Fatal(id, err, operations, wantOperations)
				}
				if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
					t.Fatal("certainty lost", err)
				}
			})
		}
	}
}
