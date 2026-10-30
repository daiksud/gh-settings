package github

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestUnifiedRulesetObservationDecoderOwnsTypedApprovalProvenance(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, count := range []int{0, 1, 10, -1, 11, 100} {
			body := approvalsIntendedPayload(count, both)
			if both {
				body["enforcement"] = "evaluate"
				for _, r := range body["rules"].([]any) {
					rule := r.(map[string]any)
					if rule["type"] == "code_quality" {
						rule["parameters"].(map[string]any)["severity"] = "FUTURE_SEVERITY"
					}
				}
			}
			// -1 is a real acquired integer here, not the desired-removal fixture selector.
			if count == -1 {
				body["rules"] = approvalsIntendedPayload(0, both)["rules"]
				for _, r := range body["rules"].([]any) {
					rule := r.(map[string]any)
					if rule["type"] == "pull_request" {
						rule["parameters"].(map[string]any)["required_approving_review_count"] = -1
					}
					if rule["type"] == "code_quality" {
						rule["parameters"].(map[string]any)["severity"] = "FUTURE_SEVERITY"
					}
				}
			}
			encoded, _ := json.Marshal(body)
			var raw map[string]json.RawMessage
			json.Unmarshal(encoded, &raw)
			original := map[string]json.RawMessage{}
			for k, v := range raw {
				original[k] = bytes.Clone(v)
			}
			policy, records, err := decodeObservedRulesetPolicy(raw)
			want := settings.RulesetNormalizations{}
			effective := count
			if count < 0 || count > 10 {
				effective = 0
				want.RequiredApprovals = settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: count, Fallback: 0}
			}
			if both {
				want.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
				want.QualitySeverity = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "FUTURE_SEVERITY", Fallback: "errors"}
			}
			if err != nil || records != want || !reflect.DeepEqual(raw, original) || policy.Rules == nil {
				t.Fatal("aggregate or immutable source lost", policy, records, want, err)
			}
			found := false
			for _, r := range *policy.Rules {
				if r.Type == "pull_request" {
					found = true
					if *r.Parameters.RequiredApprovingReviewCount != effective {
						t.Fatal("wrong effective integer", r)
					}
				}
			}
			if !found {
				t.Fatal("PR rule lost")
			}
			verifyErr := verifyReviewRulesetResponse(policy, raw)
			if (verifyErr != nil) != (want != (settings.RulesetNormalizations{})) {
				t.Fatal("verification consumed projection", verifyErr)
			}
		}
	}
}

func TestRulesetRequiredApprovalsHTTPReaderOwnsThreeRecordsAndAbsenceSilently(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, count := range []int{0, 10, -1, 11, 100, -99} {
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatal(req.Method, req.URL)
				}
				if req.URL.Path == "/repos/owner/current/rulesets" {
					return labelsResponse(req, 200, `[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":9,"source_type":"Repository","source":"owner/current"}]`), nil
				}
				value := count
				id, name := 7, "approvals"
				if strings.HasSuffix(req.URL.Path, "/9") {
					value, id, name = 12, 9, "other"
				} else if req.URL.Path != "/repos/owner/current/rulesets/7" {
					t.Fatal(req.URL)
				}
				body := approvalsIntendedPayload(0, both)
				body["id"], body["name"], body["source_type"], body["source"] = id, name, "Repository", "owner/current"
				rules := []any{}
				for _, r := range body["rules"].([]any) {
					rule := r.(map[string]any)
					if rule["type"] == "pull_request" {
						if value == -99 {
							continue
						}
						rule["parameters"].(map[string]any)["required_approving_review_count"] = value
					}
					if rule["type"] == "code_quality" {
						rule["parameters"].(map[string]any)["severity"] = "FUTURE_SEVERITY"
					}
					rules = append(rules, rule)
				}
				body["rules"] = rules
				if both {
					body["enforcement"] = "evaluate"
				}
				encoded, _ := json.Marshal(body)
				return labelsResponse(req, 200, string(encoded)), nil
			}))
			client.RulesetNormalizationWarning = func(int64, string, settings.ObservationNormalization) { t.Fatal("pureReaderwarned") }
			client.RulesetRequiredApprovalsNormalizationWarning = func(int64, string, settings.RulesetRequiredApprovalsNormalization) {
				t.Fatal("pureReaderintegerwarned")
			}
			observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
			if err != nil {
				t.Fatal(count, both, err)
			}
			values, err := observed.Values()
			if err != nil || len(values) != 2 || calls != 3 || values[0].ID != 7 || values[1].ID != 9 {
				t.Fatal(values, calls, err)
			}
			for i, want := range []int{count, 12} {
				got, err := values[i].RawRequiredApprovals()
				if err != nil || (got == nil) != (want == -99) || got != nil && *got != want {
					t.Fatal("IDbound integer lost", values, got, err)
				}
				if both {
					if enf, err := values[i].RawEnforcement(); err != nil || enf != "evaluate" {
						t.Fatal(enf, err)
					}
					if q, err := values[i].RawQualitySeverity(); err != nil || q == nil || *q != "FUTURE_SEVERITY" {
						t.Fatal(q, err)
					}
				}
			}
			if count == -99 && values[0].RequiredApprovalsNormalization != (settings.RulesetRequiredApprovalsNormalization{}) {
				t.Fatal("absence invented metadata")
			}
		}
	}
}

