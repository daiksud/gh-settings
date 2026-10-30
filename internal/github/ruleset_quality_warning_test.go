package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRulesetQualityWarningsComposeOnlyCompleteOwnedFreshEvidence(t *testing.T) {
	type diagnostic struct {
		id     int64
		name   string
		record settings.ObservationNormalization
	}
	for _, both := range []bool{false, true} {
		for _, desired := range []string{"errors", "removed"} {
			for _, test := range []struct {
				mode                           string
				sink                           bool
				freshEnforcement, freshQuality bool
			}{
				{"canonical", true, false, false}, {"retained same name", true, false, true}, {"future quality", true, false, true}, {"future both", true, true, true}, {"retained rule", true, false, false}, {"malformed", true, false, false}, {"unrelated invalid", true, false, false}, {"wrong ID", true, false, false}, {"foreign", true, false, false}, {"inherited", true, false, false}, {"nil sink", false, false, false},
			} {
				t.Run(desired+test.mode, func(t *testing.T) {
					var got []diagnostic
					client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
						var body map[string]any
						json.Unmarshal([]byte(qualityRulesetJSON), &body)
						if both {
							body["enforcement"] = "disabled"
						}
						if desired == "removed" {
							body["rules"] = []any{}
						}
						if req.Method == "PUT" {
							wantPre := 0
							if test.sink {
								wantPre = 1
								if both {
									wantPre = 2
								}
							}
							if len(got) != wantPre {
								t.Fatal("prewrite record scope", got, wantPre)
							}
						} else if req.Method == "GET" {
							body["name"] = "fresh quality"
							switch test.mode {
							case "retained same name", "future quality", "future both", "nil sink", "unrelated invalid":
								body["rules"] = []any{map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "OTHER_SEVERITY"}}}
							case "retained rule":
								body["rules"] = []any{map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "errors"}}}
							case "malformed":
								body["rules"] = []any{map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": nil}}}
							case "wrong ID":
								body["id"] = 8
							case "foreign":
								body["source"] = "owner/other"
							case "inherited":
								body["source_type"] = "Organization"
							}
							if test.mode == "future both" || test.mode == "unrelated invalid" {
								body["enforcement"] = "evaluate"
							}
							if test.mode == "unrelated invalid" {
								body["future"] = true
							}
							if test.mode == "canonical" || test.mode == "retained same name" {
								body["name"] = "quality"
							}
						} else {
							t.Fatal(req.Method)
						}
						encoded, _ := json.Marshal(body)
						return labelsResponse(req, 200, string(encoded)), nil
					}))
					if test.sink {
						client.RulesetNormalizationWarning = func(id int64, name string, n settings.ObservationNormalization) {
							got = append(got, diagnostic{id, name, n})
						}
					}
					_, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, rulesetQualityChange(t, desired, both))
					if (err == nil) != (test.mode == "canonical") {
						t.Fatal("diagnostic projection confirmed raw drift", err)
					}
					var want []diagnostic
					if test.sink {
						if both {
							want = append(want, diagnostic{7, "quality", settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}})
						}
						want = append(want, diagnostic{7, "quality", settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "FUTURE_SEVERITY", Fallback: "errors"}})
						if test.freshEnforcement {
							want = append(want, diagnostic{7, "fresh quality", settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}})
						}
						if test.freshQuality {
							name := "fresh quality"
							if test.mode == "retained same name" {
								name = "quality"
							}
							want = append(want, diagnostic{7, name, settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "OTHER_SEVERITY", Fallback: "errors"}})
						}
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatal("record order/context/scope", got, want)
					}
				})
			}
		}
	}
}
