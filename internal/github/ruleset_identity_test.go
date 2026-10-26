package github

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Existing payload tests assert the request independently; add the immutable
// response identity/ownership metadata that GitHub returns, never to the request.
func rulesetMutationResponse(t *testing.T, req *http.Request, status int, policy string, id int64) *http.Response {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(policy), &value); err != nil {
		t.Fatal(err)
	}
	value["id"], value["source_type"], value["source"] = id, "Repository", "owner/current"
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return labelsResponse(req, status, string(data))
}

func TestRulesetMutationRequiresReturnedIdentityAndCompletePolicy(t *testing.T) {
	policy, err := decodeRulesetPolicyFromTest([]byte(basicRulesetJSON))
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"create", "update"} {
		var before *settings.RulesetDeclaration
		id := int64(0)
		if operation == "update" {
			prior := policy
			before, id = &prior, 7
		}
		disabled := "disabled"
		after := policy
		after.Enforcement = &disabled
		change, err := settings.NewRulesetChange(operation, id, before, &after)
		if err != nil {
			t.Fatal(err)
		}
		valid := strings.Replace(basicRulesetJSON, `"enforcement":"active"`, `"enforcement":"disabled"`, 1)
		for _, test := range []struct {
			name, body string
			status     int
			wantID     int64
			fail       bool
		}{
			{"complete", valid, 200, 7, false},
			{"missing ID", strings.Replace(valid, `"id":7,`, "", 1), 200, 0, true},
			{"wrong repository", strings.Replace(valid, "Owner/Current", "owner/other", 1), 200, 7, true},
			{"inherited", strings.Replace(valid, `"source_type":"Repository"`, `"source_type":"Organization"`, 1), 200, 7, true},
			{"incomplete policy", strings.Replace(valid, `"bypass_actors":[],`, "", 1), 200, 7, true},
			{"wrong policy", basicRulesetJSON, 200, 7, true},
			{"denied", `{"message":"fixture denied"}`, 403, 0, true},
			{"redirect", `{}`, 307, 0, true},
		} {
			t.Run(operation+"/"+test.name, func(t *testing.T) {
				calls := 0
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					wantMethod, wantPath := "POST", "/repos/owner/current/rulesets"
					if operation == "update" {
						wantMethod, wantPath = "PUT", wantPath+"/7"
					}
					if req.Method != wantMethod || req.URL.Path != wantPath {
						t.Fatal(req.Method, req.URL)
					}
					return labelsResponse(req, test.status, test.body), nil
				}))
				got, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
				if (err != nil) != test.fail || got != test.wantID || calls != 1 {
					t.Fatal(got, err, calls)
				}
			})
		}
		if operation == "update" {
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				return labelsResponse(req, 200, strings.Replace(valid, `"id":7`, `"id":8`, 1)), nil
			}))
			if _, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change); err == nil {
				t.Fatal("wrong update ID accepted")
			}
		}
	}
}
