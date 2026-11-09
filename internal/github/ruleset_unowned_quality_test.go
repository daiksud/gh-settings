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

func qualityBody(t *testing.T, severity *string) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal([]byte(basicRulesetJSON), &body); err != nil {
		t.Fatal(err)
	}
	if severity != nil {
		body["rules"] = append(body["rules"].([]any), map[string]any{"type": "code_quality", "parameters": map[string]any{"severity": *severity}})
	}
	return body
}

func decodeQualityBody(t *testing.T, body map[string]any) settings.RulesetDeclaration {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	p, err := decodeRulesetPolicyFromTest(data)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUnownedQualityClosedAcquisition(t *testing.T) {
	for _, severity := range []string{"errors", "warnings", "notes", "all", "FUTURE_SEVERITY"} {
		p := decodeQualityBody(t, qualityBody(t, &severity))
		if got := settings.RulesetUnownedCodeQuality(&p); got == nil || *got != severity {
			t.Fatal("lost exact severity", got)
		}
		data, _ := json.Marshal(p)
		if strings.Contains(string(data), "code_quality") {
			t.Fatal("unowned rule leaked as declaration", string(data))
		}
	}
	for _, rule := range []string{
		`{"type":"code_quality"}`, `{"type":"code_quality","parameters":null}`,
		`{"type":"code_quality","parameters":{}}`, `{"type":"code_quality","parameters":{"severity":null}}`,
		`{"type":"code_quality","parameters":{"severity":false}}`, `{"type":"code_quality","parameters":{"severity":""}}`,
		`{"type":"code_quality","parameters":{"severity":"errors","future":true}}`,
		`{"type":"code_quality","parameters":{"severity":"errors"},"future":true}`,
	} {
		body := qualityBody(t, nil)
		var malformed map[string]any
		json.Unmarshal([]byte(rule), &malformed)
		body["rules"] = append(body["rules"].([]any), malformed)
		data, _ := json.Marshal(body)
		if _, err := decodeRulesetPolicyFromTest(data); err == nil {
			t.Fatal("accepted unknown/incomplete protection", rule)
		}
	}
	body := qualityBody(t, new("errors"))
	body["rules"] = append(body["rules"].([]any), body["rules"].([]any)[2])
	data, _ := json.Marshal(body)
	if _, err := decodeRulesetPolicyFromTest(data); err == nil {
		t.Fatal("duplicate quality accepted")
	}
}

func TestUnownedQualityMutationRetentionAndDrift(t *testing.T) {
	for _, initial := range []bool{false, true} {
		for _, mode := range []string{"ok", "preflight added", "preflight removed", "preflight changed", "preflight unknown", "ack added", "ack removed", "ack changed", "readback added", "readback removed", "readback changed", "readback denied"} {
			t.Run(fmt.Sprintf("present=%t/%s", initial, mode), func(t *testing.T) {
				var quality *string
				if initial {
					quality = new("FUTURE_SEVERITY")
				}
				before := decodeQualityBody(t, qualityBody(t, quality))
				afterBody := qualityBody(t, quality)
				afterBody["name"] = "edited"
				after := decodeQualityBody(t, afterBody)
				change, err := settings.NewRulesetChange("update", 7, &before, &after)
				if err != nil {
					t.Fatal(err)
				}
				calls, writes := 0, 0
				client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
					calls++
					stage := "readback"
					if calls == 1 {
						stage = "preflight"
						if req.Method != "GET" {
							t.Fatal("no preflight", req.Method)
						}
					} else if calls == 2 {
						stage = "ack"
						if req.Method != "PUT" {
							t.Fatal(req.Method)
						}
						writes++
						var payload map[string]any
						json.NewDecoder(req.Body).Decode(&payload)
						p := decodeQualityBody(t, payload)
						got := settings.RulesetUnownedCodeQuality(&p)
						if (got != nil) != initial || (initial && *got != *quality) {
							t.Fatal("payload changed unowned value", got)
						}
					} else if req.Method != "GET" {
						t.Fatal(req.Method)
					}
					value := quality
					if strings.HasPrefix(mode, stage+" ") {
						switch strings.TrimPrefix(mode, stage+" ") {
						case "added":
							value = new("errors")
						case "removed":
							value = nil
						case "changed":
							value = new("all")
						case "denied":
							return labelsResponse(req, 403, `{}`), nil
						}
					}
					body := qualityBody(t, value)
					if stage != "preflight" {
						body["name"] = "edited"
					}
					if mode == "preflight unknown" && stage == "preflight" {
						body["rules"] = append(body["rules"].([]any), map[string]any{"type": "future_rule"})
					}
					data, _ := json.Marshal(body)
					return labelsResponse(req, 200, string(data)), nil
				}))
				_, err = client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
				// No-quality updates need no readback. Removing an absent value is no drift.
				wantErr := mode != "ok" && !(!initial && (strings.HasSuffix(mode, " removed") || strings.HasPrefix(mode, "readback ")))
				if (err != nil) != wantErr {
					t.Fatal("unexpected result", err, calls, writes)
				}
				if err != nil && strings.HasPrefix(mode, "preflight ") && writes != 0 {
					t.Fatal("wrote after failed preflight")
				}
				if err != nil && writes != 0 && !strings.Contains(err.Error(), "writes may already have applied") {
					t.Fatal("lost write uncertainty", err)
				}
				if initial && mode == "ok" && calls != 3 {
					t.Fatal("missing independent readback", calls)
				}
			})
		}
	}
}

func TestUnownedQualityCannotAppearOnCreateOrDuringDeletion(t *testing.T) {
	for _, operation := range []string{"create", "delete"} {
		t.Run(operation, func(t *testing.T) {
			plain := decodeQualityBody(t, qualityBody(t, nil))
			var before, after *settings.RulesetDeclaration
			var id int64
			if operation == "delete" {
				id = 7
				plain.ID = &id
				before = &plain
			} else {
				after = &plain
			}
			change, err := settings.NewRulesetChange(operation, id, before, after)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if (operation == "delete" && req.Method != "GET") || (operation == "create" && req.Method != "POST") {
					t.Fatal("unexpected mutation", req.Method)
				}
				data, _ := json.Marshal(qualityBody(t, new("errors")))
				return labelsResponse(req, 200, string(data)), nil
			}))
			_, err = client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, change)
			if err == nil || calls != 1 {
				t.Fatal("accepted unexpected unowned rule", calls, err)
			}
			if operation == "create" && !strings.Contains(err.Error(), "writes may already have applied") {
				t.Fatal("lost creation uncertainty", err)
			}
		})
	}
}
