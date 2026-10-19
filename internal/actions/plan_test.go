package actions_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func TestPlanNoDifferenceReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	calls := 0
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		calls++
		return app.PlanResult{Target: repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1}}, nil
	})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report actions.PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "no-difference" || report.Target != "owner/current" || report.Managed != 1 || len(report.Changes) != 0 {
		t.Fatalf("report=%+v", report)
	}
	if status != 0 || stdout.Len() != 0 || stderr.Len() != 0 || calls != 1 {
		t.Fatalf("status=%d stdout=%q stderr=%q calls=%d", status, stdout.String(), stderr.String(), calls)
	}
}

func TestPlanReportStorageFailure(t *testing.T) {
	root := t.TempDir()
	for _, test := range []struct {
		name, path string
		planErr    error
	}{
		{"empty path", "", nil},
		{"missing parent", filepath.Join(root, "missing", "plan.json"), nil},
		{"directory destination", root, nil},
		{"planning and storage failure", root, &app.PlanError{Stage: "invalid-config", Err: errors.New("invalid configuration")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			calls := 0
			status := actions.Run([]string{"plan", test.path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
				calls++
				return app.PlanResult{}, test.planErr
			})
			if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "write plan report:") || calls != 1 {
				t.Fatalf("status=%d stdout=%q stderr=%q calls=%d", status, stdout.String(), stderr.String(), calls)
			}
		})
	}
}

func TestPlanTargetFailureReport(t *testing.T) {
	metadataCheckout(t)
	git(t, "remote", "remove", "origin")
	if err := os.Mkdir(".github", 0700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(".github", "settings.yml")
	input := "schemaVersion: 1\nsettings:\n  repository:\n    has_issues: false\n"
	if err := os.WriteFile(config, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	calls := 0
	status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.Plan(workspace.NewCheckout(".").Declaration, app.Reader{Repository: func(repository.Repository) (settings.RepositoryState, error) {
			calls++
			return settings.RepositoryState{}, nil
		}})
	})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report map[string]any
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report["outcome"] != "target-failure" || report["diagnostic"] != "repository target resolution failed" {
		t.Fatalf("report=%v", report)
	}
	if _, present := report["target"]; present {
		t.Fatalf("unexpected unresolved target: %v", report)
	}
	if status != 1 || stdout.Len() != 0 || stderr.Len() == 0 || calls != 0 {
		t.Fatalf("status=%d stdout=%q stderr=%q calls=%d", status, stdout.String(), stderr.String(), calls)
	}
	if data, err := os.ReadFile(config); err != nil || string(data) != input {
		t.Fatalf("configuration changed: %q error=%v", data, err)
	}
}

func comparisonChange(t *testing.T, path string, before, after any) settings.Change {
	t.Helper()
	field, err := settings.ParseFieldID(path)
	if err != nil {
		t.Fatal(err)
	}
	var change settings.Change
	switch before := before.(type) {
	case bool:
		change, err = settings.NewBoolChange(field, before, after.(bool))
	case string:
		change, err = settings.NewStringChange(field, before, after.(string))
	case []string:
		if field != settings.TopicsField() {
			t.Fatal("topic values require the topics field")
		}
		change, err = settings.NewTopicsChange(before, after.([]string))
	default:
		t.Fatalf("unsupported comparison fixture type %T", before)
	}
	if err != nil {
		t.Fatal(err)
	}
	return change
}
