package github

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestRulesetEnforcementWarningsRequireSelectedCompleteOwnedEvidence(t *testing.T) {
	type diagnostic struct {
		id     int64
		name   string
		record settings.ObservationNormalization
	}
	for _, test := range []struct {
		name, readback string
		sink           bool
		fresh          bool
	}{
		{"canonical", basicRulesetJSON, true, false},
		{"retained evaluate", strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"evaluate"`, 1), true, true},
		{"fresh unknown", strings.Replace(strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"OTHER_MODE"`, 1), `"name":"basic"`, `"name":"fresh name"`, 1), true, true},
		{"malformed", strings.Replace(strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"evaluate"`, 1), `"rules":[{"type":"deletion"},{"type":"non_fast_forward"}]`, `"rules":null`, 1), true, false},
		{"foreign", strings.Replace(strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"evaluate"`, 1), `"source":"Owner/Current"`, `"source":"owner/other"`, 1), true, false},
		{"inherited", strings.Replace(strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"evaluate"`, 1), `"source_type":"Repository"`, `"source_type":"Organization"`, 1), true, false},
		{"wrong ID", strings.Replace(strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"evaluate"`, 1), `"id":7`, `"id":8`, 1), true, false},
		{"nil sink", strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"evaluate"`, 1), false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var got []diagnostic
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == "PUT" {
					if test.sink && len(got) != 1 {
						t.Fatal("selected prewrite evidence not warned", got)
					}
					return labelsResponse(req, 200, basicRulesetJSON), nil
				}
				if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				return labelsResponse(req, 200, test.readback), nil
			}))
			if test.sink {
				client.RulesetNormalizationWarning = func(id int64, name string, record settings.ObservationNormalization) {
					got = append(got, diagnostic{id, name, record})
				}
			}
			_, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, rulesetEnforcementChange(t, "active", true))
			if (err == nil) != (test.name == "canonical") {
				t.Fatal("diagnostic projection altered verification", err)
			}
			var want []diagnostic
			if test.sink {
				want = append(want, diagnostic{7, "basic", settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}})
				if test.fresh {
					raw, name := "evaluate", "basic"
					if test.name == "fresh unknown" {
						raw, name = "OTHER_MODE", "fresh name"
					}
					want = append(want, diagnostic{7, name, settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: raw, Fallback: "disabled"}})
				}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatal("incorrect scope/fresh context", got, want)
			}
		})
	}
	client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
		return labelsResponse(req, 200, basicRulesetJSON), nil
	}))
	client.RulesetNormalizationWarning = func(int64, string, settings.ObservationNormalization) { t.Fatal("canonical update warned") }
	if _, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, rulesetEnforcementChange(t, "active", false)); err != nil {
		t.Fatal(err)
	}
}
