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

func TestRulesetEnforcementHTTPReadRetainsIDBoundRawEvidence(t *testing.T) {
	for _, token := range []string{"active", "disabled", "evaluate", "FUTURE_MODE", "", " \t"} {
		t.Run(token, func(t *testing.T) {
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatal(req.Method, req.URL)
				}
				if req.URL.Path == "/repos/owner/current/rulesets" {
					return labelsResponse(req, 200, `[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":8,"source_type":"Repository","source":"owner/current"}]`), nil
				}
				if req.URL.Path != "/repos/owner/current/rulesets/7" && req.URL.Path != "/repos/owner/current/rulesets/8" {
					t.Fatal(req.URL)
				}
				var raw map[string]json.RawMessage
				json.Unmarshal([]byte(basicRulesetJSON), &raw)
				if strings.HasSuffix(req.URL.Path, "/8") {
					raw["id"] = json.RawMessage(`8`)
					raw["name"] = json.RawMessage(`"other"`)
					raw["enforcement"] = json.RawMessage(`"OTHER_MODE"`)
				} else {
					raw["enforcement"], _ = json.Marshal(token)
				}
				body, _ := json.Marshal(raw)
				return labelsResponse(req, 200, string(body)), nil
			}))
			client.RulesetNormalizationWarning = func(int64, string, settings.ObservationNormalization) { t.Fatal("pure reader warned") }
			observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
			if err != nil {
				t.Fatal(err)
			}
			values, err := observed.Values()
			if err != nil || len(values) != 2 || values[0].ID != 7 || values[1].ID != 8 || calls != 3 {
				t.Fatal(values, calls, err)
			}
			wantPolicy, wantRecord := token, settings.ObservationNormalization{}
			if token != "active" && token != "disabled" {
				wantPolicy = "disabled"
				wantRecord = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: token, Fallback: "disabled"}
			}
			if *values[0].Policy.Enforcement != wantPolicy || values[0].EnforcementNormalization != wantRecord {
				t.Fatal(values)
			}
			for i, want := range []string{token, "OTHER_MODE"} {
				if raw, err := values[i].RawEnforcement(); err != nil || raw != want {
					t.Fatal("cross-ID provenance lost", values, raw, err)
				}
			}
		})
	}
}

func TestRulesetEnforcementHTTPReadErrorsDiscardWholeCollection(t *testing.T) {
	for _, mutate := range []func(map[string]json.RawMessage){
		func(raw map[string]json.RawMessage) { delete(raw, "enforcement") },
		func(raw map[string]json.RawMessage) { raw["enforcement"] = json.RawMessage(`null`) },
		func(raw map[string]json.RawMessage) { raw["enforcement"] = json.RawMessage(`false`) },
		func(raw map[string]json.RawMessage) {
			raw["conditions"] = json.RawMessage(`{"ref_name":{"include":[],"exclude":null}}`)
		},
		func(raw map[string]json.RawMessage) { raw["rules"] = json.RawMessage(`null`) },
		func(raw map[string]json.RawMessage) { raw["source"] = json.RawMessage(`"owner/other"`) },
		func(raw map[string]json.RawMessage) { raw["id"] = json.RawMessage(`99`) },
	} {
		calls := 0
		client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if req.Method != "GET" {
				t.Fatal(req.Method)
			}
			if req.URL.Path == "/repos/owner/current/rulesets" {
				return labelsResponse(req, 200, `[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":8,"source_type":"Repository","source":"owner/current"}]`), nil
			}
			var raw map[string]json.RawMessage
			json.Unmarshal([]byte(basicRulesetJSON), &raw)
			raw["enforcement"] = json.RawMessage(`"evaluate"`)
			if strings.HasSuffix(req.URL.Path, "/8") {
				raw["id"] = json.RawMessage(`8`)
				raw["name"] = json.RawMessage(`"later"`)
				mutate(raw)
			}
			body, _ := json.Marshal(raw)
			return labelsResponse(req, 200, string(body)), nil
		}))
		client.RulesetNormalizationWarning = func(int64, string, settings.ObservationNormalization) { t.Fatal("failed reader warned") }
		observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
		if err == nil || !reflect.DeepEqual(observed, settings.RulesetObservation{}) || calls != 3 {
			t.Fatal("partial acquisition escaped", observed, calls, err)
		}
	}
}
