package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func scanningAlertsReadBody(alerts any, companions bool) map[string]any {
	body := teamApprovalsReadBody(0, companions)
	tools := []any{map[string]any{"tool": "z tool", "alerts_threshold": alerts, "security_alerts_threshold": "all"}, map[string]any{"tool": "CodeQL", "alerts_threshold": "none", "security_alerts_threshold": "high_or_higher"}}
	body["rules"] = append(body["rules"].([]any), map[string]any{"type": "code_scanning", "parameters": map[string]any{"code_scanning_tools": tools}})
	return body
}
func scanningTools(body map[string]any) []any {
	for _, r := range body["rules"].([]any) {
		rule := r.(map[string]any)
		if rule["type"] == "code_scanning" {
			return rule["parameters"].(map[string]any)["code_scanning_tools"].([]any)
		}
	}
	return nil
}

func TestScanningAlertsProjectorAndCompleteReaderRetainExactNames(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, alerts := range []string{"none", "errors", "errors_and_warnings", "all", "", " ", "ERRORS", "future"} {
			encoded, _ := json.Marshal(scanningAlertsReadBody(alerts, companions))
			var raw map[string]json.RawMessage
			json.Unmarshal(encoded, &raw)
			original := bytes.Clone(raw["rules"])
			projected, records, err := prepareRulesetScanningAlertsRules(raw["rules"])
			if err != nil || !bytes.Equal(raw["rules"], original) {
				t.Fatal(err)
			}
			_, supported := settings.RulesetScanningAlertsChoiceForValue(alerts)
			if len(records) != 0 && supported || len(records) != 1 && !supported {
				t.Fatal(alerts, records)
			}
			if !supported && (records[0].Tool != "z tool" || records[0].RawValue != alerts || records[0].Fallback != "errors") {
				t.Fatal(records)
			}
			if len(projected) == 0 {
				t.Fatal("empty projection")
			}
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				if req.URL.Path == "/repos/owner/current/rulesets" {
					return labelsResponse(req, 200, `[{"id":7,"source_type":"Repository","source":"owner/current"}]`), nil
				}
				return labelsResponse(req, 200, string(encoded)), nil
			}))
			warnings := 0
			client.RulesetScanningAlertsNormalizationWarning = func(int64, string, settings.RulesetScanningAlertsNormalization) { warnings++ }
			observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
			if err != nil {
				t.Fatal(alerts, err)
			}
			values, err := observed.Values()
			if err != nil || len(values) != 1 {
				t.Fatal(values, err)
			}
			if !reflect.DeepEqual(values[0].ScanningAlertsNormalizations, records) || warnings != 0 {
				t.Fatal("reader lost evidence or warned", values, warnings)
			}
			actual, err := values[0].RawCodeScanningTools()
			if err != nil || len(actual) != 2 || actual[1].Tool != "z tool" || actual[1].AlertsThreshold != alerts || actual[1].SecurityAlertsThreshold != "all" || actual[0].AlertsThreshold != "none" {
				t.Fatal(actual, err)
			}
			if companions && (len(values[0].TeamApprovalsNormalizations) != 1 || values[0].RequiredApprovalsNormalization.RawValue != 11 || values[0].EnforcementNormalization.RawValue != "evaluate" || values[0].MergeMethodsNormalization.IsZero() || len(values[0].BypassModeNormalizations) != 1 || values[0].QualitySeverityNormalization.RawValue != "FUTURE_SEVERITY") {
				t.Fatal("companion evidence lost", values)
			}
			if (verifyReviewRulesetResponse(values[0].Policy, raw) != nil) != (companions || !supported) {
				t.Fatal("strict confirmation projected")
			}
		}
	}
}

func TestScanningAlertsAcquisitionRejectsPartialOrMalformedTools(t *testing.T) {
	for _, mode := range []string{"absent", "empty", "null alerts", "missing alerts", "number alerts", "null tool", "missing tool", "blank tool", "control tool", "duplicate", "null security", "missing security", "unknown security", "null tools", "unknown member", "later denied", "pagination denied"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.Path == "/repos/owner/current/rulesets" {
					if req.URL.Query().Get("page") == "2" {
						return labelsResponse(req, 403, `{}`), nil
					}
					items := []any{map[string]any{"id": 7, "source_type": "Repository", "source": "owner/current"}, map[string]any{"id": 9, "source_type": "Repository", "source": "owner/current"}}
					if mode == "pagination denied" {
						for len(items) < 100 {
							items = append(items, map[string]any{"source_type": "Organization", "source": "owner"})
						}
					}
					encoded, _ := json.Marshal(items)
					return labelsResponse(req, 200, string(encoded)), nil
				}
				body := scanningAlertsReadBody("future", false)
				if strings.HasSuffix(req.URL.Path, "/9") {
					body["id"], body["name"] = 9, "later"
					if mode == "later denied" {
						return labelsResponse(req, 403, `{}`), nil
					}
					tools := scanningTools(body)
					first := tools[0].(map[string]any)
					rules := body["rules"].([]any)
					params := rules[len(rules)-1].(map[string]any)["parameters"].(map[string]any)
					switch mode {
					case "absent":
						body["rules"] = []any{}
					case "empty":
						params["code_scanning_tools"] = []any{}
					case "null alerts":
						first["alerts_threshold"] = nil
					case "missing alerts":
						delete(first, "alerts_threshold")
					case "number alerts":
						first["alerts_threshold"] = 3
					case "null tool":
						first["tool"] = nil
					case "missing tool":
						delete(first, "tool")
					case "blank tool":
						first["tool"] = " "
					case "control tool":
						first["tool"] = "tool\n"
					case "duplicate":
						tools[1] = first
					case "null security":
						first["security_alerts_threshold"] = nil
					case "missing security":
						delete(first, "security_alerts_threshold")
					case "unknown security":
						first["security_alerts_threshold"] = "future"
					case "null tools":
						params["code_scanning_tools"] = nil
					case "unknown member":
						first["other"] = true
					}
				}
				encoded, _ := json.Marshal(body)
				return labelsResponse(req, 200, string(encoded)), nil
			}))
			observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
			if mode != "absent" && mode != "empty" {
				if err == nil || !reflect.DeepEqual(observed, settings.RulesetObservation{}) {
					t.Fatal("partial or repaired", observed, err)
				}
				if mode == "pagination denied" && calls != 2 {
					t.Fatal("read before complete pages", calls)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				values, _ := observed.Values()
				raw, err := values[1].RawCodeScanningTools()
				if err != nil || len(raw) != 0 || (mode == "absent") != (raw == nil) {
					t.Fatal(mode, raw, err)
				}
			}
		})
	}
	// Complete strings with exact, distinct names stay distinct, including spaces/case.
	body := scanningAlertsReadBody("future", false)
	tools := scanningTools(body)
	tools[0].(map[string]any)["tool"] = " CodeQL "
	data, _ := json.Marshal(body)
	var raw map[string]json.RawMessage
	json.Unmarshal(data, &raw)
	_, records, err := decodeObservedRulesetPolicy(raw)
	if err != nil || len(records.ScanningAlerts) != 1 || records.ScanningAlerts[0].Tool != " CodeQL " {
		t.Fatal(fmt.Sprint(records), err)
	}
}
