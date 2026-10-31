package github

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func teamApprovalsReadBody(count int, companions bool) map[string]any {
	body := teamApprovalsPayload("equal")
	body["id"], body["source_type"], body["source"] = 7, "Repository", "owner/current"
	params := body["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)
	teams := params["required_reviewers"].([]any)
	teams[2].(map[string]any)["minimum_approvals"] = count
	if companions {
		teams[0].(map[string]any)["minimum_approvals"] = -1
		body["enforcement"] = "evaluate"
		params["required_approving_review_count"] = 11
		params["allowed_merge_methods"] = []any{"FUTURE_METHOD"}
		body["rules"] = append(body["rules"].([]any), map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": "FUTURE_SEVERITY"}})
		body["bypass_actors"] = []any{map[string]any{"actor_type": "Team", "actor_id": 9, "bypass_mode": "FUTURE_MODE"}}
	}
	// The wire order is deliberately different from ID order.
	teams[0], teams[2] = teams[2], teams[0]
	return body
}

func TestTeamApprovalsCompleteDecoderRetainsSixIndependentEvidenceFamilies(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, count := range []int{-1, 0, 10, 11, 100} {
			encoded, _ := json.Marshal(teamApprovalsReadBody(count, companions))
			var raw map[string]json.RawMessage
			json.Unmarshal(encoded, &raw)
			original := map[string]json.RawMessage{}
			for k, v := range raw {
				original[k] = bytes.Clone(v)
			}
			policy, records, err := decodeObservedRulesetPolicy(raw)
			if err != nil || !reflect.DeepEqual(raw, original) {
				t.Fatal(policy, records, err)
			}
			var want []settings.RulesetTeamApprovalsNormalization
			if count < 0 || count > 10 {
				want = append(want, settings.RulesetTeamApprovalsNormalization{ReviewerID: 10, Path: settings.RulesetTeamApprovalsPath, RawValue: count, Fallback: 0})
			}
			if companions {
				want = append(want, settings.RulesetTeamApprovalsNormalization{ReviewerID: 2, Path: settings.RulesetTeamApprovalsPath, RawValue: -1, Fallback: 0})
			}
			if !reflect.DeepEqual(records.TeamApprovals, want) {
				t.Fatal("ID-bound counts lost", records.TeamApprovals, want)
			}
			if companions && (records.Enforcement.RawValue != "evaluate" || records.RequiredApprovals.RawValue != 11 || records.MergeMethods.IsZero() || records.QualitySeverity.RawValue != "FUTURE_SEVERITY" || len(records.BypassModes) != 1) {
				t.Fatal("companions lost", records)
			}
			state := settings.RulesetState{ID: 7, Policy: policy, EnforcementNormalization: records.Enforcement, RequiredApprovalsNormalization: records.RequiredApprovals, MergeMethodsNormalization: records.MergeMethods, TeamApprovalsNormalizations: records.TeamApprovals, QualitySeverityNormalization: records.QualitySeverity, BypassModeNormalizations: records.BypassModes}
			observed, err := settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
			if err != nil {
				t.Fatal(err)
			}
			values, _ := observed.Values()
			teams, err := values[0].RawRequiredReviewers()
			first := 0
			if companions {
				first = -1
			}
			if err != nil || len(teams) != 3 || teams[0].Reviewer.ID != 2 || *teams[0].MinimumApprovals != first || teams[1].Reviewer.ID != 5 || *teams[1].MinimumApprovals != 3 || teams[2].Reviewer.ID != 10 || *teams[2].MinimumApprovals != count || !reflect.DeepEqual(*teams[2].FilePatterns, []string{"**", "!generated/**", "**"}) {
				t.Fatal(teams, err)
			}
			if (verifyReviewRulesetResponse(policy, raw) != nil) != (companions || count < 0 || count > 10) {
				t.Fatal("verification used projected counts")
			}
		}
	}
}

func TestTeamApprovalsHTTPReaderReturnsNoPartialObservation(t *testing.T) {
	for _, mode := range []string{"normalized", "absent", "empty", "null count", "missing count", "fraction", "missing patterns", "null patterns", "bad pattern", "duplicate", "non-Team", "unknown member", "later denied", "pagination denied"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatal(req.Method, req.URL)
				}
				if req.URL.Path == "/repos/owner/current/rulesets" {
					if req.URL.Query().Get("page") == "2" {
						return labelsResponse(req, 403, `{"message":"denied"}`), nil
					}
					items := []any{map[string]any{"id": 7, "source_type": "Repository", "source": "owner/current"}, map[string]any{"id": 9, "source_type": "Repository", "source": "owner/current"}}
					if mode == "pagination denied" {
						for len(items) < 100 {
							items = append(items, map[string]any{"source_type": "Organization", "source": "owner"})
						}
					}
					encoded, _ := json.Marshal(items)
					return labelsResponse(req, 200, string(encoded)), nil
				}
				body := teamApprovalsReadBody(11, true)
				if strings.HasSuffix(req.URL.Path, "/9") {
					body["id"], body["name"] = 9, "later"
					if mode == "later denied" {
						return labelsResponse(req, 403, `{"message":"denied"}`), nil
					}
					params := body["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)
					teams := params["required_reviewers"].([]any)
					first := teams[0].(map[string]any)
					switch mode {
					case "absent":
						body["rules"] = []any{}
					case "empty":
						params["required_reviewers"] = []any{}
					case "null count":
						first["minimum_approvals"] = nil
					case "missing count":
						delete(first, "minimum_approvals")
					case "fraction":
						first["minimum_approvals"] = 1.5
					case "missing patterns":
						delete(first, "file_patterns")
					case "null patterns":
						first["file_patterns"] = nil
					case "bad pattern":
						first["file_patterns"] = []any{false}
					case "duplicate":
						params["required_reviewers"] = append(teams, first)
					case "non-Team":
						first["reviewer"].(map[string]any)["type"] = "User"
					case "unknown member":
						first["future"] = true
					}
				} else if !strings.HasSuffix(req.URL.Path, "/7") {
					t.Fatal(req.URL)
				}
				encoded, _ := json.Marshal(body)
				return labelsResponse(req, 200, string(encoded)), nil
			}))
			client.RulesetTeamApprovalsNormalizationWarning = func(int64, string, settings.RulesetTeamApprovalsNormalization) { t.Fatal("pure reader warned") }
			observed, err := client.ReadRulesets(repository.Repository{Owner: "owner", Name: "current"})
			success := mode == "normalized" || mode == "absent" || mode == "empty"
			if (err == nil) != success {
				t.Fatal(mode, err)
			}
			if !success {
				wantCalls := 3
				if mode == "pagination denied" {
					wantCalls = 2
				}
				if calls != wantCalls || !reflect.DeepEqual(observed, settings.RulesetObservation{}) {
					t.Fatal("partial evidence or failure order", calls, observed)
				}
				return
			}
			values, err := observed.Values()
			if err != nil || len(values) != 2 || calls != 3 {
				t.Fatal(values, calls, err)
			}
			for i, v := range values {
				teams, err := v.RawRequiredReviewers()
				if err != nil {
					t.Fatal(err)
				}
				if i == 1 && (mode == "absent" || mode == "empty") {
					if len(teams) != 0 || (teams == nil) != (mode == "absent") || len(v.TeamApprovalsNormalizations) != 0 {
						t.Fatal("absence became reviewer", teams, v)
					}
				} else if len(teams) != 3 || *teams[0].MinimumApprovals != -1 || *teams[2].MinimumApprovals != 11 {
					t.Fatal(teams)
				}
			}
		})
	}
}

