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
	"github.com/daiksud/gh-settings/internal/workspace"
)

const copilotRulesetJSON = `{"id":7,"name":"copilot","source_type":"Repository","source":"owner/current","target":"branch","enforcement":"active","conditions":{"ref_name":{"include":["~DEFAULT_BRANCH"],"exclude":[]}},"bypass_actors":[],"rules":[{"type":"copilot_code_review","parameters":{"review_draft_pull_requests":false,"review_on_push":true}},{"type":"deletion"}]}`

func TestCopilotRulesetUIAndExport(t *testing.T) {
	for _, draft := range []bool{false, true} {
		for _, push := range []bool{false, true} {
			ui := uiRulesetBase()
			ui.Rules.AutomaticallyRequestCopilotCodeReview = &uimodel.CopilotCodeReviewRule{ReviewNewPushes: &push, ReviewDraftPullRequests: &draft}
			got, err := RulesetDeclaration(ui)
			if err != nil {
				t.Fatal(err)
			}
			rules := *got.Rules
			if len(rules) != 1 || rules[0].Type != "copilot_code_review" || *rules[0].Parameters.ReviewOnPush != push || *rules[0].Parameters.ReviewDraftPullRequests != draft {
				t.Fatal(rules)
			}
			normalized, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{got})
			if err != nil {
				t.Fatal(err)
			}
			observed, err := RulesetObservation(normalized.Values()[0])
			if err != nil || !reflect.DeepEqual(observed.Rules.AutomaticallyRequestCopilotCodeReview, ui.Rules.AutomaticallyRequestCopilotCodeReview) {
				t.Fatal(observed, err)
			}
		}
	}
	ui := uiRulesetBase()
	ui.Rules.AutomaticallyRequestCopilotCodeReview = &uimodel.CopilotCodeReviewRule{}
	got, err := RulesetDeclaration(ui)
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := settings.NewRulesetDeclarations([]settings.RulesetDeclaration{got})
	if err != nil {
		t.Fatal(err)
	}
	got = normalized.Values()[0]
	if *(*got.Rules)[0].Parameters.ReviewOnPush || *(*got.Rules)[0].Parameters.ReviewDraftPullRequests {
		t.Fatal(got, err)
	}
	ui.TargetBranches = nil
	ui.TargetTags = &uimodel.TagTargets{}
	if _, err := RulesetDeclaration(ui); err == nil {
		t.Fatal("tag accepted")
	}
	policy, err := decodeRulesetPolicyFromTest([]byte(copilotRulesetJSON))
	if err != nil {
		t.Fatal(err)
	}
	obs, err := settings.NewRulesetObservation([]settings.RulesetState{{ID: 7, Policy: policy}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	exported, err := obs.ExportDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	source := uiExportBase()
	source.Rulesets = &exported
	config, err := UIFromExportDeclaration(source)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := workspace.EncodeConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), "automatically_request_copilot_code_review:") || !strings.Contains(string(encoded), "review_new_pushes: true") {
		t.Fatal(string(encoded))
	}
	decoded, err := workspace.DecodeConfig(encoded)
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := DeclarationFromUI(decoded)
	if err != nil {
		t.Fatal(err)
	}
	changes, err := settings.ReconcileRulesets(obs, declaration.Rulesets)
	if err != nil || len(changes) != 0 {
		t.Fatal(changes, err)
	}
	// Both constructors and returned values own checkbox pointers.
	*(*policy.Rules)[0].Parameters.ReviewOnPush = false
	values, _ := obs.Values()
	if !*(*values[0].Policy.Rules)[0].Parameters.ReviewOnPush {
		t.Fatal("aliased observed option")
	}
}

