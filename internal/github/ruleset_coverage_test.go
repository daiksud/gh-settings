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

func coveragePolicyJSON(parameters string) string {
	return strings.Replace(basicRulesetJSON, `"rules":[{"type":"deletion"},{"type":"non_fast_forward"}]`, `"rules":[{"type":"code_coverage","parameters":`+parameters+`}]`, 1)
}

func TestCoverageReadRejectsIncompleteOrUnrepresentablePolicy(t *testing.T) {
	for _, params := range []string{`{}`, `null`, `{"minimum_coverage":80}`, `{"max_coverage_drop":0}`, `{"minimum_coverage":null,"max_coverage_drop":0}`, `{"minimum_coverage":"80","max_coverage_drop":0}`, `{"minimum_coverage":80,"max_coverage_drop":101}`, `{"minimum_coverage":80,"max_coverage_drop":0,"future":true}`, `{"minimum_coverage":80,"max_coverage_drop":0,"severity":"errors"}`} {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(coveragePolicyJSON(params)), &raw)
		if _, _, err := decodeObservedRulesetPolicy(raw); err == nil {
			t.Fatal(params)
		}
	}
}

func TestCoverageMutationAndRemovalRequireExactFreshReadback(t *testing.T) {
	beforeJSON := coveragePolicyJSON(`{"minimum_coverage":80.25,"max_coverage_drop":0}`)
	before, err := decodeRulesetPolicyFromTest([]byte(beforeJSON))
	if err != nil {
		t.Fatal(err)
	}
	for _, remove := range []bool{false, true} {
		for _, mode := range []string{"canonical", "stale", "missing", "wrongID", "foreign", "unknown"} {
			t.Run(mode+map[bool]string{true: "-removal", false: "-update"}[remove], func(t *testing.T) {
				after := before
				after.Rules = rulesetPtr([]settings.RulesetRule{})
				afterJSON := strings.Replace(beforeJSON, `"rules":[{"type":"code_coverage","parameters":{"minimum_coverage":80.25,"max_coverage_drop":0}}]`, `"rules":[]`, 1)
				if !remove {
					after.Rules = rulesetPtr([]settings.RulesetRule{{Type: "code_coverage", Parameters: &settings.RuleParameters{MinimumCoverage: rulesetPtr(90.5), MaxCoverageDrop: rulesetPtr(1.25)}}})
					afterJSON = coveragePolicyJSON(`{"minimum_coverage":90.5,"max_coverage_drop":1.25}`)
				}
				change, err := settings.NewRulesetChange("update", 7, &before, &after)
				if err != nil {
					t.Fatal(err)
				}
				var methods []string
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					methods = append(methods, req.Method)
					body := afterJSON
					if req.Method == "GET" {
						switch mode {
						case "stale":
							body = beforeJSON
						case "missing":
							body = coveragePolicyJSON(`{"minimum_coverage":90.5}`)
						case "wrongID":
							body = strings.Replace(body, `"id":7`, `"id":8`, 1)
						case "foreign":
							body = strings.Replace(body, `"source":"Owner/Current"`, `"source":"owner/other"`, 1)
						case "unknown":
							body = strings.Replace(body, `"enforcement":"active"`, `"enforcement":"FUTURE_MODE"`, 1)
						}
					}
					return labelsResponse(req, 200, body), nil
				}))
				id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
				if id != 7 || (err != nil) != (mode != "canonical") || !reflect.DeepEqual(methods, []string{"PUT", "GET"}) {
					t.Fatal(id, err, methods)
				}
			})
		}
	}
}