func TestTeamApprovalsFreshWarningsNeverConfirmRetainedCounts(t *testing.T) {
	type event struct {
		name string
		id   int64
		raw  int
	}
	for _, mode := range []string{"canonical", "unknown", "renamed", "malformed", "wrong ID", "foreign", "unrelated", "denied"} {
		t.Run(mode, func(t *testing.T) {
			var events []event
			var operations []string
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				operations = append(operations, req.Method)
				body := teamApprovalsReadBody(0, false)
				if req.Method == "GET" {
					if mode == "denied" {
						return labelsResponse(req, 403, `{"message":"denied"}`), nil
					}
					if mode != "canonical" {
						body = teamApprovalsReadBody(12, false)
					}
					switch mode {
					case "renamed":
						body["name"] = "fresh"
					case "wrong ID":
						body["id"] = 8
					case "foreign":
						body["source"] = "owner/other"
					case "unrelated":
						body["future"] = true
					case "malformed":
						body["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["required_reviewers"].([]any)[0].(map[string]any)["minimum_approvals"] = nil
					}
				} else if req.Method != "PUT" {
					t.Fatal(req.Method)
				}
				encoded, _ := json.Marshal(body)
				return labelsResponse(req, 200, string(encoded)), nil
			}))
			client.RulesetTeamApprovalsNormalizationWarning = func(id int64, name string, n settings.RulesetTeamApprovalsNormalization) {
				if id != 7 || n.Path != settings.RulesetTeamApprovalsPath || n.Fallback != 0 {
					t.Fatal(id, name, n)
				}
				events = append(events, event{name: name, id: n.ReviewerID, raw: n.RawValue})
			}
			id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, teamApprovalsChange(t, "equal"))
			want := []event{{"approvals", 2, -1}, {"approvals", 10, 11}}
			if mode == "unknown" || mode == "renamed" {
				name := "approvals"
				if mode == "renamed" {
					name = "fresh"
				}
				want = append(want, event{name, 10, 12})
			}
			if id != 7 || (err != nil) != (mode != "canonical") || !reflect.DeepEqual(events, want) || !reflect.DeepEqual(operations, []string{"PUT", "GET"}) {
				t.Fatal(id, err, events, want, operations)
			}
			if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
				t.Fatal(err)
			}
		})
	}
}
