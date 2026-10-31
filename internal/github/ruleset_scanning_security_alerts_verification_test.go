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

func scanningSecurityAlertsPayload(mode string) map[string]any {
	body := approvalsIntendedPayload(0, false)
	tools := []any{map[string]any{"tool": "CodeQL", "security_alerts_threshold": "high_or_higher", "alerts_threshold": "errors"}}
	if mode != "tool removed" {
		name, security_alerts := "z tool", "high_or_higher"
		if mode == "different" {
			security_alerts = "all"
		}
		if mode == "replacement" {
			name, security_alerts = "zz new tool", "none"
		}
		tools = append(tools, map[string]any{"tool": name, "security_alerts_threshold": security_alerts, "alerts_threshold": "all"})
	}
	if mode == "empty tools" {
		tools = []any{}
	}
	body["rules"] = []any{map[string]any{"type": "code_scanning", "parameters": map[string]any{"code_scanning_tools": tools}}}
	if mode == "rule removed" {
		body["rules"] = []any{}
	}
	return body
}
func scanningSecurityAlertsChange(t *testing.T, mode string) settings.RulesetChange {
	t.Helper()
	var before, after settings.RulesetDeclaration
	for _, x := range []struct {
		body any
		out  *settings.RulesetDeclaration
	}{{scanningSecurityAlertsPayload("equal"), &before}, {scanningSecurityAlertsPayload(mode), &after}} {
		data, _ := json.Marshal(x.body)
		if err := json.Unmarshal(data, x.out); err != nil {
			t.Fatal(err)
		}
		id := int64(7)
		x.out.ID = &id
	}
	records := settings.RulesetNormalizations{ScanningSecurityAlerts: []settings.RulesetScanningSecurityAlertsNormalization{{Tool: "z tool", Path: settings.RulesetScanningSecurityAlertsPath, RawValue: "future", Fallback: "high_or_higher"}, {Tool: "CodeQL", Path: settings.RulesetScanningSecurityAlertsPath, RawValue: "", Fallback: "high_or_higher"}}}
	c, err := settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func TestScanningSecurityAlertsStrictConfirmationAndScopedFreshWarnings(t *testing.T) {
	for _, tc := range []struct {
		intent, ack, read string
		wantErr           bool
		fresh             int
	}{
		{"equal", "canonical", "canonical", false, 0}, {"different", "canonical", "canonical", false, 0}, {"tool removed", "canonical", "canonical", false, 0}, {"replacement", "canonical", "canonical", false, 0}, {"empty tools", "canonical", "canonical", false, 0}, {"rule removed", "canonical", "canonical", false, 0},
		{"equal", "canonical", "raw", true, 1}, {"equal", "raw", "canonical", true, 0}, {"different", "canonical", "swapped", true, 0}, {"tool removed", "canonical", "retained", true, 0}, {"replacement", "canonical", "retained", true, 0}, {"rule removed", "canonical", "retained", true, 0}, {"empty tools", "canonical", "retained", true, 0},
		{"equal", "canonical", "null", true, 0}, {"equal", "canonical", "foreign", true, 0}, {"equal", "canonical", "denied", true, 0},
	} {
		t.Run(tc.intent+"/"+tc.ack+"/"+tc.read, func(t *testing.T) {
			var operations, warnings []string
			client := topicsClient(t, transportFunc(func(req *http.Request) (*http.Response, error) {
				operations = append(operations, req.Method)
				mode := tc.read
				if req.URL.Path != "/repos/owner/current/rulesets/7" {
					t.Fatal(req.URL)
				}
				if req.Method == "PUT" {
					mode = tc.ack
					var payload map[string]any
					json.NewDecoder(req.Body).Decode(&payload)
					if !reflect.DeepEqual(payload, scanningSecurityAlertsPayload(tc.intent)) {
						t.Fatal(payload)
					}
					if !reflect.DeepEqual(warnings, []string{"CodeQL:", "z tool:future"}) {
						t.Fatal("warning before write", warnings)
					}
				} else if req.Method != "GET" {
					t.Fatal(req.Method)
				}
				if mode == "denied" {
					return labelsResponse(req, 403, `{}`), nil
				}
				body := scanningSecurityAlertsPayload(tc.intent)
				if mode != "canonical" {
					body = scanningSecurityAlertsPayload("equal")
					tools := scanningTools(body)
					switch mode {
					case "raw":
						tools[1].(map[string]any)["security_alerts_threshold"] = "FRESH"
					case "swapped":
						tools[0].(map[string]any)["security_alerts_threshold"] = "all"
					case "null":
						tools[0].(map[string]any)["security_alerts_threshold"] = nil
					}
				}
				body["id"], body["source_type"], body["source"] = 7, "Repository", "owner/current"
				if mode == "foreign" {
					body["source"] = "owner/other"
					scanningTools(body)[0].(map[string]any)["security_alerts_threshold"] = "PRIVATE"
				}
				data, _ := json.Marshal(body)
				return labelsResponse(req, 200, string(data)), nil
			}))
			client.RulesetScanningSecurityAlertsNormalizationWarning = func(id int64, name string, record settings.RulesetScanningSecurityAlertsNormalization) {
				if id != 7 || name != "approvals" {
					t.Fatal(id, name)
				}
				warnings = append(warnings, record.Tool+":"+record.RawValue)
				record.Tool = "corrupt"
			}
			id, err := client.ApplyRuleset(repository.Repository{Owner: "owner", Name: "current"}, scanningSecurityAlertsChange(t, tc.intent))
			want := []string{"PUT", "GET"}
			if tc.ack != "canonical" {
				want = []string{"PUT"}
			}
			if id != 7 || (err != nil) != tc.wantErr || !reflect.DeepEqual(operations, want) || len(warnings) != 2+tc.fresh {
				t.Fatal(id, err, operations, warnings)
			}
			if err != nil && !strings.Contains(err.Error(), "writes may already have applied") {
				t.Fatal("uncertainty", err)
			}
		})
	}
}
