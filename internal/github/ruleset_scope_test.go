package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func scopedPR(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	return analysisParams(t, body, "pull_request")
}
func scopeBody(t *testing.T, value bool) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal([]byte(fullReviewRulesetJSON), &body); err != nil {
		t.Fatal(err)
	}
	scopedPR(t, body)["require_extra_approval_for_unattributed_changes"] = value
	return body
}

func TestRulesetScopeAcquisitionRejectsUnownedProtection(t *testing.T) {
	for _, mode := range []string{"quality", "coverage", "teams", "missing", "null", "wrong type"} {
		t.Run(mode, func(t *testing.T) {
			body := scopeBody(t, true)
			switch mode {
			case "quality", "coverage":
				body["rules"] = append(body["rules"].([]any), map[string]any{"type": "code_" + mode, "parameters": map[string]any{}})
			case "teams":
				scopedPR(t, body)["required_reviewers"] = []any{map[string]any{"reviewer": map[string]any{"type": "Team", "id": 1}, "minimum_approvals": 0, "file_patterns": []any{}}}
			case "missing":
				delete(scopedPR(t, body), "require_extra_approval_for_unattributed_changes")
			case "null":
				scopedPR(t, body)["require_extra_approval_for_unattributed_changes"] = nil
			case "wrong type":
				scopedPR(t, body)["require_extra_approval_for_unattributed_changes"] = "true"
			}
			data, _ := json.Marshal(body)
			var raw map[string]json.RawMessage
			json.Unmarshal(data, &raw)
			if _, _, err := decodeObservedRulesetPolicy(raw); err == nil {
				t.Fatal("projected unsupported protection")
			}
			if _, err := decodeRulesetPolicy(raw); err == nil {
				t.Fatal("strict decoder accepted unsupported protection")
			}
		})
	}
}

func TestRulesetAdditionalApprovalWriterChangesIntentAndChecksDrift(t *testing.T) {
	for _, value := range []bool{false, true} {
		for _, mode := range []string{"ok", "preflight drift", "preflight quality", "preflight teams", "preflight denied", "ack drift", "readback drift"} {
			t.Run(fmt.Sprintf("%t/%s", value, mode), func(t *testing.T) {
				beforeBody := scopeBody(t, value)
				data, _ := json.Marshal(beforeBody)
				before, err := decodeRulesetPolicyFromTest(data)
				if err != nil {
					t.Fatal(err)
				}
				afterBody := scopeBody(t, !value)
				afterBody["name"] = "edited"
				data, _ = json.Marshal(afterBody)
				after, err := decodeRulesetPolicyFromTest(data)
				if err != nil {
					t.Fatal(err)
				}
				change, err := settings.NewRulesetChange("update", 7, &before, &after)
				if err != nil {
					t.Fatal(err)
				}
				calls, writes := 0, 0
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					body := afterBody
					if calls == 1 {
						if req.Method != "GET" {
							t.Fatal("missing preflight", req.Method)
						}
						body = beforeBody
						switch mode {
						case "preflight denied":
							return labelsResponse(req, 403, `{"message":"denied"}`), nil
						case "preflight drift":
							scopedPR(t, body)["require_extra_approval_for_unattributed_changes"] = !value
						case "preflight quality":
							body["rules"] = append(body["rules"].([]any), map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "errors"}})
						case "preflight teams":
							scopedPR(t, body)["required_reviewers"] = []any{map[string]any{"reviewer": map[string]any{"type": "Team", "id": 1}}}
						}
					} else if req.Method == "PUT" {
						writes++
						var payload map[string]any
						if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
							t.Fatal(err)
						}
						if got := scopedPR(t, payload)["require_extra_approval_for_unattributed_changes"]; got != !value {
							t.Fatal("lost declared flag", got)
						}
						if mode == "ack drift" {
							scopedPR(t, body)["require_extra_approval_for_unattributed_changes"] = value
						}
					} else if calls == 3 && req.Method == "GET" {
						if mode == "readback drift" {
							scopedPR(t, body)["require_extra_approval_for_unattributed_changes"] = value
						}
					} else {
						t.Fatal(req.Method, calls)
					}
					data, _ := json.Marshal(body)
					return labelsResponse(req, 200, string(data)), nil
				}))
				_, err = client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
				if (err != nil) != (mode != "ok") {
					t.Fatal(mode, err)
				}
				if strings.HasPrefix(mode, "preflight") {
					if writes != 0 || calls != 1 {
						t.Fatal("wrote after failed preflight", calls, writes)
					}
				} else if writes != 1 {
					t.Fatal(writes)
				}
				if strings.Contains(mode, "drift") && !strings.HasPrefix(mode, "preflight") && !strings.Contains(err.Error(), "writes may already have applied") {
					t.Fatal("lost uncertainty", err)
				}
			})
		}
	}
}

func TestNewPRRuleDeclaresAdditionalApprovalAndConfirmsReadback(t *testing.T) {
	for _, declaration := range []struct {
		name  string
		value *bool
	}{
		{"omitted", nil}, {"false", rulesetPtr(false)}, {"true", rulesetPtr(true)},
	} {
		for _, mode := range []string{"ok", "ack mismatch", "readback mismatch", "readback missing", "readback denied"} {
			t.Run(declaration.name+"/"+mode, func(t *testing.T) {
				value := declaration.value == nil || *declaration.value
				body := scopeBody(t, value)
				data, _ := json.Marshal(body)
				policy, err := decodeRulesetPolicyFromTest(data)
				if err != nil {
					t.Fatal(err)
				}
				for _, rule := range *policy.Rules {
					if rule.Type == "pull_request" {
						rule.Parameters.RequireExtraApprovalForUnattributedChanges = declaration.value
					}
				}
				declarations, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{policy})
				if err != nil {
					t.Fatal(err)
				}
				desired := declarations.Values()[0]
				change, err := settings.NewRulesetChange("create", 0, nil, &desired)
				if err != nil {
					t.Fatal(err)
				}
				calls := 0
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						if req.Method != "POST" || req.URL.Path != "/repos/owner/current/rulesets" {
							t.Fatal(req.Method, req.URL)
						}
						var payload map[string]any
						if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
							t.Fatal(err)
						}
						if got := scopedPR(t, payload)["require_extra_approval_for_unattributed_changes"]; got != value {
							t.Fatal("lost normalized declaration", got)
						}
						if mode == "ack mismatch" {
							scopedPR(t, body)["require_extra_approval_for_unattributed_changes"] = !value
						}
					} else if calls == 2 {
						if req.Method != "GET" || req.URL.Path != "/repos/owner/current/rulesets/7" || req.URL.Query().Get("includes_parents") != "false" {
							t.Fatal(req.Method, req.URL)
						}
						switch mode {
						case "readback mismatch":
							scopedPR(t, body)["require_extra_approval_for_unattributed_changes"] = !value
						case "readback missing":
							delete(scopedPR(t, body), "require_extra_approval_for_unattributed_changes")
						case "readback denied":
							return labelsResponse(req, 403, `{"message":"denied"}`), nil
						}
					} else {
						t.Fatal("unexpected retry", calls)
					}
					data, _ := json.Marshal(body)
					return labelsResponse(req, 200, string(data)), nil
				}))
				id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
				wantCalls := 2
				if mode == "ack mismatch" {
					wantCalls = 1
				}
				if (err != nil) != (mode != "ok") || calls != wantCalls || id != 7 {
					t.Fatal("outcome or recovery ID", calls, id, err)
				}
				if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
					t.Fatal("lost uncertainty", err)
				}
			})
		}
	}
}
