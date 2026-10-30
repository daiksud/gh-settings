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

func TestRulesetBypassPrewriteWarningsKeepIdentityOwnershipAndIndependentSinks(t *testing.T) {
	type event struct {
		kind    string
		actorID int64
		record  settings.ObservationNormalization
	}
	for _, mode := range []string{"normalized", "removed", "canonical"} {
		for _, scalar := range []bool{false, true} {
			for sinks := 0; sinks < 4; sinks++ {
				t.Run(fmt.Sprintf("%s/%t/%d", mode, scalar, sinks), func(t *testing.T) {
					actors := []any{
						map[string]any{"actor_type": "Team", "actor_id": 10, "bypass_mode": "always"},
						map[string]any{"actor_type": "DeployKey", "actor_id": nil, "bypass_mode": "always"},
						map[string]any{"actor_type": "Team", "actor_id": 2, "bypass_mode": "always"},
						map[string]any{"actor_type": "User", "actor_id": 2, "bypass_mode": "exempt"},
					}
					var body map[string]any
					json.Unmarshal([]byte(basicRulesetJSON), &body)
					body["bypass_actors"], body["enforcement"] = actors, "disabled"
					encoded, _ := json.Marshal(body)
					before, err := decodeRulesetPolicyFromTest(encoded)
					if err != nil {
						t.Fatal(err)
					}
					id, team2, team10 := int64(7), int64(2), int64(10)
					before.ID = &id
					if mode == "removed" {
						body["bypass_actors"] = actors[1:]
					}
					if mode == "canonical" {
						actors[2].(map[string]any)["bypass_mode"] = "exempt"
					}
					encoded, _ = json.Marshal(body)
					after, err := decodeRulesetPolicyFromTest(encoded)
					if err != nil {
						t.Fatal(err)
					}
					after.ID = &id
					record := func(raw string) settings.ObservationNormalization {
						return settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: raw, Fallback: "always"}
					}
					records := settings.RulesetNormalizations{}
					if mode != "canonical" {
						records.BypassModes = []settings.RulesetBypassModeNormalization{
							{ActorType: "Team", ActorID: &team10, Normalization: record("TEN_MODE")},
							{ActorType: "DeployKey", Normalization: record("KEY_MODE")},
							{ActorType: "Team", ActorID: &team2, Normalization: record("TWO_MODE")},
						}
					}
					if scalar {
						records.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
					}
					var change settings.RulesetChange
					if records.IsZero() {
						change, err = settings.NewRulesetChange("update", 7, &before, &after)
					} else {
						change, err = settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
					}
					if err != nil {
						t.Fatal(err)
					}
					original := change.Normalizations()
					var got, want []event
					if scalar && sinks&1 != 0 {
						want = append(want, event{kind: "scalar", record: records.Enforcement})
					}
					if mode != "canonical" && sinks&2 != 0 {
						want = append(want, event{kind: "DeployKey", record: record("KEY_MODE")}, event{kind: "Team", actorID: 2, record: record("TWO_MODE")}, event{kind: "Team", actorID: 10, record: record("TEN_MODE")})
					}
					var operations []string
					client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
						operations = append(operations, req.Method)
						if req.URL.Path != "/repos/owner/current/rulesets/7" {
							t.Fatal(req.URL)
						}
						if req.Method == "PUT" {
							if !reflect.DeepEqual(got, want) {
								t.Fatal("warnings missing before PUT", got, want)
							}
							var payload map[string]any
							if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
								t.Fatal(err)
							}
							for _, rawActor := range payload["bypass_actors"].([]any) {
								actor := rawActor.(map[string]any)
								if actor["actor_id"] == float64(99) {
									t.Fatal("warning sink mutated write identity", payload)
								}
							}
						} else if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
							t.Fatal(req.Method, req.URL)
						}
						return labelsResponse(req, 200, string(encoded)), nil
					}))
					if sinks&1 != 0 {
						client.RulesetNormalizationWarning = func(id int64, name string, n settings.ObservationNormalization) {
							if id != 7 || name != "basic" {
								t.Fatal(id, name)
							}
							got = append(got, event{kind: "scalar", record: n})
						}
					}
					if sinks&2 != 0 {
						client.RulesetBypassModeNormalizationWarning = func(id int64, name string, n settings.RulesetBypassModeNormalization) {
							if id != 7 || name != "basic" {
								t.Fatal(id, name)
							}
							e := event{kind: n.ActorType, record: n.Normalization}
							if n.ActorID != nil {
								e.actorID = *n.ActorID
								*n.ActorID = 99
							}
							got = append(got, e)
						}
					}
					gotID, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
					wantOperations := []string{"PUT"}
					if mode != "canonical" || scalar {
						wantOperations = append(wantOperations, "GET")
					}
					if err != nil || gotID != 7 || !reflect.DeepEqual(got, want) || !reflect.DeepEqual(change.Normalizations(), original) || !reflect.DeepEqual(operations, wantOperations) {
						t.Fatal("prewrite ownership/order or canonical silence lost", gotID, got, want, operations, err)
					}
				})
			}
		}
	}
}
