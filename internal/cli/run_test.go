package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--version"}, &stdout, &stderr, "fixture-version", Operations{Export: func() error {
		t.Fatal("version command must not call GitHub")
		return nil
	}})
	if code != 0 || stdout.String() != "gh-settings fixture-version\n" || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestHelpAndVersionOutputFailure(t *testing.T) {
	for _, test := range []struct {
		arg        string
		wantStderr string
	}{
		{"--help", "gh settings: write help: output unavailable\n"},
		{"-h", "gh settings: write help: output unavailable\n"},
		{"--version", "gh settings: write version: output unavailable\n"},
		{"-v", "gh settings: write version: output unavailable\n"},
	} {
		t.Run(test.arg, func(t *testing.T) {
			var stderr bytes.Buffer
			status := Run([]string{test.arg}, brokenWriter{}, &stderr, "fixture-version", Operations{
				Export: func() error {
					t.Fatal("help/version output failure invoked export")
					return nil
				},
				Validate: func() (settings.Declaration, error) {
					t.Fatal("help/version output failure invoked validate")
					return settings.Declaration{}, nil
				},
			})
			if status != 1 || stderr.String() != test.wantStderr {
				t.Fatalf("status=%d stderr=%q, want %q", status, stderr.String(), test.wantStderr)
			}
		})
	}
}

