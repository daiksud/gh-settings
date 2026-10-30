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

// teamApprovalsPayload is fixture intent, independent of constructed changes and requests.
func teamApprovalsPayload(mode string) map[string]any {
	body := approvalsIntendedPayload(0, false)
	if mode == "PR removed" {
		body["rules"] = []any{}
		return body
	}
	reviewers := []any{
		map[string]any{"reviewer": map[string]any{"type": "Team", "id": float64(2)}, "minimum_approvals": float64(0), "file_patterns": []any{"docs/**"}},
		map[string]any{"reviewer": map[string]any{"type": "Team", "id": float64(5)}, "minimum_approvals": float64(3), "file_patterns": []any{}},
	}
	if mode != "reviewer removed" {
		id, count := float64(10), float64(0)
		if mode == "different" {
			count = 10
		}
		if mode == "replacement" {
			id, count = 99, 4
		}
		reviewers = append(reviewers, map[string]any{"reviewer": map[string]any{"type": "Team", "id": id}, "minimum_approvals": count, "file_patterns": []any{"**", "!generated/**", "**"}})
	}
	body["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["required_reviewers"] = reviewers
	return body
}

func teamApprovalsChange(t *testing.T, mode string) settings.RulesetChange {
	t.Helper()
	var before, after settings.RulesetDeclaration
	for _, item := range []struct {
		value  any
		target *settings.RulesetDeclaration
	}{{teamApprovalsPayload("equal"), &before}, {teamApprovalsPayload(mode), &after}} {
		data, _ := json.Marshal(item.value)
		if err := json.Unmarshal(data, item.target); err != nil {
			t.Fatal(err)
		}
		id := int64(7)
		item.target.ID = &id
	}
	records := settings.RulesetNormalizations{TeamApprovals: []settings.RulesetTeamApprovalsNormalization{
		{ReviewerID: 10, Path: settings.RulesetTeamApprovalsPath, RawValue: 11, Fallback: 0},
		{ReviewerID: 2, Path: settings.RulesetTeamApprovalsPath, RawValue: -1, Fallback: 0},
	}}
	c, err := settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestTeamApprovalsRequireStrictPerIdentityAcknowledgmentAndFreshGET(t *testing.T) {
	for _, tc := range []struct {
		intent, ack, read string
		wantErr           bool
	}{
		{"equal", "canonical", "canonical", false}, {"different", "canonical", "canonical", false}, {"reviewer removed", "canonical", "canonical", false}, {"replacement", "canonical", "canonical", false}, {"PR removed", "canonical", "canonical", false},
		{"equal", "canonical", "raw11", true}, {"equal", "canonical", "rawnegative", true}, {"different", "canonical", "swapped", true},
		{"reviewer removed", "canonical", "retained", true}, {"replacement", "canonical", "retained", true}, {"PR removed", "canonical", "retained", true},
		{"equal", "raw11", "canonical", true}, {"equal", "canonical", "null", true}, {"equal", "canonical", "patterns", true},
	} {
		t.Run(tc.intent+"/"+tc.ack+"/"+tc.read, func(t *testing.T) {
			var operations []string
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				operations = append(operations, req.Method)
				if req.URL.Path != "/repos/owner/current/rulesets/7" {
					t.Fatal(req.URL)
				}
				mode := tc.read
				if req.Method == "PUT" {
					mode = tc.ack
					var got map[string]any
					if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(got, teamApprovalsPayload(tc.intent)) {
						t.Fatal("whole policy differs", got, teamApprovalsPayload(tc.intent))
					}
				} else if req.Method != "GET" || req.URL.Query().Get("includes_parents") != "false" {
					t.Fatal(req.Method, req.URL)
				}
				body := teamApprovalsPayload(tc.intent)
				if mode != "canonical" {
					body = teamApprovalsPayload("equal")
					reviewers := body["rules"].([]any)[0].(map[string]any)["parameters"].(map[string]any)["required_reviewers"].([]any)
					first, last := reviewers[0].(map[string]any), reviewers[2].(map[string]any)
					switch mode {
					case "raw11":
						last["minimum_approvals"] = 11
					case "rawnegative":
						first["minimum_approvals"] = -1
					case "swapped":
						first["minimum_approvals"] = 10
					case "null":
						last["minimum_approvals"] = nil
					case "patterns":
						last["file_patterns"] = []any{"**", "**", "!generated/**"}
					}
				}
				body["id"], body["source_type"], body["source"] = 7, "Repository", "owner/current"
				data, _ := json.Marshal(body)
				return labelsResponse(req, 200, string(data)), nil
			}))
			id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, teamApprovalsChange(t, tc.intent))
			want := []string{"PUT", "GET"}
			if tc.ack != "canonical" {
				want = []string{"PUT"}
			}
			if id != 7 || (err != nil) != tc.wantErr || !reflect.DeepEqual(operations, want) {
				t.Fatal(id, err, operations, want)
			}
			if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
				t.Fatal("lost uncertainty", err)
			}
		})
	}
}
