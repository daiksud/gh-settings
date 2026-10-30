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

func TestRulesetBypassFreshWarningsNeverConfirmRawDrift(t *testing.T) {
	type event struct {
		name, kind, raw string
		actorID         int64
	}
	for _, mode := range []string{"canonical", "unknown", "renamed", "malformed", "foreign", "wrong ID", "unrelated", "denied"} {
		t.Run(mode, func(t *testing.T) {
			canonical, _ := json.Marshal(bypassReadBody("always", false))
			before, err := decodeRulesetPolicyFromTest(canonical)
			if err != nil {
				t.Fatal(err)
			}
			after, err := decodeRulesetPolicyFromTest(canonical)
			if err != nil {
				t.Fatal(err)
			}
			id, actorID := int64(7), int64(9)
			before.ID, after.ID = &id, &id
			change, err := settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, settings.RulesetNormalizations{BypassModes: []settings.RulesetBypassModeNormalization{{ActorType: "Team", ActorID: &actorID, Normalization: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: "BEFORE_MODE", Fallback: "always"}}}})
			if err != nil {
				t.Fatal(err)
			}
			var got []event
			var operations []string
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				operations = append(operations, req.Method)
				if req.URL.Path != "/repos/owner/current/rulesets/7" {
					t.Fatal(req.URL)
				}
				if req.Method == "PUT" {
					return labelsResponse(req, 200, string(canonical)), nil
				}
				if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatal(req.Method, req.URL)
				}
				if mode == "denied" {
					return labelsResponse(req, 403, `{"message":"denied"}`), nil
				}
				body := bypassReadBody("FRESH_MODE", false)
				switch mode {
				case "canonical":
					body = bypassReadBody("always", false)
				case "renamed":
					body["name"] = "fresh bypass"
				case "malformed":
					body["bypass_actors"].([]any)[1].(map[string]any)["bypass_mode"] = nil
				case "foreign":
					body["source"] = "owner/other"
				case "wrong ID":
					body["id"] = 8
				case "unrelated":
					body["future"] = true
				}
				encoded, _ := json.Marshal(body)
				return labelsResponse(req, 200, string(encoded)), nil
			}))
			client.RulesetBypassModeNormalizationWarning = func(id int64, name string, n settings.RulesetBypassModeNormalization) {
				if id != 7 || n.Normalization.Path != settings.RulesetBypassModePath || n.Normalization.Fallback != "always" {
					t.Fatal(id, name, n)
				}
				e := event{name: name, kind: n.ActorType, raw: n.Normalization.RawValue}
				if n.ActorID != nil {
					e.actorID = *n.ActorID
					*n.ActorID = 99
				}
				got = append(got, e)
			}
			gotID, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
			want := []event{{name: "bypass", kind: "Team", raw: "BEFORE_MODE", actorID: 9}}
			if mode == "unknown" || mode == "renamed" {
				name := "bypass"
				if mode == "renamed" {
					name = "fresh bypass"
				}
				want = append(want, event{name: name, kind: "DeployKey", raw: "KEY_MODE"}, event{name: name, kind: "Team", raw: "FRESH_MODE", actorID: 9})
			}
			if gotID != 7 || (err != nil) != (mode != "canonical") || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(operations, []string{"PUT", "GET"}) {
				t.Fatal(gotID, got, want, operations, err)
			}
			if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
				t.Fatal("uncertain effects lost", err)
			}
		})
	}
}
