//go:build acceptance

package acceptance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func workflowTargetInput(id bool, enforcement, target string) string {
	input := strings.Replace(actionsEventPoliciesInput, "enforcement_status: active", "enforcement_status: "+enforcement, 1)
	if id {
		input = strings.Replace(input, "- policy_name:", "- id: 17\n          policy_name:", 1)
	}
	if target != "" {
		input += "          target_workflows: " + target + "\n"
	}
	return input
}

// These literal payloads deliberately do not use the product's projection helpers.
func workflowTargetPayload(enforcement string, include, exclude []any) map[string]any {
	payload := map[string]any{"name": "policy-canary", "enforcement": enforcement, "rules": []any{map[string]any{"type": "restrict_action_events", "parameters": map[string]any{"allowed_events": []any{"push"}}}}}
	if include != nil {
		payload["conditions"] = map[string]any{"workflow_path": map[string]any{"include": include, "exclude": exclude}}
	}
	return payload
}

func TestActionsWorkflowTargetsValidationBeforeRemoteProcess(t *testing.T) {
	for name, target := range map[string]string{
		"missing exclude":   "{include: [ci.yml]}",
		"empty criteria":    "{include: [], exclude: []}",
		"mixed sentinel":    "{include: ['~ALL', ci.yml], exclude: []}",
		"excluded sentinel": "{include: [ci.yml], exclude: ['~ALL']}",
		"wrong type":        "{include: ci.yml, exclude: []}",
	} {
		t.Run(name, func(t *testing.T) {
			f := newActionsEventPoliciesFixture(t, "")
			f.write(t, workflowTargetInput(false, "active", target))
			for _, command := range []string{"validate", "plan", "apply"} {
				_, diagnostic := f.run(t, true, command)
				if !strings.Contains(diagnostic, "workflow") {
					t.Fatal(diagnostic)
				}
			}
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.calls) != 0 {
				t.Fatal("malformed target contacted remote", f.calls)
			}
		})
	}
}

func TestActionsWorkflowTargetsPlanReportSummaryProcess(t *testing.T) {
	f := newActionsEventPoliciesFixture(t, "")
	input := workflowTargetInput(false, "active", "{include: ['.github/workflows/ci*.yml'], exclude: ['.github/workflows/ci-skip.yml']}")
	f.write(t, input)
	out, _ := f.run(t, false, "plan")
	for _, want := range []string{"target_workflows", ".github/workflows/ci*.yml", ".github/workflows/ci-skip.yml"} {
		if !strings.Contains(out, want) {
			t.Fatal(out)
		}
	}
	report := filepath.Join(f.fixture.root, "target-plan.json")
	f.run(t, false, "action", "plan", report)
	raw, err := os.ReadFile(report)
	var wire struct {
		Managed  int `json:"managed"`
		Policies []struct {
			After struct {
				Target map[string][]string `json:"target_workflows"`
			} `json:"after"`
		} `json:"actions_policies"`
	}
	if err != nil || json.Unmarshal(raw, &wire) != nil || wire.Managed != 1 || len(wire.Policies) != 1 {
		t.Fatal(string(raw), err)
	}
	want := map[string][]string{"include": {".github/workflows/ci*.yml"}, "exclude": {".github/workflows/ci-skip.yml"}}
	if !reflect.DeepEqual(wire.Policies[0].After.Target, want) {
		t.Fatal(string(raw))
	}
	for _, mode := range []string{"detailed", "status-only"} {
		summary := filepath.Join(f.fixture.root, mode+".md")
		args := []string{"action", "summary", report, summary}
		if mode == "status-only" {
			args = append(args, mode)
		}
		f.run(t, false, args...)
		data, err := os.ReadFile(summary)
		if err != nil || strings.Contains(string(data), "ci*.yml") != (mode == "detailed") || strings.Contains(string(data), "ci-skip.yml") != (mode == "detailed") {
			t.Fatal(string(data), err)
		}
	}
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint)
	if f.data(t) != input || f.writes != 0 {
		t.Fatal("plan or Summary modified state")
	}
}