func TestRulesetRequiredApprovalsHTTPReaderRejectsLaterMalformedPolicyWithoutPartialEvidence(t *testing.T) {
	var base map[string]json.RawMessage
	json.Unmarshal([]byte(approvalsRulesetJSON), &base)
	var invalidRules []json.RawMessage
	for _, token := range []string{`null`, `"11"`, `true`, `11.5`, `1e100`, `9223372036854775808`} {
		invalidRules = append(invalidRules, json.RawMessage(strings.Replace(string(base["rules"]), `"required_approving_review_count":0`, `"required_approving_review_count":`+token, 1)))
	}
	invalidRules = append(invalidRules, json.RawMessage(`null`), json.RawMessage(`{}`), json.RawMessage(`[{"type":"pull_request"}]`), json.RawMessage(`[{"type":"pull_request","parameters":null}]`), json.RawMessage(strings.Replace(string(base["rules"]), `"required_approving_review_count":0,`, "", 1)), json.RawMessage(strings.Replace(string(base["rules"]), `"required_approving_review_count":0`, `"required_approving_review_count":11,"future":true`, 1)))
	for _, rules := range invalidRules {
		var raw map[string]json.RawMessage
		json.Unmarshal([]byte(approvalsRulesetJSON), &raw)
		raw["enforcement"], raw["rules"] = json.RawMessage(`"evaluate"`), rules
		original := map[string]json.RawMessage{}
		for k, v := range raw {
			original[k] = bytes.Clone(v)
		}
		policy, records, err := decodeObservedRulesetPolicy(raw)
		if err == nil || !reflect.DeepEqual(policy, settings.RulesetDeclaration{}) || records != (settings.RulesetNormalizations{}) || !reflect.DeepEqual(raw, original) {
			t.Fatal("partialdecoderescape", string(rules), policy, records, err)
		}
		calls := 0
		client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != "GET" {
				t.Fatal(req.Method)
			}
			if req.URL.Path == "/repos/owner/current/rulesets" {
				return labelsResponse(req, 200, `[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":9,"source_type":"Repository","source":"owner/current"}]`), nil
			}
			id := 7
			if strings.HasSuffix(req.URL.Path, "/9") {
				id = 9
			}
			var response map[string]json.RawMessage
			json.Unmarshal([]byte(approvalsRulesetJSON), &response)
			response["id"], response["enforcement"] = json.RawMessage(strconv.Itoa(id)), json.RawMessage(`"evaluate"`)
			if id == 9 {
				response["name"], response["rules"] = json.RawMessage(`"later"`), rules
			} else {
				response["rules"] = json.RawMessage(strings.Replace(string(response["rules"]), `"required_approving_review_count":0`, `"required_approving_review_count":11`, 1))
			}
			body, _ := json.Marshal(response)
			return labelsResponse(req, 200, string(body)), nil
		}))
		client.RulesetNormalizationWarning = func(int64, string, settings.ObservationNormalization) { t.Fatal("failedReaderwarned") }
		client.RulesetRequiredApprovalsNormalizationWarning = func(int64, string, settings.RulesetRequiredApprovalsNormalization) {
			t.Fatal("failedReaderintegerwarned")
		}
		observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
		if err == nil || calls != 3 || !reflect.DeepEqual(observed, settings.RulesetObservation{}) {
			t.Fatal("partialcollectionescape", observed, calls, err)
		}
	}
}
