package actions_test

import (
	"bytes"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

func environmentRefTypeReportChange(t *testing.T, collision, clear, partial bool) settings.EnvironmentChange {
	t.Helper()
	zero, off, reviewers := 0, false, []settings.EnvironmentReviewer{}
	before := settings.EnvironmentDeclaration{Name: "<REF_OBJECT>", WaitTimer: &zero, PreventSelfReview: &off, CanAdminsBypass: &off, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true, DeploymentBranchPolicy: &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}}
	states := []settings.EnvironmentPatternState{
		{ID: 7, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "<PATTERN>"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "", Fallback: "branch"}},
		{ID: 8, Pattern: settings.EnvironmentPattern{Type: "tag", Name: "CANONICAL_COMPANION"}},
	}
	if collision {
		states = append(states, settings.EnvironmentPatternState{ID: 9, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "<PATTERN>"}})
	} else {
		patterns, err := settings.EffectiveEnvironmentPatterns(states)
		if err != nil {
			t.Fatal(err)
		}
		before.DeploymentBranchPatterns = &patterns
	}
	after := before
	if partial {
		after = settings.EnvironmentDeclaration{Name: before.Name}
	}
	desired := []settings.EnvironmentPattern{{Type: "tag", Name: "<PATTERN>"}}
	if clear {
		desired = []settings.EnvironmentPattern{}
	}
	after.DeploymentBranchPatterns = &desired
	change, err := settings.NewEnvironmentChangeWithPatternNormalizations("update", &before, &after, states)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func TestEnvironmentRefTypePrivateReportRetainsFullListAndPartialRequest(t *testing.T) {
	for _, collision := range []bool{false, true} {
		for _, clear := range []bool{false, true} {
			for _, partial := range []bool{false, true} {
				change := environmentRefTypeReportChange(t, collision, clear, partial)
				path := filepath.Join(t.TempDir(), "plan.json")
				var stdout, stderr bytes.Buffer
				status := actions.Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
					return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Environments: []settings.EnvironmentChange{change}}}, nil
				})
				data, err := os.ReadFile(path)
				if err != nil || status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
					t.Fatal(status, err, stdout.String(), stderr.String())
				}
				var report actions.PlanReport
				if err := json.Unmarshal(data, &report); err != nil {
					t.Fatal(err)
				}
				entry := report.Environments[0]
				if entry.PatternObservations == nil || len(*entry.PatternObservations) != len(change.PatternObservations()) || entry.DeploymentModeNormalization != nil || !reflect.DeepEqual(entry.After, change.After()) || !reflect.DeepEqual(entry.Before, change.Before()) {
					t.Fatal("full list or exact request lost", string(data))
				}
				for i, want := range change.PatternObservations() {
					got := (*entry.PatternObservations)[i]
					if got.ID != want.ID || got.Pattern != want.Pattern || (got.RefTypeNormalization == nil) != (want.RefTypeNormalization == (settings.ObservationNormalization{})) || got.RefTypeNormalization != nil && *got.RefTypeNormalization != want.RefTypeNormalization {
						t.Fatal("per-ID raw evidence lost", got, want)
					}
				}
				content, err := actions.RenderSummary(report, summaryMetadata())
				if err != nil || !strings.Contains(html.UnescapeString(content), presentation.EnvironmentLine(change)) || strings.Contains(content, "<REF_OBJECT>") || strings.Contains(content, "<PATTERN>") {
					t.Fatal("detailed summary or HTML boundary", content, err)
				}
				report.Managed = 0
				if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
					t.Fatal("undercount accepted")
				}
				private := filepath.Join(t.TempDir(), "status.md")
				t.Setenv("SETTINGS_TARGET", "owner/current")
				stdout.Reset()
				stderr.Reset()
				status = actions.Run([]string{"summary", path, private, "status-only"}, &stdout, &stderr, "dev", nil)
				data, err = os.ReadFile(private)
				if err != nil || status != 0 || stdout.String() != "plan-outcome=changed\n" || stderr.Len() != 0 || strings.Contains(string(data), "REF_OBJECT") || strings.Contains(string(data), "PATTERN") || strings.Contains(string(data), "CANONICAL_COMPANION") || strings.Contains(string(data), settings.EnvironmentRefTypePath) {
					t.Fatal("status privacy", status, string(data), err)
				}
			}
		}
	}
}