func TestActionsWorkflowTargetsCreatePreserveResetRepeatProcess(t *testing.T) {
	f := newActionsEventPoliciesFixture(t, "")
	f.payloads = []map[string]any{
		workflowTargetPayload("active", []any{"ci*.yml"}, []any{"ci-skip.yml"}),
		workflowTargetPayload("disabled", nil, nil),
		workflowTargetPayload("disabled", []any{"~ALL"}, []any{}),
		workflowTargetPayload("active", []any{}, []any{"skip*.yml"}),
	}
	f.write(t, workflowTargetInput(false, "active", "{include: ['ci*.yml'], exclude: [ci-skip.yml]}"))
	f.run(t, false, "apply")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "POST "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17")
	if !strings.Contains(f.data(t), "id: 17") || !strings.Contains(f.data(t), "ci*.yml") {
		t.Fatal(f.data(t))
	}
	f.run(t, false, "apply")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17")
	// Omitted target on a selected-ID update retains the acquired remote criteria.
	f.write(t, workflowTargetInput(true, "disabled", ""))
	f.run(t, false, "apply")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17", "GET "+actionsEventPoliciesEndpoint+"/17", "PUT "+actionsEventPoliciesEndpoint+"/17", "GET "+actionsEventPoliciesEndpoint+"/17")
	f.run(t, false, "apply")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17")
	// The explicit all-workflows declaration resets the previous selection.
	f.write(t, workflowTargetInput(true, "disabled", "{include: ['~ALL'], exclude: []}"))
	f.run(t, false, "apply")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17", "GET "+actionsEventPoliciesEndpoint+"/17", "PUT "+actionsEventPoliciesEndpoint+"/17", "GET "+actionsEventPoliciesEndpoint+"/17")
	f.run(t, false, "apply")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17")
	// Exclusion-only intent preserves the API's empty include array.
	f.write(t, workflowTargetInput(true, "active", "{include: [], exclude: ['skip*.yml']}"))
	f.run(t, false, "apply")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17", "GET "+actionsEventPoliciesEndpoint+"/17", "PUT "+actionsEventPoliciesEndpoint+"/17", "GET "+actionsEventPoliciesEndpoint+"/17")
	f.run(t, false, "apply")
	f.expectPolicyCalls(t, "GET "+actionsEventPoliciesEndpoint, "GET "+actionsEventPoliciesEndpoint+"/17")
	if f.writes != 4 {
		t.Fatal(f.writes)
	}
}

func TestActionsWorkflowTargetsExportAndReadbackRecoveryProcess(t *testing.T) {
	t.Run("export", func(t *testing.T) {
		f := newActionsEventPoliciesFixture(t, "")
		f.payloads = []map[string]any{workflowTargetPayload("active", []any{"ci*.yml"}, []any{"skip*.yml"})}
		f.write(t, workflowTargetInput(false, "active", "{include: ['ci*.yml'], exclude: ['skip*.yml']}"))
		f.run(t, false, "apply")
		f.run(t, false, "export")
		data := f.data(t)
		for _, want := range []string{"target_workflows:", "include:", "exclude:", "ci*.yml", "skip*.yml"} {
			if !strings.Contains(data, want) {
				t.Fatal(data)
			}
		}
		f.run(t, false, "validate")
		// Repeat only this family: ordinary export also selects unrelated settings.
		f.write(t, workflowTargetInput(true, "active", "{include: ['ci*.yml'], exclude: ['skip*.yml']}"))
		f.run(t, false, "apply")
		if f.writes != 1 {
			t.Fatal("export/apply changed matching policy", f.writes)
		}
	})
	t.Run("fresh target mismatch", func(t *testing.T) {
		f := newActionsEventPoliciesFixture(t, "")
		f.payloads = []map[string]any{workflowTargetPayload("active", []any{"ci*.yml"}, []any{})}
		input := workflowTargetInput(false, "active", "{include: ['ci*.yml'], exclude: []}")
		f.write(t, input)
		f.failure = "target-readback"
		_, diagnostic := f.run(t, true, "apply")
		for _, want := range []string{"id 17", "readback differs"} {
			if !strings.Contains(diagnostic, want) {
				t.Fatal(diagnostic)
			}
		}
		if f.data(t) != input || f.writes != 1 {
			t.Fatal("failed verification lost file or mutation evidence", f.data(t), f.writes)
		}
	})
}