func TestCopilotRulesetAcquisitionFailsClosed(t *testing.T) {
	for _, params := range []string{`{}`, `{"review_on_push":true}`, `{"review_on_push":true,"review_draft_pull_requests":null}`, `{"review_on_push":"true","review_draft_pull_requests":false}`, `{"review_on_push":true,"review_draft_pull_requests":false,"severity":"errors"}`, `{"review_on_push":true,"review_draft_pull_requests":false,"future":true}`} {
		raw := strings.Replace(copilotRulesetJSON, `{"review_draft_pull_requests":false,"review_on_push":true}`, params, 1)
		if _, err := decodeRulesetPolicyFromTest([]byte(raw)); err == nil {
			t.Fatal("incomplete/unknown parameters accepted", params)
		}
	}
	// Copilot parameters cannot be silently discarded on an unrelated rule.
	if _, err := decodeRulesetPolicyFromTest([]byte(strings.Replace(copilotRulesetJSON, `"copilot_code_review"`, `"deletion"`, 1))); err == nil {
		t.Fatal("cross-rule parameters accepted")
	}
}

func TestCopilotRulesetStrictIndependentReadback(t *testing.T) {
	for _, remove := range []bool{false, true} {
		for _, mode := range []string{"canonical", "drift", "denied", "missing", "wrong-id", "wrong-owner", "bad-ack"} {
			t.Run(mode+map[bool]string{true: "-remove", false: "-update"}[remove], func(t *testing.T) {
				before, err := decodeRulesetPolicyFromTest([]byte(copilotRulesetJSON))
				if err != nil {
					t.Fatal(err)
				}
				after, err := decodeRulesetPolicyFromTest([]byte(copilotRulesetJSON))
				if err != nil {
					t.Fatal(err)
				}
				if remove {
					*after.Rules = (*after.Rules)[1:]
				} else {
					*(*after.Rules)[0].Parameters.ReviewDraftPullRequests = true
				}
				change, err := settings.NewRulesetChange("update", 7, &before, &after)
				if err != nil {
					t.Fatal(err)
				}
				calls := []string{}
				client := rulesetMutationClient(t, change, transportFunc(func(req *http.Request) (*http.Response, error) {
					calls = append(calls, req.Method)
					if req.URL.Path != "/repos/owner/current/rulesets/7" {
						t.Fatal(req.URL)
					}
					response := strings.Replace(copilotRulesetJSON, `"review_draft_pull_requests":false`, `"review_draft_pull_requests":true`, 1)
					if remove {
						response = strings.Replace(copilotRulesetJSON, `{"type":"copilot_code_review","parameters":{"review_draft_pull_requests":false,"review_on_push":true}},`, "", 1)
					}
					if req.Method == "PUT" {
						var got map[string]any
						json.NewDecoder(req.Body).Decode(&got)
						var want map[string]any
						json.Unmarshal([]byte(response), &want)
						delete(want, "id")
						delete(want, "source")
						delete(want, "source_type")
						if !reflect.DeepEqual(got, want) {
							t.Fatal("payload", got, want)
						}
					} else if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
						t.Fatal(req.Method, req.URL)
					}
					if (req.Method == "GET" && mode == "drift") || (req.Method == "PUT" && mode == "bad-ack") {
						response = copilotRulesetJSON
					}
					if req.Method == "GET" {
						switch mode {
						case "denied":
							return labelsResponse(req, 403, `{"message":"denied"}`), nil
						case "missing":
							response = strings.Replace(response, `"rules":`, `"unobserved_rules":`, 1)
						case "wrong-id":
							response = strings.Replace(response, `"id":7`, `"id":8`, 1)
						case "wrong-owner":
							response = strings.Replace(response, `"source":"owner/current"`, `"source":"owner/foreign"`, 1)
						}
					}
					return labelsResponse(req, 200, response), nil
				}))
				id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
				wantCalls := []string{"PUT", "GET"}
				if mode == "bad-ack" {
					wantCalls = []string{"PUT"}
				}
				if id != 7 || (err != nil) != (mode != "canonical") || !reflect.DeepEqual(calls, wantCalls) {
					t.Fatal(id, err, calls)
				}
				if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
					t.Fatal("uncertainty lost", err)
				}
			})
		}
	}
}