func refTypeReportEntry(change settings.EnvironmentChange) actions.EnvironmentPlanChange {
	states := []actions.EnvironmentPatternObservation{}
	for _, state := range change.PatternObservations() {
		entry := actions.EnvironmentPatternObservation{ID: state.ID, Pattern: state.Pattern}
		if state.RefTypeNormalization != (settings.ObservationNormalization{}) {
			record := state.RefTypeNormalization
			entry.RefTypeNormalization = &record
		}
		states = append(states, entry)
	}
	return actions.EnvironmentPlanChange{Operation: change.Operation(), Before: change.Before(), After: change.After(), PatternObservations: &states}
}

func TestEnvironmentRefTypePrivateReportRejectsInconsistentOrUnownedEvidence(t *testing.T) {
	for _, mutate := range []func(*actions.EnvironmentPlanChange){
		func(e *actions.EnvironmentPlanChange) {
			empty := []actions.EnvironmentPatternObservation{}
			e.PatternObservations = &empty
		},
		func(e *actions.EnvironmentPlanChange) { (*e.PatternObservations)[0].RefTypeNormalization = nil },
		func(e *actions.EnvironmentPlanChange) {
			(*e.PatternObservations)[1].RefTypeNormalization = &settings.ObservationNormalization{}
		},
		func(e *actions.EnvironmentPlanChange) {
			(*e.PatternObservations)[0].RefTypeNormalization.Path = "wrong"
		},
		func(e *actions.EnvironmentPlanChange) {
			(*e.PatternObservations)[0].RefTypeNormalization.RawValue = "branch"
		},
		func(e *actions.EnvironmentPlanChange) { (*e.PatternObservations)[0].ID = 0 },
		func(e *actions.EnvironmentPlanChange) { (*e.PatternObservations)[1].ID = 7 },
		func(e *actions.EnvironmentPlanChange) { *e.PatternObservations = (*e.PatternObservations)[:1] },
		func(e *actions.EnvironmentPlanChange) { e.Before.DeploymentBranchPatterns = nil },
		func(e *actions.EnvironmentPlanChange) { e.After.DeploymentBranchPatterns = nil },
		func(e *actions.EnvironmentPlanChange) { e.After.Name = "OTHER" },
		func(e *actions.EnvironmentPlanChange) { e.Operation = "create"; e.Before = nil },
		func(e *actions.EnvironmentPlanChange) {
			e.DeploymentModeNormalization = &settings.EnvironmentDeploymentNormalization{}
		},
		func(e *actions.EnvironmentPlanChange) {
			e.DeploymentModeNormalization = &settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: settings.EnvironmentBranchPolicy{}, Fallback: "no_restriction"}
		},
	} {
		entry := refTypeReportEntry(environmentRefTypeReportChange(t, false, false, false))
		report := actions.PlanReport{Outcome: "changed", Target: "owner/current", Managed: 1, Changes: []actions.PlanChange{}, Environments: []actions.EnvironmentPlanChange{entry}}
		if _, err := actions.RenderSummary(report, summaryMetadata()); err != nil {
			t.Fatal("valid report rejected", err)
		}
		mutate(&report.Environments[0])
		if _, err := actions.RenderSummary(report, summaryMetadata()); err == nil {
			t.Fatal("invalid pattern evidence accepted", report.Environments[0])
		}
	}
}

func TestEnvironmentRefTypePrivateReportRejectsMalformedNestedShape(t *testing.T) {
	for _, raw := range []string{
		`null`, `[null]`, `[{"id":7,"pattern":null}]`, `[{"id":7,"pattern":{"type":"branch","name":"main"},"extra":true}]`,
		`[{"id":7,"pattern":{"type":"branch","name":"main"},"ref_type_normalization":null}]`,
		`[{"id":7,"pattern":{"type":"branch","name":"main"},"ref_type_normalization":{}}]`,
		`[{"id":7,"pattern":{"type":"branch","name":"main"},"ref_type_normalization":{"path":"x","raw_value":null,"fallback":"branch"}}]`,
		`[{"ID":7,"pattern":{"type":"branch","name":"main"}}]`,
	} {
		var entry actions.EnvironmentPlanChange
		if err := json.Unmarshal([]byte(`{"operation":"update","pattern_observations":`+raw+`}`), &entry); err == nil {
			t.Fatal("malformed pattern shape decoded", raw)
		}
	}
}
