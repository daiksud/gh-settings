package github

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestUnifiedRulesetObservationDecoderOwnsSimultaneousProvenance(t *testing.T) {
	for _, enforcement := range []string{"active", "evaluate", "FUTURE_MODE"} {
		for _, severity := range []string{"errors", "warnings", "notes", "all", "FUTURE_SEVERITY", "", " \t", "warnings_and_higher"} {
			var raw map[string]json.RawMessage
			json.Unmarshal([]byte(qualityRulesetJSON), &raw)
			raw["enforcement"], _ = json.Marshal(enforcement)
			raw["rules"], _ = json.Marshal([]any{map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": severity}}})
			original := map[string]json.RawMessage{}
			for k, v := range raw {
				original[k] = bytes.Clone(v)
			}
			policy, records, err := decodeObservedRulesetPolicy(raw)
			want := settings.RulesetNormalizations{}
			wantEnforcement, wantSeverity := enforcement, severity
			if enforcement != "active" {
				wantEnforcement = "disabled"
				want.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: enforcement, Fallback: "disabled"}
			}
			if severity != "errors" && severity != "warnings" && severity != "notes" && severity != "all" {
				wantSeverity = "errors"
				want.QualitySeverity = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: severity, Fallback: "errors"}
			}
			if err != nil || policy.Enforcement == nil || *policy.Enforcement != wantEnforcement || policy.Rules == nil || *(*policy.Rules)[0].Parameters.Severity != wantSeverity || !reflect.DeepEqual(records, want) || !reflect.DeepEqual(raw, original) {
				t.Fatal("decoder lost evidence or mutated input", policy, records, want, err)
			}
			verifyErr := verifyReviewRulesetResponse(policy, raw)
			if (verifyErr != nil) != (!want.IsZero()) {
				t.Fatal("strict verification consumed projection", verifyErr)
			}
		}
	}
}

func TestRulesetQualityHTTPReaderAcquiresBothRecordsAndAbsenceSilently(t *testing.T) {
	for _, both := range []bool{false, true} {
		for _, severity := range []string{"errors", "warnings", "notes", "all", "FUTURE_SEVERITY", "", " \t", "warnings_and_higher", "ABSENT"} {
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatal(req.Method, req.URL)
				}
				if req.URL.Path == "/repos/owner/current/rulesets" {
					return labelsResponse(req, 200, `[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":9,"source_type":"Repository","source":"owner/current"}]`), nil
				}
				var raw map[string]any
				json.Unmarshal([]byte(qualityRulesetJSON), &raw)
				value := severity
				if strings.HasSuffix(req.URL.Path, "/9") {
					raw["id"], raw["name"], value = 9, "other", "OTHER_SEVERITY"
				} else if req.URL.Path != "/repos/owner/current/rulesets/7" {
					t.Fatal(req.URL)
				}
				if both {
					raw["enforcement"] = "evaluate"
				}
				raw["rules"] = []any{map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": value}}}
				if value == "ABSENT" {
					raw["rules"] = []any{}
				}
				body, _ := json.Marshal(raw)
				return labelsResponse(req, 200, string(body)), nil
			}))
			client.RulesetNormalizationWarning = func(int64, string, settings.ObservationNormalization) { t.Fatal("pure reader warned") }
			observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
			if err != nil {
				t.Fatal(severity, both, err)
			}
			values, err := observed.Values()
			if err != nil || len(values) != 2 || values[0].ID != 7 || values[1].ID != 9 || calls != 3 {
				t.Fatal(values, calls, err)
			}
			for i, want := range []string{severity, "OTHER_SEVERITY"} {
				got, err := values[i].RawQualitySeverity()
				if err != nil || (got == nil) != (want == "ABSENT") || got != nil && *got != want {
					t.Fatal("ID-bound quality lost", values, got, err)
				}
				wantEnforcement := "active"
				if both {
					wantEnforcement = "evaluate"
				}
				if raw, err := values[i].RawEnforcement(); err != nil || raw != wantEnforcement {
					t.Fatal("enforcement evidence lost", raw, err)
				}
			}
			if severity == "ABSENT" && values[0].QualitySeverityNormalization != (settings.ObservationNormalization{}) {
				t.Fatal("absent rule invented metadata")
			}
		}
	}
}

func TestRulesetQualityHTTPReaderDiscardsLaterIncompletePolicy(t *testing.T) {
	for _, rules := range []string{`null`, `{}`, `[{"type":"code_quality"}]`, `[{"type":"code_quality","parameters":null}]`, `[{"type":"code_quality","parameters":{}}]`, `[{"type":"code_quality","parameters":{"severity":null}}]`, `[{"type":"code_quality","parameters":{"severity":42}}]`, `[{"type":"code_quality","parameters":{"severity":false}}]`, `[{"type":"code_quality","parameters":{"severity":"errors"}},{"type":"code_quality","parameters":{"severity":"future"}}]`, `[{"type":"code_quality","parameters":{"severity":"future","extra":true}}]`} {
		var incomplete map[string]json.RawMessage
		json.Unmarshal([]byte(qualityRulesetJSON), &incomplete)
		incomplete["enforcement"], incomplete["rules"] = json.RawMessage(`"evaluate"`), json.RawMessage(rules)
		original := map[string]json.RawMessage{}
		for key, value := range incomplete {
			original[key] = bytes.Clone(value)
		}
		policy, records, decodeErr := decodeObservedRulesetPolicy(incomplete)
		if decodeErr == nil || !reflect.DeepEqual(policy, settings.RulesetDeclaration{}) || !reflect.DeepEqual(records, settings.RulesetNormalizations{}) || !reflect.DeepEqual(incomplete, original) {
			t.Fatal("invalid decoder returned partial evidence or mutated input", policy, records, decodeErr)
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
			var raw map[string]json.RawMessage
			json.Unmarshal([]byte(qualityRulesetJSON), &raw)
			raw["enforcement"] = json.RawMessage(`"evaluate"`)
			if strings.HasSuffix(req.URL.Path, "/9") {
				raw["id"] = json.RawMessage(`9`)
				raw["name"] = json.RawMessage(`"later"`)
				raw["rules"] = json.RawMessage(rules)
			}
			body, _ := json.Marshal(raw)
			return labelsResponse(req, 200, string(body)), nil
		}))
		client.RulesetNormalizationWarning = func(int64, string, settings.ObservationNormalization) { t.Fatal("failed acquisition warned") }
		observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
		if err == nil || calls != 3 || !reflect.DeepEqual(observed, settings.RulesetObservation{}) {
			t.Fatal("partial collection escaped", observed, calls, err)
		}
	}
}
