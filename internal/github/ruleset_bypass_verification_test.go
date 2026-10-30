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

func TestNormalizedRulesetBypassBasicUpdatesRequireStrictFreshGET(t *testing.T) {
	withActor := func(mode string) string {
		return strings.Replace(basicRulesetJSON, `"bypass_actors":[]`, `"bypass_actors":[{"actor_type":"User","actor_id":9,"bypass_mode":"`+mode+`"}]`, 1)
	}
	canonical, unknown := withActor("always"), withActor("FUTURE_MODE")
	for _, tc := range []struct {
		name     string
		remove   bool
		response string
		readback string
		wantErr  bool
		wantGET  bool
	}{
		{"canonical confirmed", false, canonical, canonical, false, true},
		{"unknown retained", false, canonical, unknown, true, true},
		{"removed actor retained", true, basicRulesetJSON, canonical, true, true},
		{"invalid acknowledgment", false, unknown, canonical, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, err := decodeRulesetPolicyFromTest([]byte(canonical))
			if err != nil {
				t.Fatal(err)
			}
			desired := canonical
			wantActors := []any{map[string]any{"actor_type": "User", "actor_id": float64(9), "bypass_mode": "always"}}
			if tc.remove {
				desired, wantActors = basicRulesetJSON, []any{}
			}
			after, err := decodeRulesetPolicyFromTest([]byte(desired))
			if err != nil {
				t.Fatal(err)
			}
			id, actorID := int64(7), int64(9)
			before.ID, after.ID = &id, &id
			change, err := settings.NewRulesetChangeWithNormalizations("update", id, &before, &after, settings.RulesetNormalizations{
				BypassModes: []settings.RulesetBypassModeNormalization{{ActorType: "User", ActorID: &actorID, Normalization: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: "FUTURE_MODE", Fallback: "always"}}},
			})
			if err != nil {
				t.Fatal(err)
			}
			var operations []string
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				operations = append(operations, req.Method)
				if req.URL.Path != "/repos/owner/current/rulesets/7" {
					t.Fatal("wrong selected target", req.URL)
				}
				if req.Method == "PUT" {
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					want := map[string]any{
						"name": "basic", "target": "branch", "enforcement": "active",
						"conditions":    map[string]any{"ref_name": map[string]any{"include": []any{"~DEFAULT_BRANCH"}, "exclude": []any{}}},
						"bypass_actors": wantActors,
						"rules":         []any{map[string]any{"type": "deletion"}, map[string]any{"type": "non_fast_forward"}},
					}
					if !reflect.DeepEqual(payload, want) {
						t.Fatal("literal whole-object payload changed", payload, want)
					}
					return labelsResponse(req, 200, tc.response), nil
				}
				if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatal("unexpected verification request", req.Method, req.URL)
				}
				return labelsResponse(req, 200, tc.readback), nil
			}))
			gotID, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
			wantOperations := []string{"PUT"}
			if tc.wantGET {
				wantOperations = append(wantOperations, "GET")
			}
			if gotID != 7 || (err != nil) != tc.wantErr || !reflect.DeepEqual(operations, wantOperations) {
				t.Fatal("raw confirmation contract lost", gotID, err, operations, wantOperations)
			}
			if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
				t.Fatal("uncertain effects lost", err)
			}
		})
	}
}
