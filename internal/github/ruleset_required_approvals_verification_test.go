package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

const approvalsRulesetJSON = `{"id":7,"name":"approvals","source_type":"Repository","source":"owner/current","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"pull_request","parameters":{"allowed_merge_methods":["squash"],"dismiss_stale_reviews_on_push":false,"require_code_owner_review":false,"require_extra_approval_for_unattributed_changes":false,"require_last_push_approval":false,"required_approving_review_count":0,"required_review_thread_resolution":false,"required_reviewers":[]}}]}`

func rulesetApprovalsChange(t *testing.T, count int, both bool) settings.RulesetChange {
	t.Helper()
	before, err := decodeRulesetPolicyFromTest([]byte(approvalsRulesetJSON))
	if err != nil {
		t.Fatal(err)
	}
	after, err := decodeRulesetPolicyFromTest([]byte(approvalsRulesetJSON))
	if err != nil {
		t.Fatal(err)
	}
	id := int64(7)
	before.ID, after.ID = &id, &id
	if count == -1 {
		after.Rules = &[]settings.RulesetRule{}
	} else {
		(*after.Rules)[0].Parameters.RequiredApprovingReviewCount = &count
	}
	records := settings.RulesetNormalizations{RequiredApprovals: settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: 11, Fallback: 0}}
	if both {
		before.Enforcement, after.Enforcement = actionsPointer("disabled"), actionsPointer("disabled")
		records.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
		records.QualitySeverity = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "FUTURE_SEVERITY", Fallback: "errors"}
		*before.Rules = append(*before.Rules, settings.RulesetRule{Type: "code_quality", Parameters: &settings.RuleParameters{Severity: actionsPointer("errors")}})
		*after.Rules = append(*after.Rules, settings.RulesetRule{Type: "code_quality", Parameters: &settings.RuleParameters{Severity: actionsPointer("errors")}})
	}
	c, err := settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// approvalsIntendedPayload is literal fixture intent, independent of constructed changes and actual requests.
func approvalsIntendedPayload(count int, both bool) map[string]any {
	rules := []any{}
	if both {
		rules = append(rules, map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "errors"}})
	}
	if count != -1 {
		rules = append(rules, map[string]any{"type": "pull_request", "parameters": map[string]any{"allowed_merge_methods": []any{"squash"}, "dismiss_stale_reviews_on_push": false, "require_code_owner_review": false, "require_extra_approval_for_unattributed_changes": false, "require_last_push_approval": false, "required_approving_review_count": float64(count), "required_review_thread_resolution": false, "required_reviewers": []any{}}})
	}
	enforcement := "active"
	if both {
		enforcement = "disabled"
	}
	return map[string]any{"name": "approvals", "target": "branch", "enforcement": enforcement, "conditions": map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}}, "bypass_actors": []any{}, "rules": rules}
}

func TestNormalizedRulesetRequiredApprovalsHTTPVerificationRequiresRawCountOrAbsence(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, test := range []struct {
			name      string
			count     int
			ack, read string
			wantErr   bool
		}{
			{"zero", 0, "canonical", "canonical", false}, {"ten", 10, "canonical", "canonical", false}, {"remove", -1, "canonical", "canonical", false},
			{"retained11", 0, "canonical", "11", true}, {"retainednegative", 0, "canonical", "negative", true}, {"differentcount", 10, "canonical", "11", true},
			{"remove retained11", -1, "canonical", "11", true}, {"remove retainedcanonical", -1, "canonical", "retained", true},
			{"badack", 0, "11", "canonical", true}, {"removal badack", -1, "retained", "canonical", true},
			{"nullread", 0, "canonical", "null", true}, {"wrongID", 0, "canonical", "wrongID", true}, {"foreign", 0, "canonical", "foreign", true}, {"denied", 0, "canonical", "denied", true},
		} {
			t.Run(fmt.Sprintf("%t/%s", both, test.name), func(t *testing.T) {
				var operations []string
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					operations = append(operations, req.Method)
					if req.URL.Path != "/repos/owner/current/rulesets/7" {
						t.Fatal(req.URL)
					}
					mode := test.read
					if req.Method == "PUT" {
						mode = test.ack
						var got map[string]any
						if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
							t.Fatal(err)
						}
						if want := approvalsIntendedPayload(test.count, both); !reflect.DeepEqual(got, want) {
							t.Fatal("payload differs from independent intended policy", got, want)
						}
					} else if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
						t.Fatal(req.Method, req.URL)
					}
					if mode == "denied" {
						return labelsResponse(req, 403, `{"message":"denied"}`), nil
					}
					body := approvalsIntendedPayload(test.count, both)
					body["id"], body["source_type"], body["source"] = 7, "Repository", "owner/current"
					if mode == "11" || mode == "negative" || mode == "retained" || mode == "null" {
						value := any(float64(11))
						if mode == "negative" {
							value = float64(-1)
						}
						if mode == "retained" {
							value = float64(0)
						}
						if mode == "null" {
							value = nil
						}
						retained := approvalsIntendedPayload(0, both)["rules"].([]any)
						for _, r := range retained {
							rule := r.(map[string]any)
							if rule["type"] == "pull_request" {
								rule["parameters"].(map[string]any)["required_approving_review_count"] = value
							}
						}
						body["rules"] = retained
					}
					if mode == "wrongID" {
						body["id"] = 8
					}
					if mode == "foreign" {
						body["source"] = "owner/other"
					}
					encoded, _ := json.Marshal(body)
					return labelsResponse(req, 200, string(encoded)), nil
				}))
				id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, rulesetApprovalsChange(t, test.count, both))
				want := []string{"PUT", "GET"}
				if test.ack != "canonical" {
					want = []string{"PUT"}
				}
				if (err != nil) != test.wantErr || id != 7 || !reflect.DeepEqual(operations, want) {
					t.Fatal(id, err, operations, want)
				}
				if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
					t.Fatal("lost uncertainty", err)
				}
			})
		}
	}
}
