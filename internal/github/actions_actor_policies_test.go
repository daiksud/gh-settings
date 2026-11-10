package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

const actorPolicyJSON = `{"id":7,"name":"actors","source_type":"Repository","source":"owner/current","target":"actions","enforcement":"disabled","rules":[{"type":"restrict_actions_actors","parameters":{"allowed_actors":[{"id":5,"type":"RepositoryRole"},{"id":123,"type":"User"}]}}]}`

func TestActionsActorsCompleteRead(t *testing.T) {
	for _, body := range []string{actorPolicyJSON, strings.Replace(actorPolicyJSON, `[{"id":5,"type":"RepositoryRole"},{"id":123,"type":"User"}]`, `[]`, 1), strings.Replace(actorPolicyJSON, `"rules":[`, `"rules":[{"type":"restrict_action_events","parameters":{"allowed_events":["push"]}},`, 1)} {
		state, err := eventPolicyDecode(t, body)
		if err != nil || state.Policy == nil || state.Policy.Actors == nil {
			t.Fatalf("%s: %+v %v", body, state, err)
		}
	}
	for _, replacement := range []string{`null`, `[null]`, `[{}]`, `[{"id":5}]`, `[{"type":"User"}]`, `[{"id":null,"type":"User"}]`, `[{"id":1.5,"type":"User"}]`, `[{"id":"5","type":"User"}]`, `[{"id":5,"type":null}]`, `[{"id":5,"type":1}]`} {
		body := strings.Replace(actorPolicyJSON, `[{"id":5,"type":"RepositoryRole"},{"id":123,"type":"User"}]`, replacement, 1)
		if _, err := eventPolicyDecode(t, body); err == nil {
			t.Fatal("accepted incomplete actor", body)
		}
	}
	for _, replacement := range []string{`[{"id":1,"type":"Team"}]`, `[{"id":1,"type":"RepositoryRole"}]`, `[{"id":0,"type":"User"}]`, `[{"id":5,"type":"RepositoryRole","future":true}]`, `[{"id":5,"type":"RepositoryRole"},{"id":5,"type":"RepositoryRole"}]`} {
		body := strings.Replace(actorPolicyJSON, `[{"id":5,"type":"RepositoryRole"},{"id":123,"type":"User"}]`, replacement, 1)
		state, err := eventPolicyDecode(t, body)
		if err != nil || state.Policy != nil || state.Unsupported == "" || state.ID != 7 {
			t.Fatalf("%s: %+v %v", body, state, err)
		}
	}
	duplicate := strings.Replace(actorPolicyJSON, `"rules":[`, `"rules":[{"type":"restrict_actions_actors","parameters":{"allowed_actors":[]}},`, 1)
	state, err := eventPolicyDecode(t, duplicate)
	if err != nil || state.Policy != nil || state.Unsupported == "" {
		t.Fatalf("duplicate %+v %v", state, err)
	}
}

func TestActionsActorsExactWritesAndIndependentReadback(t *testing.T) {
	for _, scenario := range []string{"create", "combined", "empty", "remove", "preserve", "drift", "ack-mismatch", "readback-mismatch"} {
		t.Run(scenario, func(t *testing.T) {
			state, err := eventPolicyDecode(t, actorPolicyJSON)
			if err != nil || state.Policy == nil {
				t.Fatal(err)
			}
			before := *state.Policy
			actors := []settings.ActionsPolicyActor{{Type: "User", ID: 123}, {Type: "RepositoryRole", ID: 4}}
			desired := settings.ActionsEventPolicy{ID: 7, Name: "renamed", Enforcement: "disabled", Actors: &actors}
			expectedRules := `[{"type":"restrict_actions_actors","parameters":{"allowed_actors":[{"id":4,"type":"RepositoryRole"},{"id":123,"type":"User"}]}}]`
			prev := &before
			if scenario == "create" {
				desired.ID = 0
				prev = nil
			}
			if scenario == "combined" || scenario == "remove" {
				desired.Events = []string{"push"}
				expectedRules = `[{"type":"restrict_action_events","parameters":{"allowed_events":["push"]}},{"type":"restrict_actions_actors","parameters":{"allowed_actors":[{"id":4,"type":"RepositoryRole"},{"id":123,"type":"User"}]}}]`
			}
			if scenario == "remove" {
				desired.Actors = nil
				desired.DisableActors = true
				expectedRules = `[{"type":"restrict_action_events","parameters":{"allowed_events":["push"]}}]`
			}
			if scenario == "empty" {
				empty := []settings.ActionsPolicyActor{}
				desired.Actors = &empty
				expectedRules = `[{"type":"restrict_actions_actors","parameters":{"allowed_actors":[]}}]`
			}
			if scenario == "preserve" {
				desired.Actors = nil
				expectedRules = `[{"type":"restrict_actions_actors","parameters":{"allowed_actors":[{"id":5,"type":"RepositoryRole"},{"id":123,"type":"User"}]}}]`
			}
			change, err := settings.NewActionsEventPolicyChange(prev, desired)
			if err != nil {
				t.Fatal(err)
			}
			written := false
			calls := []string{}
			expected := `{"name":"renamed","enforcement":"disabled","rules":` + expectedRules + `}`
			final := `{"id":7,"name":"renamed","source_type":"Repository","source":"owner/current","target":"actions","enforcement":"disabled","rules":` + expectedRules + `}`
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				calls = append(calls, r.Method)
				body := actorPolicyJSON
				if r.Method != "GET" {
					data, _ := io.ReadAll(r.Body)
					var got, want any
					json.Unmarshal(data, &got)
					json.Unmarshal([]byte(expected), &want)
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("payload %s want %s", data, expected)
					}
					written = true
					body = final
					if scenario == "ack-mismatch" {
						body = actorPolicyJSON
					}
				} else if written {
					body = final
					if scenario == "readback-mismatch" {
						body = actorPolicyJSON
					}
				} else if scenario == "drift" {
					body = strings.Replace(actorPolicyJSON, `"id":123`, `"id":124`, 1)
				}
				return labelsResponse(r, 200, body), nil
			}))
			id, err := client.WriteActionsEventPolicy(eventPolicyRepo, change)
			bad := scenario == "drift" || scenario == "ack-mismatch" || scenario == "readback-mismatch"
			if (err != nil) != bad {
				t.Fatalf("id%d err%v calls%v", id, err, calls)
			}
			if scenario == "drift" && written {
				t.Fatal("wrote after actor drift")
			}
			if !bad {
				want := []string{"GET", "PUT", "GET"}
				if scenario == "create" {
					want = []string{"POST", "GET"}
				}
				if !reflect.DeepEqual(calls, want) {
					t.Fatal(calls)
				}
			}
			if bad && scenario != "drift" && id != 7 {
				t.Fatal(fmt.Sprint("lost identity ", id))
			}
		})
	}
}
