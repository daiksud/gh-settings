package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRulesetRequiredApprovalsWarningsRequireCompleteOwnedEvidenceAndIndependentSinks(t *testing.T) {
	type event struct {
		id      int64
		name    string
		text    settings.ObservationNormalization
		integer settings.RulesetRequiredApprovalsNormalization
	}
	for _, both := range []bool{false, true} {
		for _, desired := range []int{0, -1} {
			for _, sinks := range []int{0, 1, 2, 3} {
				for _, mode := range []string{"canonical", "same11", "negative", "renamed12", "futureall", "malformed", "unrelated", "wrongID", "foreign", "inherited", "denied"} {
					t.Run(fmt.Sprintf("%t/%d/%d/%s", both, desired, sinks, mode), func(t *testing.T) {
						var got []event
						stringSink, intSink := sinks&1 != 0, sinks&2 != 0
						expected := func(name string, enf, q string, count *int) []event {
							var events []event
							if stringSink && enf != "" {
								events = append(events, event{id: 7, name: name, text: settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: enf, Fallback: "disabled"}})
							}
							if intSink && count != nil {
								events = append(events, event{id: 7, name: name, integer: settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: *count, Fallback: 0}})
							}
							if stringSink && q != "" {
								events = append(events, event{id: 7, name: name, text: settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: q, Fallback: "errors"}})
							}
							return events
						}
						preCount := 11
						preEnf, preQ := "", ""
						if both {
							preEnf, preQ = "evaluate", "FUTURE_SEVERITY"
						}
						want := expected("approvals", preEnf, preQ, &preCount)
						client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
							body := approvalsIntendedPayload(desired, both)
							body["id"], body["source_type"], body["source"] = 7, "Repository", "owner/current"
							if req.Method == "PUT" {
								if !reflect.DeepEqual(got, want) {
									t.Fatal("prewrite evidence/order", got, want)
								}
							} else if req.Method == "GET" {
								if mode == "denied" {
									return labelsResponse(req, 403, `{"message":"denied"}`), nil
								}
								if mode != "canonical" {
									body["rules"] = approvalsIntendedPayload(0, both)["rules"]
									for _, r := range body["rules"].([]any) {
										rule := r.(map[string]any)
										if rule["type"] == "pull_request" {
											value := any(11)
											if mode == "negative" {
												value = -1
											}
											if mode == "renamed12" || mode == "futureall" {
												value = 12
											}
											if mode == "malformed" {
												value = nil
											}
											rule["parameters"].(map[string]any)["required_approving_review_count"] = value
										}
										if rule["type"] == "code_quality" && mode == "futureall" {
											rule["parameters"].(map[string]any)["severity"] = "OTHER_SEVERITY"
										}
									}
								}
								switch mode {
								case "renamed12":
									body["name"] = "fresh approvals"
								case "futureall":
									body["enforcement"] = "evaluate"
								case "unrelated":
									body["future"] = true
									body["enforcement"] = "evaluate"
								case "wrongID":
									body["id"] = 9
								case "foreign":
									body["source"] = "owner/other"
								case "inherited":
									body["source_type"] = "Organization"
								}
							} else {
								t.Fatal(req.Method)
							}
							data, _ := json.Marshal(body)
							return labelsResponse(req, 200, string(data)), nil
						}))
						if stringSink {
							client.RulesetNormalizationWarning = func(id int64, name string, r settings.ObservationNormalization) {
								got = append(got, event{id: id, name: name, text: r})
							}
						}
						if intSink {
							client.RulesetRequiredApprovalsNormalizationWarning = func(id int64, name string, r settings.RulesetRequiredApprovalsNormalization) {
								got = append(got, event{id: id, name: name, integer: r})
							}
						}
						_, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, rulesetApprovalsChange(t, desired, both))
						if (err == nil) != (mode == "canonical") {
							t.Fatal("diagnostic projection confirmed raw drift", err)
						}
						name := "approvals"
						freshEnf, freshQ := "", ""
						var freshCount *int
						switch mode {
						case "same11":
							value := 11
							freshCount = &value
						case "negative":
							value := -1
							freshCount = &value
						case "renamed12":
							value := 12
							freshCount = &value
							name = "fresh approvals"
						case "futureall":
							value := 12
							freshCount = &value
							freshEnf = "evaluate"
							if both {
								freshQ = "OTHER_SEVERITY"
							}
						}
						want = append(want, expected(name, freshEnf, freshQ, freshCount)...)
						if !reflect.DeepEqual(got, want) {
							t.Fatal("scope/order/typedrecords", got, want)
						}
					})
				}
			}
		}
	}
}
