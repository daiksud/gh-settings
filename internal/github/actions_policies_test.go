package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

const eventPolicyJSON = `{"id":7,"name":"events","source_type":"Repository","source":"owner/current","target":"actions","enforcement":"active","rules":[{"type":"restrict_action_events","parameters":{"allowed_events":["push","pull_request"]}}]}`

var eventPolicyRepo = repository.Repository{Owner: "owner", Name: "current"}

func eventPolicyDecode(t *testing.T, body string) (settings.ActionsEventPolicyState, error) {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatal(err)
	}
	return decodeActionsEventPolicy(raw, eventPolicyRepo)
}
func TestActionsPoliciesRepresentableConditions(t *testing.T) {
	for _, conditions := range []string{"", "null", "{}", `{"workflow_path":{"include":["~ALL"],"exclude":[]}}`, `{"workflow_path":{"include":[],"exclude":[]}}`} {
		for _, enforcement := range []string{"active", "disabled"} {
			body := strings.Replace(eventPolicyJSON, `"enforcement":"active"`, `"enforcement":"`+enforcement+`"`, 1)
			if conditions != "" {
				body = strings.Replace(body, `"rules":`, `"conditions":`+conditions+`,"rules":`, 1)
			}
			state, err := eventPolicyDecode(t, body)
			if err != nil || state.Policy == nil || state.Policy.Enforcement != enforcement {
				t.Fatalf("%s: %+v %v", body, state, err)
			}
		}
	}
}
func TestActionsPoliciesPreserveUnsupportedProtections(t *testing.T) {
	replacements := [][2]string{
		{`"enforcement":"active"`, `"enforcement":"evaluate"`},
		{`"rules":`, `"future_protection":true,"rules":`},
		{`"type":"restrict_action_events"`, `"type":"restrict_actions_actors"`},
		{`"allowed_events":["push","pull_request"]`, `"allowed_events":["future_event"]`},
		{`"parameters":{`, `"future_rule":true,"parameters":{`},
		{`"allowed_events":`, `"future_parameter":true,"allowed_events":`},
		{`"rules":`, `"conditions":{"workflow_path":{"include":[".github/workflows/one.yml"],"exclude":[]}},"rules":`},
		{`"rules":`, `"conditions":{"repository_name":{"include":["*"]}},"rules":`},
		{`"rules":[`, `"rules":[{"type":"restrict_actions_actors","parameters":{"allowed_actors":[]}},`},
	}
	for _, r := range replacements {
		body := strings.Replace(eventPolicyJSON, r[0], r[1], 1)
		state, err := eventPolicyDecode(t, body)
		if err != nil || state.Policy != nil || state.Unsupported == "" || state.ID != 7 || state.Name != "events" {
			t.Fatalf("%s: %+v %v", body, state, err)
		}
	}
}
func TestActionsPoliciesRejectIncompleteOrWrongOwnership(t *testing.T) {
	for _, r := range [][2]string{
		{`"id":7`, `"id":0`}, {`"source":"owner/current"`, `"source":"other/current"`}, {`"source_type":"Repository"`, `"source_type":"Organization"`}, {`"target":"actions"`, `"target":"branch"`},
		{`"rules":[{"type":"restrict_action_events","parameters":{"allowed_events":["push","pull_request"]}}]`, `"rules":null`},
		{`"enforcement":"active"`, `"enforcement":null`}, {`"parameters":{"allowed_events":["push","pull_request"]}`, `"parameters":null`}, {`"allowed_events":["push","pull_request"]`, `"allowed_events":null`}, {`"allowed_events":["push","pull_request"]`, `"allowed_events":[1]`},
		{`"rules":`, `"conditions":{"workflow_path":{"include":null,"exclude":[]}},"rules":`},
		{`"rules":`, `"conditions":{"workflow_path":{"include":["~ALL"]}},"rules":`},
	} {
		body := strings.Replace(eventPolicyJSON, r[0], r[1], 1)
		if _, err := eventPolicyDecode(t, body); err == nil {
			t.Fatal("accepted", body)
		}
	}
}
func TestReadActionsPoliciesPagination(t *testing.T) {
	calls := 0
	client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" {
			t.Fatal(r.Method)
		}
		if strings.HasSuffix(r.URL.Path, "/policies") {
			if r.URL.Query().Get("has_parents") != "false" || r.URL.Query().Get("per_page") != "100" {
				t.Fatal(r.URL)
			}
			policies := []any{}
			start, end := 1, 100
			if r.URL.Query().Get("page") == "2" {
				start, end = 101, 101
			}
			for id := start; id <= end; id++ {
				var raw map[string]any
				json.Unmarshal([]byte(eventPolicyJSON), &raw)
				raw["id"] = id
				policies = append(policies, raw)
			}
			data, _ := json.Marshal(map[string]any{"total_count": 101, "policies": policies})
			return labelsResponse(r, 200, string(data)), nil
		}
		var id int
		fmt.Sscanf(r.URL.Path, "/repos/owner/current/actions/policies/%d", &id)
		return labelsResponse(r, 200, strings.Replace(eventPolicyJSON, `"id":7`, fmt.Sprintf(`"id":%d`, id), 1)), nil
	}))
	observed, err := client.ReadActionsPolicies(eventPolicyRepo)
	if err != nil {
		t.Fatal(err)
	}
	values, err := observed.Values()
	if err != nil || len(values) != 101 || calls != 103 {
		t.Fatalf("%d %d %v", len(values), calls, err)
	}
}
func TestReadActionsPoliciesFailsClosed(t *testing.T) {
	for _, body := range []string{`null`, `{"total_count":0,"policies":null}`, `{"total_count":1,"policies":[]}`, `{"total_count":0,"policies":[` + eventPolicyJSON + `]}`, `{"total_count":2,"policies":[` + eventPolicyJSON + `,` + eventPolicyJSON + `]}`} {
		client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) { return labelsResponse(r, 200, body), nil }))
		observed, err := client.ReadActionsPolicies(eventPolicyRepo)
		if err == nil {
			t.Fatal("accepted", body)
		}
		if values, _ := observed.Values(); len(values) != 0 {
			t.Fatal("partial", values)
		}
	}
}
func TestWriteActionsEventPolicyExactPayloadAndVerification(t *testing.T) {
	for _, op := range []string{"create", "update"} {
		for _, enforcement := range []string{"active", "disabled"} {
			t.Run(op+enforcement, func(t *testing.T) {
				before := settings.ActionsEventPolicy{ID: 7, Name: "events", Enforcement: "active", Events: []string{"push", "pull_request"}}
				desired := before
				desired.Enforcement = enforcement
				var beforePtr *settings.ActionsEventPolicy
				if op == "update" {
					if enforcement == "active" {
						before.Enforcement = "disabled"
					}
					beforePtr = &before
				} else {
					desired.ID = 0
				}
				change, err := settings.NewActionsEventPolicyChange(beforePtr, desired)
				if err != nil {
					t.Fatal(err)
				}
				calls := []string{}
				written := false
				client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
					calls = append(calls, r.Method)
					if r.Method != "GET" {
						expectedMethod := "POST"
						if op == "update" {
							expectedMethod = "PUT"
						}
						if r.Method != expectedMethod {
							t.Fatal(r.Method)
						}
						data, _ := io.ReadAll(r.Body)
						var payload map[string]any
						json.Unmarshal(data, &payload)
						var expected map[string]any
						json.Unmarshal([]byte(`{"name":"events","enforcement":"`+enforcement+`","rules":[{"type":"restrict_action_events","parameters":{"allowed_events":["pull_request","push"]}}]}`), &expected)
						if !reflect.DeepEqual(payload, expected) {
							t.Fatalf("unexpected payload %s", data)
						}
						written = true
					}
					body := strings.Replace(eventPolicyJSON, `"enforcement":"active"`, `"enforcement":"`+before.Enforcement+`"`, 1)
					if written {
						body = eventPolicyJSON
						body = strings.Replace(body, `"enforcement":"active"`, `"enforcement":"`+enforcement+`"`, 1)
					}
					return labelsResponse(r, 200, body), nil
				}))
				id, err := client.WriteActionsEventPolicy(eventPolicyRepo, change)
				if err != nil || id != 7 {
					t.Fatalf("%d %v", id, err)
				}
				expected := []string{"POST", "GET"}
				if op == "update" {
					expected = []string{"GET", "PUT", "GET"}
				}
				if !reflect.DeepEqual(calls, expected) {
					t.Fatal(calls)
				}
			})
		}
	}
}
func TestWriteActionsEventPolicyRecoveryAndPreflight(t *testing.T) {
	before := settings.ActionsEventPolicy{ID: 7, Name: "events", Enforcement: "active", Events: []string{"push", "pull_request"}}
	for _, scenario := range []string{"readback-error", "readback-mismatch", "ack-mismatch", "uncertain-create", "unsupported-preflight"} {
		t.Run(scenario, func(t *testing.T) {
			desired := before
			desired.ID = 0
			var prev *settings.ActionsEventPolicy
			if scenario == "unsupported-preflight" {
				desired.ID = 7
				desired.Enforcement = "disabled"
				prev = &before
			}
			change, _ := settings.NewActionsEventPolicyChange(prev, desired)
			writes := 0
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				body := eventPolicyJSON
				if r.Method != "GET" {
					writes++
					if scenario == "uncertain-create" {
						return nil, fmt.Errorf("connection lost")
					}
					if scenario == "ack-mismatch" {
						body = strings.Replace(body, `"enforcement":"active"`, `"enforcement":"disabled"`, 1)
					}
				} else {
					switch scenario {
					case "readback-error":
						return labelsResponse(r, 500, `{}`), nil
					case "readback-mismatch", "unsupported-preflight":
						body = strings.Replace(body, `"type":"restrict_action_events"`, `"type":"restrict_actions_actors"`, 1)
					}
				}
				return labelsResponse(r, 200, body), nil
			}))
			id, err := client.WriteActionsEventPolicy(eventPolicyRepo, change)
			if err == nil {
				t.Fatal("accepted")
			}
			expectedID := int64(7)
			if scenario == "uncertain-create" || scenario == "unsupported-preflight" {
				expectedID = 0
			}
			if id != expectedID {
				t.Fatalf("id %d err %v", id, err)
			}
			expectedWrites := 1
			if scenario == "unsupported-preflight" {
				expectedWrites = 0
			}
			if writes != expectedWrites {
				t.Fatal(writes)
			}
		})
	}
}
