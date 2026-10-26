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

const basicRulesetJSON = `{"id":7,"name":"basic","source_type":"Repository","source":"Owner/Current","current_user_can_bypass":"never","target":"branch","enforcement":"active","bypass_actors":[],"conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"rules":[{"type":"deletion"},{"type":"non_fast_forward"}]}`

func TestReadRulesetsCompleteOwnershipAndTargets(t *testing.T) {
	calls := []string{}
	client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.URL.RequestURI())
		if r.Method != "GET" || r.URL.Query().Get("includes_parents") != "false" {
			t.Fatal(r)
		}
		var body string
		switch r.URL.Path {
		case "/repos/owner/current/rulesets":
			body = `[{"id":7,"name":"basic","source_type":"Repository","source":"owner/current"},{"id":8,"name":"push","source_type":"Repository","source":"owner/current"},{"id":9,"name":"basic","source_type":"Organization","source":"owner"}]`
			if r.URL.Query().Get("per_page") != "100" || r.URL.Query().Get("page") != "1" {
				t.Fatal(r.URL)
			}
		case "/repos/owner/current/rulesets/7":
			body = basicRulesetJSON
		case "/repos/owner/current/rulesets/8":
			body = `{"id":8,"name":"push","source_type":"Repository","source":"owner/current","target":"push","rules":[{"type":"unknown"}]}`
		default:
			t.Fatal("unexpected path", r.URL)
		}
		return labelsResponse(r, 200, body), nil
	}))
	observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
	if err != nil {
		t.Fatal(err)
	}
	values, err := observed.Values()
	if err != nil || len(values) != 1 || values[0].ID != 7 || observed.ExcludedPush() != 1 || len(calls) != 3 {
		t.Fatalf("%+v %v %v", observed, err, calls)
	}
}
func TestRulesetsRejectIncompleteAndUnknownPolicy(t *testing.T) {
	paths := [][]string{{"future"}, {"conditions", "Ref_name"}, {"conditions", "future"}, {"conditions", "ref_name", "future"}, {"rules", "0", "future"}, {"rules", "0", "parameters", "future"}, {"bypass_actors", "0", "future"}}
	for _, path := range paths {
		t.Run(strings.Join(path, "/"), func(t *testing.T) {
			var value map[string]any
			json.Unmarshal([]byte(basicRulesetJSON), &value)
			value["bypass_actors"] = []any{map[string]any{"actor_type": "Team", "actor_id": 3, "bypass_mode": "always"}}
			node := value
			for _, key := range path[:len(path)-1] {
				if key == "rules" || key == "bypass_actors" {
					node = node[key].([]any)[0].(map[string]any)
				} else if key != "0" {
					if node[key] == nil {
						node[key] = map[string]any{}
					}
					node = node[key].(map[string]any)
				}
			}
			node[path[len(path)-1]] = true
			body, _ := json.Marshal(value)
			if _, err := decodeRulesetPolicyFromTest(body); err == nil {
				t.Fatal("unknown content accepted", string(body))
			}
		})
	}
	for _, body := range []string{strings.Replace(basicRulesetJSON, `"bypass_actors":[],`, "", 1), strings.Replace(basicRulesetJSON, `"bypass_actors":[]`, `"bypass_actors":null`, 1), strings.Replace(basicRulesetJSON, `"exclude":[]`, `"exclude":null`, 1), strings.Replace(basicRulesetJSON, `"exclude":[],`, "", 1), strings.Replace(basicRulesetJSON, `"type":"deletion"`, `"type":"merge_queue"`, 1), strings.Replace(basicRulesetJSON, `"type":"deletion"`, `"type":"deletion","parameters":null`, 1), strings.Replace(basicRulesetJSON, `"target":"branch"`, `"target":null`, 1), strings.Replace(basicRulesetJSON, `"rules":[{"type":"deletion"},{"type":"non_fast_forward"}]`, `"rules":null`, 1)} {
		if body == basicRulesetJSON {
			continue
		}
		if _, err := decodeRulesetPolicyFromTest([]byte(body)); err == nil {
			t.Fatal("incomplete accepted", body)
		}
	}
	// Explicit read-only metadata can be discarded, not arbitrary future fields.
	if _, err := decodeRulesetPolicyFromTest([]byte(strings.Replace(basicRulesetJSON, `"id":7`, `"id":7,"node_id":"n","created_at":"date","updated_at":"date","_links":{}`, 1))); err != nil {
		t.Fatal(err)
	}
}
func decodeRulesetPolicyFromTest(data []byte) (settings.RulesetDeclaration, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return settings.RulesetDeclaration{}, err
	}
	return decodeRulesetPolicy(raw)
}
func TestRulesetsPaginationAndFailure(t *testing.T) {
	for _, bad := range []bool{false, true} {
		calls := 0
		client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			if strings.HasSuffix(r.URL.Path, "/rulesets") {
				if r.URL.Query().Get("page") == "1" {
					items := []map[string]any{}
					for i := 1; i <= 100; i++ {
						items = append(items, map[string]any{"id": i, "source_type": "Repository", "source": "owner/current"})
					}
					body, _ := json.Marshal(items)
					return labelsResponse(r, 200, string(body)), nil
				}
				if bad {
					return labelsResponse(r, 403, `{"message":"denied"}`), nil
				}
				return labelsResponse(r, 200, `[]`), nil
			}
			var id int
			fmt.Sscanf(r.URL.Path, "/repos/owner/current/rulesets/%d", &id)
			body := strings.Replace(basicRulesetJSON, `"id":7`, fmt.Sprintf(`"id":%d`, id), 1)
			body = strings.Replace(body, `"name":"basic"`, fmt.Sprintf(`"name":"r-%d"`, id), 1)
			return labelsResponse(r, 200, body), nil
		}))
		o, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
		values, valueErr := o.Values()
		if bad {
			if err == nil || valueErr == nil || calls != 2 {
				t.Fatalf("partial result %v %v %d", o, err, calls)
			}
		} else if err != nil || len(values) != 100 || calls != 102 {
			t.Fatalf("%v %d %d", err, len(values), calls)
		}
	}
}
func TestRulesetsRejectInventoryAndDetailChanges(t *testing.T) {
	for _, tt := range []struct{ list, detail string }{{`null`, basicRulesetJSON}, {`[{"id":7,"source_type":"Repository","source":"other/repo"}]`, basicRulesetJSON}, {`[{"id":7,"source_type":"Repository","source":"owner/current"},{"id":7,"source_type":"Repository","source":"owner/current"}]`, basicRulesetJSON}, {`[{"id":7,"source_type":"Repository","source":"owner/current"}]`, strings.Replace(basicRulesetJSON, `"id":7`, `"id":8`, 1)}, {`[{"id":7,"source_type":"Repository","source":"owner/current"}]`, strings.Replace(basicRulesetJSON, `"source_type":"Repository"`, `"source_type":"Organization"`, 1)}} {
		client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
			body := tt.detail
			if strings.HasSuffix(r.URL.Path, "/rulesets") {
				body = tt.list
			}
			return labelsResponse(r, 200, body), nil
		}))
		o, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
		if _, valueErr := o.Values(); err == nil || valueErr == nil {
			t.Fatal("accepted", tt)
		}
	}
}
func TestRulesetsCRUDExactPolicyAndActorNull(t *testing.T) {
	policy, err := decodeRulesetPolicyFromTest([]byte(strings.Replace(basicRulesetJSON, `"bypass_actors":[]`, `"bypass_actors":[{"actor_type":"DeployKey","actor_id":null,"bypass_mode":"always"},{"actor_type":"OrganizationAdmin","actor_id":99,"bypass_mode":"always"}]`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	if (*policy.BypassActors)[0].ActorID != nil || (*policy.BypassActors)[1].ActorID != nil {
		t.Fatal("actor normalization")
	}
	before := policy
	disabled := "disabled"
	policy.Enforcement = &disabled
	for _, op := range []string{"create", "update"} {
		t.Run(op, func(t *testing.T) {
			var prior, after *settings.RulesetDeclaration
			id := int64(7)
			method := "PUT"
			path := "/repos/owner/current/rulesets/7"
			switch op {
			case "create":
				id = 0
				after = &policy
				method = "POST"
				path = "/repos/owner/current/rulesets"
			case "update":
				prior = &before
				selected := policy
				selected.ID = &id
				after = &selected
			case "delete":
				prior = &before
				method = "DELETE"
			}
			change, err := settings.NewRulesetChange(op, id, prior, after)
			if err != nil {
				t.Fatal(err)
			}
			client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != method || r.URL.Path != path {
					t.Fatal(r)
				}
				var data []byte
				if r.Body != nil {
					data, _ = io.ReadAll(r.Body)
				}
				if op == "delete" {
					if len(data) != 0 {
						t.Fatal("delete body")
					}
				} else {
					var value map[string]any
					json.Unmarshal(data, &value)
					if len(value) != 6 || value["id"] != nil || value["source"] != nil || value["enforcement"] != "disabled" {
						t.Fatal(string(data))
					}
					actors := value["bypass_actors"].([]any)
					if actor := actors[0].(map[string]any); actor["actor_type"] != "DeployKey" || !reflect.DeepEqual(actor["actor_id"], nil) {
						t.Fatal(actors)
					}
					if _, ok := actors[0].(map[string]any)["actor_id"]; !ok {
						t.Fatal("DeployKey null omitted")
					}
				}
				return rulesetMutationResponse(t, r, 200, string(data), 7), nil
			}))
			if _, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change); err != nil {
				t.Fatal(err)
			}
		})
	}
}
