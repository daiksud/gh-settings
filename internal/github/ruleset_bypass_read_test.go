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

func bypassReadBody(mode string, companions bool) map[string]any {
	var body map[string]any
	json.Unmarshal([]byte(basicRulesetJSON), &body)
	if companions {
		body = approvalsIntendedPayload(11, true)
		body["enforcement"] = "evaluate"
		for _, r := range body["rules"].([]any) {
			rule := r.(map[string]any)
			if rule["type"] == "code_quality" {
				rule["parameters"].(map[string]any)["severity"] = "FUTURE_SEVERITY"
			}
		}
	}
	keyMode := "KEY_MODE"
	if mode == "always" || mode == "exempt" || mode == "pull_request" {
		keyMode = "always"
	}
	body["id"], body["name"], body["source_type"], body["source"] = 7, "bypass", "Repository", "owner/current"
	body["bypass_actors"] = []any{
		map[string]any{"actor_type": "Team", "actor_id": 9, "bypass_mode": mode},
		map[string]any{"actor_type": "User", "actor_id": 9, "bypass_mode": "exempt"},
		map[string]any{"actor_type": "DeployKey", "actor_id": nil, "bypass_mode": keyMode},
	}
	return body
}

func TestRulesetObservedBypassDecoderRetainsExactEvidenceAndStrictVerification(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, mode := range []string{"always", "exempt", "pull_request", "FUTURE_MODE", "", " \t", "for_pull_requests_only"} {
			encoded, _ := json.Marshal(bypassReadBody(mode, companions))
			var raw map[string]json.RawMessage
			json.Unmarshal(encoded, &raw)
			original := map[string]json.RawMessage{}
			for k, v := range raw {
				original[k] = bytes.Clone(v)
			}
			policy, records, err := decodeObservedRulesetPolicy(raw)
			want := settings.RulesetNormalizations{}
			unknown := mode != "always" && mode != "exempt" && mode != "pull_request"
			id := int64(9)
			if unknown {
				want.BypassModes = []settings.RulesetBypassModeNormalization{
					{ActorType: "Team", ActorID: &id, Normalization: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: mode, Fallback: "always"}},
					{ActorType: "DeployKey", Normalization: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: "KEY_MODE", Fallback: "always"}},
				}
			}
			if companions {
				want.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
				want.QualitySeverity = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "FUTURE_SEVERITY", Fallback: "errors"}
				want.RequiredApprovals = settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: 11, Fallback: 0}
			}
			if err != nil || !reflect.DeepEqual(records, want) || !reflect.DeepEqual(raw, original) {
				t.Fatal(policy, records, want, err)
			}
			state := settings.RulesetState{ID: 7, Policy: policy, EnforcementNormalization: records.Enforcement, QualitySeverityNormalization: records.QualitySeverity, RequiredApprovalsNormalization: records.RequiredApprovals, BypassModeNormalizations: records.BypassModes}
			actors, err := state.RawBypassActors()
			if err != nil || len(actors) != 3 {
				t.Fatal(actors, err)
			}
			for _, actor := range actors {
				if actor.ActorType == "Team" && (*actor.ActorID != 9 || *actor.BypassMode != mode) {
					t.Fatal("raw Team evidence lost", actor)
				}
				if actor.ActorType == "User" && (*actor.ActorID != 9 || *actor.BypassMode != "exempt") {
					t.Fatal("canonical companion changed", actor)
				}
			}
			if err := verifyReviewRulesetResponse(policy, raw); (err != nil) != (unknown || companions) {
				t.Fatal("strict verification consumed projection", err)
			}
			if unknown {
				*records.BypassModes[0].ActorID = 99
				for _, actor := range *policy.BypassActors {
					if actor.ActorType == "Team" && *actor.ActorID != 9 {
						t.Fatal("record aliases policy")
					}
				}
			}
		}
	}
}

func TestRulesetBypassHTTPReaderIsSilentAndRejectsPartialAcquisition(t *testing.T) {
	for _, invalid := range []string{"", "missing mode", "null mode", "wrong type", "unsupported actor", "extra field", "invalid applicability", "denied"} {
		t.Run(invalid, func(t *testing.T) {
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatal(req.Method, req.URL)
				}
				if req.URL.Path == "/repos/owner/current/rulesets" {
					return labelsResponse(req, 200, `[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":9,"source_type":"Repository","source":"owner/current"}]`), nil
				}
				if req.URL.Path != "/repos/owner/current/rulesets/7" && req.URL.Path != "/repos/owner/current/rulesets/9" {
					t.Fatal(req.URL)
				}
				body := bypassReadBody("FUTURE_MODE", true)
				if strings.HasSuffix(req.URL.Path, "/9") {
					body = bypassReadBody("always", false)
					body["id"], body["name"] = 9, "other"
					actor := body["bypass_actors"].([]any)[0].(map[string]any)
					switch invalid {
					case "missing mode":
						delete(actor, "bypass_mode")
					case "null mode":
						actor["bypass_mode"] = nil
					case "wrong type":
						actor["bypass_mode"] = true
					case "unsupported actor":
						actor["actor_type"] = "FutureActor"
					case "extra field":
						body["future"] = true
					case "invalid applicability":
						body["bypass_actors"].([]any)[2].(map[string]any)["bypass_mode"] = "pull_request"
					case "denied":
						return labelsResponse(req, 403, `{"message":"denied"}`), nil
					}
				}
				data, _ := json.Marshal(body)
				return labelsResponse(req, 200, string(data)), nil
			}))
			client.RulesetBypassModeNormalizationWarning = func(int64, string, settings.RulesetBypassModeNormalization) { t.Fatal("pure reader warned") }
			client.RulesetNormalizationWarning = func(int64, string, settings.ObservationNormalization) { t.Fatal("pure reader scalar warned") }
			client.RulesetRequiredApprovalsNormalizationWarning = func(int64, string, settings.RulesetRequiredApprovalsNormalization) {
				t.Fatal("pure reader integer warned")
			}
			observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
			values, valueErr := observed.Values()
			if invalid != "" {
				if err == nil || valueErr == nil || len(values) != 0 {
					t.Fatal("partial acquisition escaped", values, err, valueErr)
				}
			} else if err != nil || valueErr != nil || len(values) != 2 || len(values[0].BypassModeNormalizations) != 2 || len(values[1].BypassModeNormalizations) != 0 || values[0].EnforcementNormalization.RawValue != "evaluate" || values[0].RequiredApprovalsNormalization.RawValue != 11 {
				t.Fatal("complete records lost", values, err, valueErr)
			}
			if calls != 3 {
				t.Fatal("read stopped before later validation", calls)
			}
		})
	}
}
