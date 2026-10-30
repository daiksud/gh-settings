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

func mergeMethodsChange(t *testing.T, methods []string) settings.RulesetChange {
	t.Helper()
	var before, after settings.RulesetDeclaration
	fixture := strings.Replace(approvalsRulesetJSON, `["squash"]`, `["merge","rebase","squash"]`, 1)
	if err := json.Unmarshal([]byte(fixture), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(fixture), &after); err != nil {
		t.Fatal(err)
	}
	if methods == nil {
		after.Rules = &[]settings.RulesetRule{}
	} else {
		(*after.Rules)[0].Parameters.AllowedMergeMethods = &methods
	}
	records := settings.RulesetNormalizations{MergeMethods: settings.RulesetMergeMethodsNormalization{Path: settings.RulesetMergeMethodsPath, RawValue: []string{"future", "squash"}, Fallback: []string{"merge", "squash", "rebase"}}}
	change, err := settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

// This fixture expresses literal whole-policy intent, independently of a change or request.
func mergeMethodsIntendedPayload(methods []string) map[string]any {
	body := approvalsIntendedPayload(0, false)
	if methods == nil {
		body["rules"] = []any{}
		return body
	}
	values := make([]any, len(methods))
	for i, method := range methods {
		values[i] = method
	}
	body["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["allowed_merge_methods"] = values
	return body
}

func TestNormalizedMergeMethodsRequireStrictAcknowledgmentAndIndependentRead(t *testing.T) {
	for _, test := range []struct {
		name      string
		methods   []string
		ack, read string
		wantErr   bool
	}{
		{"fallback", []string{"merge", "rebase", "squash"}, "canonical", "canonical", false},
		{"subset", []string{"squash"}, "canonical", "canonical", false},
		{"remove", nil, "canonical", "canonical", false},
		{"retained unknown", []string{"merge", "rebase", "squash"}, "canonical", "unknown", true},
		{"different subset", []string{"squash"}, "canonical", "different", true},
		{"removed rule retained", nil, "canonical", "retained", true},
		{"unknown acknowledgment", []string{"merge", "rebase", "squash"}, "unknown", "canonical", true},
		{"removal acknowledgment retains rule", nil, "retained", "canonical", true},
		{"null member", []string{"squash"}, "canonical", "null member", true},
		{"empty list", []string{"squash"}, "canonical", "empty", true},
		{"missing list", []string{"squash"}, "canonical", "missing", true},
		{"wrong identity", []string{"squash"}, "canonical", "wrong ID", true},
		{"denied read", []string{"squash"}, "canonical", "denied", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var operations []string
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				operations = append(operations, req.Method)
				if req.URL.Path != "/repos/owner/current/rulesets/7" {
					t.Fatal(req.URL)
				}
				mode := test.read
				if req.Method == "PUT" {
					mode = test.ack
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(payload, mergeMethodsIntendedPayload(test.methods)) {
						t.Fatal("unexpected whole-policy request", payload)
					}
				} else if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatal(req.Method, req.URL)
				}
				if mode == "denied" {
					return labelsResponse(req, 403, `{"message":"denied"}`), nil
				}
				body := mergeMethodsIntendedPayload(test.methods)
				body["id"], body["source_type"], body["source"] = 7, "Repository", "owner/current"
				if mode == "wrong ID" {
					body["id"] = 8
				}
				if mode != "canonical" && mode != "wrong ID" {
					retained := mergeMethodsIntendedPayload([]string{"merge", "rebase", "squash"})["rules"]
					params := retained.([]any)[0].(map[string]any)["parameters"].(map[string]any)
					switch mode {
					case "unknown":
						params["allowed_merge_methods"] = []any{"future", "squash"}
					case "different":
						params["allowed_merge_methods"] = []any{"merge"}
					case "null member":
						params["allowed_merge_methods"] = []any{nil}
					case "empty":
						params["allowed_merge_methods"] = []any{}
					case "missing":
						delete(params, "allowed_merge_methods")
					}
					body["rules"] = retained
				}
				encoded, err := json.Marshal(body)
				if err != nil {
					t.Fatal(err)
				}
				return labelsResponse(req, 200, string(encoded)), nil
			}))
			id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, mergeMethodsChange(t, test.methods))
			want := []string{"PUT", "GET"}
			if test.ack != "canonical" {
				want = []string{"PUT"}
			}
			if (err != nil) != test.wantErr || id != 7 || !reflect.DeepEqual(operations, want) {
				t.Fatal(id, err, operations, want)
			}
			if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
				t.Fatal("lost uncertain effects", err)
			}
		})
	}
}
