package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

const pushRulesetJSON = `{"id":8,"name":"push","source_type":"Repository","source":"owner/current","target":"push","enforcement":"active","conditions":null,"bypass_actors":[],"rules":[{"type":"file_extension_restriction","parameters":{"restricted_file_extensions":[".exe"]}},{"type":"file_path_restriction","parameters":{"restricted_file_paths":["secrets/**"],"ignored_file_paths":["secrets/example"]}},{"type":"max_file_path_length","parameters":{"max_file_path_length":240}},{"type":"max_file_size","parameters":{"max_file_size":20,"ignored_file_paths":[]}}]}`

func TestPushRulesetsStrictAcquisitionAndUIMapping(t *testing.T) {
	policy, err := decodeRulesetPolicyFromTest([]byte(pushRulesetJSON))
	if err != nil {
		t.Fatal(err)
	}
	ui, err := RulesetObservation(policy)
	if err != nil {
		t.Fatal(err)
	}
	if ui.TargetPushes == nil || !*ui.TargetPushes || ui.TargetBranches != nil || ui.TargetTags != nil || policy.Conditions != nil {
		t.Fatal("push target not retained", ui)
	}
	declaration, err := RulesetDeclaration(ui.Ruleset)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{declaration})
	if err != nil {
		t.Fatal(err)
	}
	observed, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 8, Policy: policy}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	values, _ := observed.Values()
	if !reflect.DeepEqual(canonical.Values()[0], values[0].Policy) {
		t.Fatal("mapping dropped protection", canonical.Values(), values)
	}
	*ui.Rules.RestrictFilePaths.AllowedExceptions = append(*ui.Rules.RestrictFilePaths.AllowedExceptions, "changed")
	if len(*(*policy.Rules)[1].Parameters.IgnoredFilePaths) != 1 {
		t.Fatal("aliased observed exceptions")
	}
	for _, replacement := range [][2]string{
		{`"max_file_size":20`, `"max_file_size":null`}, {`"max_file_size":20`, `"max_file_size":1.5`}, {`"max_file_size":20`, `"max_file_size":101`}, {`,"ignored_file_paths":[]`, ``},
		{`"max_file_path_length":240`, `"max_file_path_length":0`}, {`"restricted_file_extensions":[".exe"]`, `"restricted_file_extensions":[null]`},
		{`"restricted_file_paths":["secrets/**"]`, `"restricted_file_paths":null`}, {`"ignored_file_paths":[]`, `"ignored_file_paths":null`},
		{`"max_file_size":20`, `"Max_file_size":20`}, {`"max_file_size":20`, `"max_file_size":20,"future":true`},
		{`"max_file_size":20`, `"ignored_file_paths":[],"max_file_size":20,"severity":"errors"`},
		{`"type":"max_file_size"`, `"type":"future_push_rule"`}, {`"conditions":null`, `"conditions":{"ref_name":{"include":[],"exclude":[]}}`},
	} {
		body := strings.Replace(pushRulesetJSON, replacement[0], replacement[1], 1)
		if body == pushRulesetJSON {
			t.Fatal("invalid fixture replacement", replacement)
		}
		if _, err := decodeRulesetPolicyFromTest([]byte(body)); err == nil {
			t.Fatal("lossy protection acquired", replacement)
		}
	}
	for _, conditions := range []string{`"conditions":null,`, ""} {
		body := strings.Replace(pushRulesetJSON, `"conditions":null,`, conditions, 1)
		if _, err := decodeRulesetPolicyFromTest([]byte(body)); err != nil {
			t.Fatal("documented absence rejected", err)
		}
	}
}

