package github

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func rulesetEnforcementChange(t *testing.T, desired string, normalized bool) settings.RulesetChange {
	t.Helper()
	before, err := decodeRulesetPolicyFromTest([]byte(strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"disabled"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	id := int64(7)
	before.ID = &id
	after, err := decodeRulesetPolicyFromTest([]byte(strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"`+desired+`"`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	afterID := id
	after.ID = &afterID
	var change settings.RulesetChange
	if normalized {
		change, err = settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, settings.RulesetNormalizations{Enforcement: settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}})
	} else {
		change, err = settings.NewRulesetChange("update", 7, &before, &after)
	}
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func TestNormalizedRulesetBasicUpdatesRequireStrictFreshGET(t *testing.T) {
	for _, desired := range []string{"active", "disabled"} {
		for _, response := range []string{"canonical", "evaluate", "unknown", "null", "wrongID"} {
			for _, readback := range []string{"canonical", "evaluate", "unknown", "null", "wrongID"} {
				t.Run(desired+response+readback, func(t *testing.T) {
					var operations []string
					change := rulesetEnforcementChange(t, desired, true)
					client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
						operations = append(operations, req.Method)
						mode := readback
						if req.Method == "PUT" {
							mode = response
							var payload map[string]any
							data, err := io.ReadAll(req.Body)
							if err != nil {
								t.Fatal(err)
							}
							if err := json.Unmarshal(data, &payload); err != nil {
								t.Fatal(err)
							}
							want := map[string]any{
								"name": "basic", "target": "branch", "enforcement": desired,
								"conditions":    map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}},
								"bypass_actors": []any{},
								"rules":         []any{map[string]any{"type": "deletion"}, map[string]any{"type": "non_fast_forward"}},
							}
							if !reflect.DeepEqual(payload, want) || payload["id"] != nil {
								t.Fatal("payload ownership changed", payload, want)
							}
						} else if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
							t.Fatal(req.Method, req.URL)
						}
						if req.URL.Path != "/repos/owner/current/rulesets/7" {
							t.Fatal("wrong selected target", req.URL)
						}
						body := strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"`+desired+`"`, 1)
						switch mode {
						case "evaluate":
							body = strings.Replace(body, `"enforcement":"`+desired+`"`, `"enforcement":"evaluate"`, 1)
						case "unknown":
							body = strings.Replace(body, `"enforcement":"`+desired+`"`, `"enforcement":"FUTURE_MODE"`, 1)
						case "null":
							body = strings.Replace(body, `"enforcement":"`+desired+`"`, `"enforcement":null`, 1)
						case "wrongID":
							body = strings.Replace(body, `"id":7`, `"id":8`, 1)
						}
						return labelsResponse(req, 200, body), nil
					}))
					id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
					wantErr := response != "canonical" || readback != "canonical"
					wantOperations := []string{"PUT"}
					if response == "canonical" {
						wantOperations = append(wantOperations, "GET")
					}
					if (err != nil) != wantErr || id <= 0 || !reflect.DeepEqual(operations, wantOperations) {
						t.Fatal(id, err, operations, wantOperations)
					}
					if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
						t.Fatal("uncertainty lost", err)
					}
				})
			}
		}
	}
	var calls []string
	client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		calls = append(calls, req.Method)
		return labelsResponse(req, 200, basicRulesetJSON), nil
	}))
	if _, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, rulesetEnforcementChange(t, "active", false)); err != nil || !reflect.DeepEqual(calls, []string{"PUT"}) {
		t.Fatal("ordinary basic update boundary changed", calls, err)
	}
}
