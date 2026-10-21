package githubactions

import (
	"fmt"
	"strings"
	"testing"
)

func TestPrivateCompositeOutcomesAndCleanup(t *testing.T) {
	for _, outcome := range []string{"changed", "no-difference", "unmanaged"} {
		t.Run(outcome, func(t *testing.T) {
			r := runComposite(t, "plan", "fixture-token", map[string]string{"inputs.plan-reporting": "status-only", "MOCK_PLAN_OUTCOME": outcome, "MOCK_DESCRIPTION_BEFORE": "CANARY-hidden-value\n::error::injection", "github.actor": "CANARY-private-actor"})
			if r.status != 0 || r.outputs["plan-outcome"] != outcome || len(r.outputs) != 5 || r.api != "" || len(r.remainingReports) != 0 || r.injected {
				t.Fatalf("private result: %+v", r)
			}
			public := r.log + r.summary + fmt.Sprint(r.outputs)
			for _, secret := range []string{"CANARY", "fixture-token", "before", "desired", "repository.description"} {
				if strings.Contains(public, secret) {
					t.Fatalf("private disclosure %q: %s", secret, public)
				}
			}
			if !strings.Contains(r.summary, "Outcome: "+outcome) {
				t.Fatal(r.summary)
			}
		})
	}
}

func TestPrivateCompositeFailures(t *testing.T) {
	for _, test := range []struct {
		name  string
		extra map[string]string
	}{
		{"validation", map[string]string{"MOCK_VALIDATE_STATUS": "37", "MOCK_ERROR": "CANARY-validation"}},
		{"API", map[string]string{"MOCK_REMOTE_STATUS": "37", "MOCK_ERROR": "CANARY-API"}},
		{"missing report", map[string]string{"MOCK_REPORT_MODE": "missing"}},
		{"malformed report", map[string]string{"MOCK_REPORT_MODE": "malformed"}},
		{"summary write", map[string]string{"SUMMARY_DIRECTORY": "1"}},
		{"status handoff", map[string]string{"STATUS_DIRECTORY": "1"}},
		{"old package", map[string]string{"MOCK_OLD_CLI": "1"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.extra["inputs.plan-reporting"] = "status-only"
			r := runComposite(t, "plan", "fixture-token", test.extra)
			if r.status == 0 || len(r.outputs) != 0 || len(r.remainingReports) != 0 || r.api != "" {
				t.Fatalf("private failure: %+v", r)
			}
			if strings.Contains(r.log+r.summary, "CANARY") || strings.Contains(r.log+r.summary, "fixture-token") {
				t.Fatal("private failure leaked", r.log, r.summary)
			}
			if test.name == "old package" && r.calls != "" {
				t.Fatal("unsupported package reached remote work")
			}
		})
	}
}

func TestPrivateCompositeInputBoundary(t *testing.T) {
	for _, command := range []string{"validate", "apply", "plan"} {
		mode := "status-only"
		if command == "plan" {
			mode = "CANARY-$(touch injected)"
		}
		r := runComposite(t, command, "fixture-token", map[string]string{"inputs.plan-reporting": mode})
		if r.status == 0 || r.downloaded || r.calls != "" || r.injected || strings.Contains(r.log, "CANARY") {
			t.Fatalf("reporting input boundary: %+v", r)
		}
	}
	f := newCompositeFixture(t)
	f.git("remote", "set-url", "origin", "https://github.com/CANARY-invalid%0Atarget/config")
	r := f.run(t, "plan", "fixture-token", map[string]string{"inputs.plan-reporting": "status-only", "MOCK_REAL_METADATA": "1"})
	if r.status == 0 || r.calls != "" || strings.Contains(r.log, "CANARY") {
		t.Fatalf("metadata leaked: %+v", r)
	}
}
