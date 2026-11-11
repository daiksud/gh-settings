package github

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func targetedPolicyJSON(include, exclude string) string {
	return strings.Replace(eventPolicyJSON, `"rules":`, `"conditions":{"workflow_path":{"include":`+include+`,"exclude":`+exclude+`}},"rules":`, 1)
}
func TestActionsWorkflowTargetDecode(t *testing.T) {
	for _, lists := range [][2]string{{`[".github/workflows/*.yml"]`, `["**/skip.yml"]`}, {`[]`, `["skip"]`}, {`["~ALL"]`, `[]`}} {
		state, err := eventPolicyDecode(t, targetedPolicyJSON(lists[0], lists[1]))
		if err != nil || state.Policy == nil {
			t.Fatal(state, err)
		}
	}
	for _, lists := range [][2]string{{`[null]`, `[]`}, {`["~ALL","x"]`, `[]`}, {`[]`, `["~ALL"]`}, {`[""]`, `[]`}, {`["x","x"]`, `[]`}, {`null`, `[]`}, {`[]`, `null`}} {
		state, err := eventPolicyDecode(t, targetedPolicyJSON(lists[0], lists[1]))
		if err == nil && state.Policy != nil {
			t.Fatal("accepted malformed targeting", lists)
		}
	}
}
func TestActionsWorkflowTargetWriteTransitions(t *testing.T) {
	for _, scenario := range []string{"create", "include-to-exclude", "preserve", "reset", "drift", "ack-mismatch", "readback-mismatch"} {
		t.Run(scenario, func(t *testing.T) {
			before := settings.ActionsEventPolicy{ID: 7, Name: "events", Enforcement: "active", Events: []string{"push", "pull_request"}, TargetWorkflows: &settings.ActionsWorkflowTarget{Include: []string{"ci*.yml"}, Exclude: []string{}}}
			after := before
			after.Enforcement = "disabled"
			after.TargetWorkflows = &settings.ActionsWorkflowTarget{Include: []string{}, Exclude: []string{"skip*.yml"}}
			previous := &before
			if scenario == "create" {
				previous = nil
				after.ID = 0
			}
			if scenario == "preserve" {
				after.TargetWorkflows = nil
			}
			if scenario == "reset" {
				after.TargetWorkflows = &settings.ActionsWorkflowTarget{Include: []string{"~ALL"}}
			}
			change, err := settings.NewActionsEventPolicyChange(previous, after)
			if err != nil {
				t.Fatal(err)
			}
			writes, reads := 0, 0
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				body := targetedPolicyJSON(`["ci*.yml"]`, `[]`)
				if r.Method == "GET" && writes == 0 {
					if scenario == "drift" {
						body = targetedPolicyJSON(`["other.yml"]`, `[]`)
					}
					return labelsResponse(r, 200, body), nil
				}
				if r.Method != "GET" {
					writes++
					data, _ := io.ReadAll(r.Body)
					var payload map[string]any
					json.Unmarshal(data, &payload)
					if scenario == "preserve" {
						if _, ok := payload["conditions"]; ok {
							t.Fatal("preserve sent conditions", string(data))
						}
					} else {
						expected := `{"workflow_path":{"include":[],"exclude":["skip*.yml"]}}`
						if scenario == "reset" {
							expected = `{"workflow_path":{"include":["~ALL"],"exclude":[]}}`
						}
						var want any
						json.Unmarshal([]byte(expected), &want)
						if !reflect.DeepEqual(payload["conditions"], want) {
							t.Fatal(string(data))
						}
					}
				} else {
					reads++
				}
				if scenario != "preserve" {
					body = targetedPolicyJSON(`["~ALL"]`, `["skip*.yml"]`)
				}
				if scenario == "reset" {
					body = targetedPolicyJSON(`[]`, `[]`)
				}
				if (scenario == "ack-mismatch" && r.Method != "GET") || (scenario == "readback-mismatch" && r.Method == "GET") {
					body = targetedPolicyJSON(`["drift.yml"]`, `[]`)
				}
				body = strings.Replace(body, `"enforcement":"active"`, `"enforcement":"disabled"`, 1)
				return labelsResponse(r, 200, body), nil
			}))
			id, err := client.WriteActionsEventPolicy(eventPolicyRepo, change)
			failure := scenario == "drift" || scenario == "ack-mismatch" || scenario == "readback-mismatch"
			if (err != nil) != failure {
				t.Fatal(id, err)
			}
			if scenario == "drift" {
				if writes != 0 || id != 0 {
					t.Fatal(writes, id)
				}
			} else if writes != 1 || id != 7 {
				t.Fatal(writes, id)
			}
			if !failure && reads != 1 {
				t.Fatal("missing final read", reads)
			}
		})
	}
}