func TestPushRulesetApplyChecksIndependentReadback(t *testing.T) {
	for _, stale := range []bool{false, true} {
		before, err := decodeRulesetPolicyFromTest([]byte(pushRulesetJSON))
		if err != nil {
			t.Fatal(err)
		}
		observed, _ := settings.NewRulesetObservation([]settings.RulesetState{{ID: 8, Policy: before}}, 0)
		after, _ := decodeRulesetPolicyFromTest([]byte(pushRulesetJSON))
		after.ID = rulesetPtr(int64(8))
		(*after.Rules)[3].Parameters.MaxFileSize = rulesetPtr(10)
		desired, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{after})
		if err != nil {
			t.Fatal(err)
		}
		changes, err := settings.ReconcileRulesets(observed, &desired)
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		methods := []string{}
		client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
			methods = append(methods, r.Method)
			if r.URL.Path != "/repos/owner/current/rulesets/8" {
				t.Fatal(r.URL)
			}
			expected := `{"name":"push","target":"push","enforcement":"active","bypass_actors":[],"rules":[{"type":"file_extension_restriction","parameters":{"restricted_file_extensions":[".exe"]}},{"type":"file_path_restriction","parameters":{"restricted_file_paths":["secrets/**"],"ignored_file_paths":["secrets/example"]}},{"type":"max_file_path_length","parameters":{"max_file_path_length":240}},{"type":"max_file_size","parameters":{"max_file_size":10,"ignored_file_paths":[]}}]}`
			if r.Method == "PUT" {
				var actual, want any
				json.NewDecoder(r.Body).Decode(&actual)
				json.Unmarshal([]byte(expected), &want)
				if !reflect.DeepEqual(actual, want) {
					t.Fatal("whole-object protection lost", actual, want)
				}
			} else if r.Method != "GET" {
				t.Fatal(r.Method)
			}
			if stale && r.Method == "GET" {
				return labelsResponse(r, 200, pushRulesetJSON), nil
			}
			return rulesetMutationResponse(t, r, 200, expected, 8), nil
		}))
		_, err = client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, changes[0])
		if (err != nil) != stale || !reflect.DeepEqual(methods, []string{"PUT", "GET"}) {
			t.Fatal(stale, methods, err)
		}
		if !stale {
			current, _ := settings.NewRulesetObservation([]settings.RulesetState{{ID: 8, Policy: *changes[0].After()}}, 0)
			again, err := settings.ReconcileRulesets(current, &desired)
			if err != nil || len(again) != 0 {
				t.Fatal("repeat differed", again, err)
			}
		}
	}
}

func TestPushRulesetCreationAndRuleRemoval(t *testing.T) {
	for _, create := range []bool{true, false} {
		policy, err := decodeRulesetPolicyFromTest([]byte(pushRulesetJSON))
		if err != nil {
			t.Fatal(err)
		}
		observed, _ := settings.NewRulesetObservation([]settings.RulesetState{}, 0)
		if !create {
			observed, _ = settings.NewRulesetObservation([]settings.RulesetState{{ID: 8, Policy: policy}}, 0)
			policy.ID = rulesetPtr(int64(8))
			policy.Rules = rulesetPtr([]settings.RulesetRule{})
		}
		desired, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{policy})
		if err != nil {
			t.Fatal(err)
		}
		changes, err := settings.ReconcileRulesets(observed, &desired)
		if err != nil || len(changes) != 1 {
			t.Fatal(changes, err)
		}
		methods := []string{}
		client := topicsClient(t, transportFunc(func(r *http.Request) (*http.Response, error) {
			methods = append(methods, r.Method)
			if create && r.Method == "POST" {
				if r.URL.Path != "/repos/owner/current/rulesets" {
					t.Fatal(r.URL)
				}
			} else if r.URL.Path != "/repos/owner/current/rulesets/8" {
				t.Fatal(r.URL)
			}
			if r.Method != "GET" {
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if len(body) != 5 || body["conditions"] != nil || body["id"] != nil {
					t.Fatal("push payload contains refs or metadata", body)
				}
				if !create && len(body["rules"].([]any)) != 0 {
					t.Fatal("rules not explicitly removed", body)
				}
			}
			expected := pushRulesetJSON
			if !create {
				expected = `{"id":8,"name":"push","source_type":"Repository","source":"owner/current","target":"push","enforcement":"active","bypass_actors":[],"rules":[]}`
			}
			return labelsResponse(r, 200, expected), nil
		}))
		id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, changes[0])
		if err != nil || id != 8 {
			t.Fatal(id, err)
		}
		expectedMethod := "POST"
		if !create {
			expectedMethod = "PUT"
		}
		if !reflect.DeepEqual(methods, []string{expectedMethod, "GET"}) {
			t.Fatal("fresh readback missing", methods)
		}
	}
}

func TestPushTargetSelectionRejectsInvalidCombinations(t *testing.T) {
	policy, err := decodeRulesetPolicyFromTest([]byte(pushRulesetJSON))
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := RulesetObservation(policy)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(){
		func() { prepared.TargetPushes = new(false) },
		func() { prepared.TargetBranches = &uimodel.BranchTargets{} },
		func() { prepared.TargetTags = &uimodel.TagTargets{} },
	} {
		prepared, err = RulesetObservation(policy)
		if err != nil {
			t.Fatal(err)
		}
		mutate()
		if _, err := RulesetDeclaration(prepared.Ruleset); err == nil {
			t.Fatal("invalid push target combination accepted", prepared)
		}
	}
}
