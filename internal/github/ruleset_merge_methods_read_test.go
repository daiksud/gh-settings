package github

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func mergeMethodsReadBody(methods any, companions bool) map[string]any {
	body := approvalsIntendedPayload(0, companions)
	body["id"], body["source_type"], body["source"] = 7, "Repository", "owner/current"
	for _, r := range body["rules"].([]any) {
		rule := r.(map[string]any)
		params := rule["parameters"].(map[string]any)
		if rule["type"] == "pull_request" {
			params["allowed_merge_methods"] = methods
			if companions {
				params["required_approving_review_count"] = 11
			}
		}
		if rule["type"] == "code_quality" {
			params["severity"] = "FUTURE_SEVERITY"
		}
	}
	if companions {
		body["enforcement"] = "evaluate"
		body["bypass_actors"] = []any{map[string]any{"actor_type": "Team", "actor_id": 9, "bypass_mode": "FUTURE_MODE"}}
	}
	return body
}

func TestMergeMethodsCompleteDecoderPreservesAllEvidenceAndStrictVerification(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, methods := range [][]string{{"squash"}, {"rebase", "merge", "squash"}, {"squash", "future"}, {""}, {" "}} {
			body := mergeMethodsReadBody(methods, companions)
			encoded, _ := json.Marshal(body)
			var raw map[string]json.RawMessage
			json.Unmarshal(encoded, &raw)
			original := map[string]json.RawMessage{}
			for k, v := range raw {
				original[k] = bytes.Clone(v)
			}
			policy, records, err := decodeObservedRulesetPolicy(raw)
			normalized := slices.Contains(methods, "future") || slices.Contains(methods, "") || slices.Contains(methods, " ")
			if err != nil || records.MergeMethods.IsZero() == normalized || !reflect.DeepEqual(raw, original) {
				t.Fatal(policy, records, err)
			}
			if normalized && (!slices.Equal(records.MergeMethods.RawValue, methods) || !slices.Equal(records.MergeMethods.Fallback, []string{"merge", "squash", "rebase"})) {
				t.Fatal(records)
			}
			if companions && (records.Enforcement.RawValue != "evaluate" || records.RequiredApprovals.RawValue != 11 || records.QualitySeverity.RawValue != "FUTURE_SEVERITY" || len(records.BypassModes) != 1 || records.BypassModes[0].Normalization.RawValue != "FUTURE_MODE") {
				t.Fatal("companion evidence lost", records)
			}
			state := settings.RulesetState{ID: 7, Policy: policy, EnforcementNormalization: records.Enforcement, RequiredApprovalsNormalization: records.RequiredApprovals, MergeMethodsNormalization: records.MergeMethods, QualitySeverityNormalization: records.QualitySeverity, BypassModeNormalizations: records.BypassModes}
			observed, err := settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
			if err != nil {
				t.Fatal(err)
			}
			values, _ := observed.Values()
			got, err := values[0].RawMergeMethods()
			expectedRaw := methods
			if !normalized && len(methods) == 3 {
				expectedRaw = []string{"merge", "rebase", "squash"}
			}
			if err != nil || !slices.Equal(got, expectedRaw) {
				t.Fatal("raw list order lost", got, methods, err)
			}
			if (verifyReviewRulesetResponse(policy, raw) != nil) != (normalized || companions) {
				t.Fatal("strict verification used projected values")
			}
		}
	}
}

func TestMergeMethodsHTTPReadIsCompleteAndSilent(t *testing.T) {
	for _, mode := range []string{"normalized", "absent", "null", "null member", "empty", "duplicate", "missing", "unknown member", "later denied", "pagination denied"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatal(req.Method, req.URL)
				}
				if req.URL.Path == "/repos/owner/current/rulesets" {
					if req.URL.Query().Get("page") == "2" {
						return labelsResponse(req, 403, `{"message":"denied"}`), nil
					}
					response := labelsResponse(req, 200, `[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":9,"source_type":"Repository","source":"owner/current"}]`)
					if mode == "pagination denied" {
						items := []any{map[string]any{"id": 7, "source_type": "Repository", "source": "owner/current"}}
						for i := 1; i < 100; i++ {
							items = append(items, map[string]any{"source_type": "Organization", "source": "owner"})
						}
						encoded, _ := json.Marshal(items)
						response = labelsResponse(req, 200, string(encoded))
					}
					return response, nil
				}
				body := mergeMethodsReadBody([]string{"squash", "future"}, true)
				if strings.HasSuffix(req.URL.Path, "/9") {
					body["id"], body["name"] = 9, "later"
					if mode == "later denied" {
						return labelsResponse(req, 403, `{"message":"denied"}`), nil
					}
					params := body["rules"].([]any)[1].(map[string]any)["parameters"].(map[string]any)
					switch mode {
					case "absent":
						body["rules"] = []any{}
					case "null":
						params["allowed_merge_methods"] = nil
					case "null member":
						params["allowed_merge_methods"] = []any{"future", nil}
					case "empty":
						params["allowed_merge_methods"] = []any{}
					case "duplicate":
						params["allowed_merge_methods"] = []any{"future", "future"}
					case "missing":
						delete(params, "allowed_merge_methods")
					case "unknown member":
						params["unknown"] = true
					}
				} else if !strings.HasSuffix(req.URL.Path, "/7") {
					t.Fatal(req.URL)
				}
				encoded, _ := json.Marshal(body)
				return labelsResponse(req, 200, string(encoded)), nil
			}))
			client.RulesetMergeMethodsNormalizationWarning = func(int64, string, settings.RulesetMergeMethodsNormalization) { t.Fatal("pure reader warned") }
			observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
			success := mode == "normalized" || mode == "absent"
			if (err == nil) != success {
				t.Fatal(mode, err)
			}
			if !success {
				wantCalls := 3
				if mode == "pagination denied" {
					wantCalls = 2
				}
				if calls != wantCalls {
					t.Fatal("failure ordering", calls, wantCalls)
				}
				if !reflect.DeepEqual(observed, settings.RulesetObservation{}) {
					t.Fatal("partial observation escaped", observed)
				}
				return
			}
			values, err := observed.Values()
			if err != nil || len(values) != 2 || calls != 3 || values[0].ID != 7 || values[1].ID != 9 {
				t.Fatal(values, err, calls)
			}
			for i, v := range values {
				raw, err := v.RawMergeMethods()
				if err != nil {
					t.Fatal(err)
				}
				if mode == "absent" && i == 1 {
					if raw != nil || !v.MergeMethodsNormalization.IsZero() {
						t.Fatal("absence invented list", v)
					}
				} else if !slices.Equal(raw, []string{"squash", "future"}) {
					t.Fatal(raw)
				}
			}
		})
	}
}

func TestMergeMethodsIncompletePolicyPublishesNoProjection(t *testing.T) {
	for _, field := range []string{"conditions", "bypass_actors", "name", "rules", "future"} {
		body := mergeMethodsReadBody([]string{"future"}, true)
		if field == "future" {
			body[field] = true
		} else {
			delete(body, field)
		}
		encoded, _ := json.Marshal(body)
		var raw map[string]json.RawMessage
		json.Unmarshal(encoded, &raw)
		policy, records, err := decodeObservedRulesetPolicy(raw)
		if err == nil || !reflect.DeepEqual(policy, settings.RulesetDeclaration{}) || !records.IsZero() {
			t.Fatal(field, policy, records, err)
		}
	}
}