func TestRunSelectsOperation(t *testing.T) {
	operationError := errors.New("operation failed")
	for _, test := range []struct {
		name              string
		args              []string
		operationError    error
		wantExportCalls   int
		wantValidateCalls int
		wantPlanCalls     int
		wantApplyCalls    int
		wantStatus        int
	}{
		{"export", []string{"export"}, nil, 1, 0, 0, 0, 0},
		{"validate", []string{"validate"}, nil, 0, 1, 0, 0, 0},
		{"export failure", []string{"export"}, operationError, 1, 0, 0, 0, 1},
		{"validate failure", []string{"validate"}, operationError, 0, 1, 0, 0, 1},
		{"help", []string{"--help"}, nil, 0, 0, 0, 0, 0},
		{"short help", []string{"-h"}, nil, 0, 0, 0, 0, 0},
		{"version", []string{"--version"}, nil, 0, 0, 0, 0, 0},
		{"short version", []string{"-v"}, nil, 0, 0, 0, 0, 0},
		{"no arguments", nil, nil, 0, 0, 0, 0, 1},
		{"apply", []string{"apply"}, nil, 0, 0, 0, 1, 0},
		{"apply failure", []string{"apply"}, operationError, 0, 0, 0, 1, 1},
		{"apply arguments", []string{"apply", "--plan", "saved"}, nil, 0, 0, 0, 0, 1},
		{"plan", []string{"plan"}, nil, 0, 0, 1, 0, 0},
		{"plan failure", []string{"plan"}, operationError, 0, 0, 1, 0, 1},
		{"plan arguments", []string{"plan", "--json"}, nil, 0, 0, 0, 0, 1},
		{"export arguments", []string{"export", "--repo", "owner/other"}, nil, 0, 0, 0, 0, 1},
		{"validate arguments", []string{"validate", "--repo", "owner/other"}, nil, 0, 0, 0, 0, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exportCalls, validateCalls, planCalls, applyCalls := 0, 0, 0, 0
			status := Run(test.args, &stdout, &stderr, "fixture-version", Operations{
				Export: func() error {
					exportCalls++
					return test.operationError
				},
				Plan: func() (app.PlanResult, error) {
					planCalls++
					return app.PlanResult{Comparison: settings.Comparison{Managed: 1}}, test.operationError
				},
				Apply: func() (app.ApplyResult, error) {
					applyCalls++
					return app.ApplyResult{Outcome: app.ApplyCompleted}, test.operationError
				},
				Validate: func() (settings.Declaration, error) {
					validateCalls++
					return settings.Declaration{}, test.operationError
				},
			})
			if exportCalls != test.wantExportCalls || validateCalls != test.wantValidateCalls || planCalls != test.wantPlanCalls || applyCalls != test.wantApplyCalls {
				t.Fatalf("export calls=%d, validate calls=%d, plan calls=%d, apply calls=%d", exportCalls, validateCalls, planCalls, applyCalls)
			}
			if status != test.wantStatus {
				t.Fatalf("status=%d stderr=%q", status, stderr.String())
			}
			if test.operationError != nil && (stdout.Len() != 0 || stderr.String() != "gh settings: operation failed\n") {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if test.name == "validate" && (stdout.Len() != 0 || stderr.Len() != 0) {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if test.name == "plan" && (stdout.String() != "No differences for managed repository settings.\n" || stderr.Len() != 0) {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if test.name == "apply" && (stdout.String() != "Applied repository settings.\n" || stderr.Len() != 0) {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if test.name == "export" && (stdout.Len() != 0 || stderr.Len() != 0) {
				t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestUnsupportedCommand(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown"}, {"apply", "extra"}, {"plan", "--json"}, {"export", "--repo", "owner/other"}, {"validate", "--repo", "owner/other"}} {
		var stdout, stderr bytes.Buffer
		code := Run(args, &stdout, &stderr, "fixture-version", Operations{Export: func() error {
			t.Fatal("unsupported command invoked API operation")
			return errors.New("unexpected API call")
		}})
		if code != 1 || stdout.Len() != 0 || stderr.String() != "gh settings: unsupported command or arguments; usage: gh settings <export|validate|plan|apply>\n" {
			t.Fatalf("args=%v status=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestExportDoesNotWriteStdout(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"export"}, brokenWriter{}, &stderr, "fixture-version", Operations{Export: func() error { return nil }})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("status=%d stderr=%q", code, stderr.String())
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }

func TestHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := Run([]string{"--help"}, &stdout, &stderr, "fixture-version", Operations{Export: func() error { t.Fatal("help invoked API operation"); return nil }})
	if status != 0 || !strings.Contains(stdout.String(), "as YAML to .github/settings.yml at the checkout root") || stderr.Len() != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestPlanOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	code := Run([]string{"plan"}, brokenWriter{}, &stderr, "fixture-version", Operations{
		Plan: func() (app.PlanResult, error) { return app.PlanResult{}, nil },
	})
	if code != 1 || stderr.String() != "gh settings: write output: output unavailable\n" {
		t.Fatalf("status=%d stderr=%q", code, stderr.String())
	}
}

func TestApplyOutputFailure(t *testing.T) {
	for _, outcome := range []app.ApplyOutcome{app.ApplyCompleted, app.ApplyMatchingLabels, app.ApplyUnmanaged} {
		var stderr bytes.Buffer
		calls := 0
		code := Run([]string{"apply"}, brokenWriter{}, &stderr, "fixture-version", Operations{
			Apply: func() (app.ApplyResult, error) {
				calls++
				return app.ApplyResult{Outcome: outcome}, nil
			},
		})
		if code != 1 || calls != 1 || stderr.String() != "gh settings: write output: output unavailable\n" {
			t.Fatalf("status=%d calls=%d stderr=%q", code, calls, stderr.String())
		}
	}
}

func TestApplyResultMessages(t *testing.T) {
	for _, test := range []struct {
		name    string
		outcome app.ApplyOutcome
		message string
	}{
		{"unmanaged", app.ApplyUnmanaged, "No managed repository settings.\n"},
		{"matching labels", app.ApplyMatchingLabels, "Applied repository settings.\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			calls := 0
			status := Run([]string{"apply"}, &stdout, &stderr, "fixture-version", Operations{
				Apply: func() (app.ApplyResult, error) { calls++; return app.ApplyResult{Outcome: test.outcome}, nil },
			})
			if status != 0 || calls != 1 || stdout.String() != test.message || stderr.Len() != 0 {
				t.Fatalf("status=%d calls=%d stdout=%q stderr=%q", status, calls, stdout.String(), stderr.String())
			}
		})
	}
}
