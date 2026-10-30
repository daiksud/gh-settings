package actions

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
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

func bypassReportChange(t *testing.T, raw string, remove, companions bool) settings.RulesetChange {
	t.Helper()
	policy := `{"id":7,"name":"<BYPASS_OBJECT>","target":"branch","enforcement":"disabled","conditions":{"ref_name":{"include":[],"exclude":[]}},"bypass_actors":[{"actor_type":"DeployKey","bypass_mode":"always"},{"actor_type":"Team","actor_id":9,"bypass_mode":"always"},{"actor_type":"User","actor_id":9,"bypass_mode":"exempt"}],"rules":[]}`
	var before, after settings.RulesetDeclaration
	if err := json.Unmarshal([]byte(policy), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(policy), &after); err != nil {
		t.Fatal(err)
	}
	if remove {
		*after.BypassActors = append((*after.BypassActors)[:1], (*after.BypassActors)[2:]...)
	}
	id := int64(9)
	records := settings.RulesetNormalizations{BypassModes: []settings.RulesetBypassModeNormalization{
		{ActorType: "Team", ActorID: &id, Normalization: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: raw, Fallback: "always"}},
		{ActorType: "DeployKey", Normalization: settings.ObservationNormalization{Path: settings.RulesetBypassModePath, RawValue: "KEY_MODE", Fallback: "always"}},
	}}
	if companions {
		rules := `[{"type":"code_quality","parameters":{"severity":"errors"}},{"type":"pull_request","parameters":{"allowed_merge_methods":["squash"],"dismiss_stale_reviews_on_push":false,"require_code_owner_review":false,"require_extra_approval_for_unattributed_changes":false,"require_last_push_approval":false,"required_approving_review_count":0,"required_review_thread_resolution":false,"required_reviewers":[]}}]`
		if err := json.Unmarshal([]byte(rules), &before.Rules); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(rules), &after.Rules); err != nil {
			t.Fatal(err)
		}
		records.Enforcement = settings.ObservationNormalization{Path: settings.RulesetEnforcementPath, RawValue: "evaluate", Fallback: "disabled"}
		records.QualitySeverity = settings.ObservationNormalization{Path: settings.RulesetQualitySeverityPath, RawValue: "QUALITY_MODE", Fallback: "errors"}
		records.RequiredApprovals = settings.RulesetRequiredApprovalsNormalization{Path: settings.RulesetRequiredApprovalsPath, RawValue: 11, Fallback: 0}
	}
	change, err := settings.NewRulesetChangeWithNormalizations("update", 7, &before, &after, records)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func writeBypassReport(t *testing.T, change settings.RulesetChange) (PlanReport, []byte, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plan.json")
	var stdout, stderr bytes.Buffer
	status := Run([]string{"plan", path}, &stdout, &stderr, "dev", func() (app.PlanResult, error) {
		return app.PlanResult{Target: repository.Repository{Owner: "owner", Name: "current"}, Comparison: settings.Comparison{Managed: 1, Rulesets: []settings.RulesetChange{change}}}, nil
	})
	data, err := os.ReadFile(path)
	if err != nil || status != 0 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatal(status, err, stdout.String(), stderr.String())
	}
	var report PlanReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	return report, data, path
}

func TestRulesetBypassPrivateReportPreservesOwnedEvidenceAndPrivacy(t *testing.T) {
	for _, companions := range []bool{false, true} {
		for _, remove := range []bool{false, true} {
			for _, raw := range []string{"", "<script>BYPASS_RAW</script>\n::warning::untrusted"} {
				change := bypassReportChange(t, raw, remove, companions)
				report, data, path := writeBypassReport(t, change)
				validated, err := validatePlanReport(report, "owner/current")
				if err != nil || report.Outcome != "changed" || report.Managed != 1 || len(validated.rulesets) != 1 || !reflect.DeepEqual(validated.rulesets[0].Normalizations(), change.Normalizations()) {
					t.Fatal("closed evidence lost", string(data), err)
				}
				var wire map[string]any
				json.Unmarshal(data, &wire)
				entries := wire["rulesets"].([]any)[0].(map[string]any)["bypass_mode_normalizations"].([]any)
				if len(entries) != 2 || entries[0].(map[string]any)["actor_id"] != float64(9) {
					t.Fatal("typed actor identity lost", entries)
				}
				if _, present := entries[1].(map[string]any)["actor_id"]; present {
					t.Fatal("nil identity must be omitted", entries)
				}
				content := renderSummary(validated, RunMetadata{Target: "owner/current"}, []string{"owner", "workflows", ".github/workflows/settings.yml"})
				if !strings.Contains(html.UnescapeString(content), presentation.RulesetLine(change.Display())) || strings.Contains(content, "<script>") || strings.Contains(content, "<BYPASS_OBJECT>") {
					t.Fatal("detailed escaping lost", content)
				}
				private := filepath.Join(t.TempDir(), "private.md")
				t.Setenv("SETTINGS_TARGET", "owner/current")
				var stdout, stderr bytes.Buffer
				status := Run([]string{"summary", path, private, "status-only"}, &stdout, &stderr, "dev", nil)
				privateData, err := os.ReadFile(private)
				if err != nil || status != 0 || stdout.String() != "plan-outcome=changed\n" || stderr.Len() != 0 || strings.Contains(string(privateData), "BYPASS_") || strings.Contains(string(privateData), settings.RulesetBypassModePath) || strings.Contains(string(privateData), "actor_type") {
					t.Fatal("status-only privacy lost", status, string(privateData), err)
				}
				*(*report.Rulesets[0].BypassModeNormalizations)[0].ActorID = 42
				(*report.Rulesets[0].BypassModeNormalizations)[0].Normalization.RawValue = "mutated"
				*(*report.Rulesets[0].Before.BypassActors)[1].ActorID = 88
				if !reflect.DeepEqual(validated.rulesets[0].Normalizations(), change.Normalizations()) || presentation.RulesetLine(validated.rulesets[0]) != presentation.RulesetLine(change.Display()) {
					t.Fatal("validated report aliases input")
				}
			}
		}
	}
}

func TestRulesetBypassReportRejectsForgedAndEmptyEvidence(t *testing.T) {
	for index, mutate := range []func(*PlanReport){
		func(r *PlanReport) { r.Rulesets[0].BypassModeNormalizations = nil },
		func(r *PlanReport) { *r.Rulesets[0].BypassModeNormalizations = []RulesetBypassModeNormalization{} },
		func(r *PlanReport) { (*r.Rulesets[0].BypassModeNormalizations)[1] = RulesetBypassModeNormalization{} },
		func(r *PlanReport) { (*r.Rulesets[0].BypassModeNormalizations)[0].ActorType = "User" },
		func(r *PlanReport) { (*r.Rulesets[0].BypassModeNormalizations)[0].ActorID = nil },
		func(r *PlanReport) { *(*r.Rulesets[0].BypassModeNormalizations)[0].ActorID = 8 },
		func(r *PlanReport) {
			(*r.Rulesets[0].BypassModeNormalizations)[1] = (*r.Rulesets[0].BypassModeNormalizations)[0]
		},
		func(r *PlanReport) { (*r.Rulesets[0].BypassModeNormalizations)[0].Normalization.RawValue = "always" },
		func(r *PlanReport) { (*r.Rulesets[0].BypassModeNormalizations)[0].Normalization.Fallback = "exempt" },
		func(r *PlanReport) { (*r.Rulesets[0].BypassModeNormalizations)[0].Normalization.Path = "other" },
		func(r *PlanReport) { *r.Rulesets[0].After.ID = 8 },
		func(r *PlanReport) { r.Rulesets[0].Before.ID = nil },
		func(r *PlanReport) { r.Managed = 0 },
	} {
		for _, companions := range []bool{false, true} {
			// Other valid evidence must not hide an invalid bypass record. Only bypass-only
			// equality can independently reject an entirely omitted collection.
			if index == 0 && companions {
				continue
			}
			report, _, _ := writeBypassReport(t, bypassReportChange(t, "UNKNOWN", false, companions))
			if _, err := validatePlanReport(report, "owner/current"); err != nil {
				t.Fatal("valid proof rejected", err)
			}
			mutate(&report)
			if got, err := validatePlanReport(report, "owner/current"); err == nil || !reflect.DeepEqual(got, validatedPlanReport{}) {
				t.Fatal("forged or partial evidence accepted", got, err)
			}
		}
	}

	for _, raw := range []string{
		`null`, `[]`, `[null]`, `[{}]`,
		`[{"actor_id":9,"normalization":{"path":"repository.settings.rulesets.bypass_list.bypass_mode","raw_value":"UNKNOWN","fallback":"always"}}]`,
		`[{"actor_type":null,"actor_id":9,"normalization":{"path":"repository.settings.rulesets.bypass_list.bypass_mode","raw_value":"UNKNOWN","fallback":"always"}}]`,
		`[{"actor_type":"Team","actor_id":9}]`,
		`[{"actor_type":"Team","actor_id":9,"normalization":null}]`,
		`[{"actor_type":"Team","actor_id":null,"normalization":{"path":"repository.settings.rulesets.bypass_list.bypass_mode","raw_value":"UNKNOWN","fallback":"always"}}]`,
		`[{"actor_type":"Team","actor_id":"9","normalization":{"path":"repository.settings.rulesets.bypass_list.bypass_mode","raw_value":"UNKNOWN","fallback":"always"}}]`,
		`[{"actor_type":"Team","actor_id":1.5,"normalization":{"path":"repository.settings.rulesets.bypass_list.bypass_mode","raw_value":"UNKNOWN","fallback":"always"}}]`,
		`[{"actor_type":"Team","actor_id":9,"normalization":{}}]`,
		`[{"actor_type":"Team","actor_id":9,"normalization":{"path":"repository.settings.rulesets.bypass_list.bypass_mode","fallback":"always"}}]`,
		`[{"Actor_Type":"Team","actor_id":9,"normalization":{"path":"repository.settings.rulesets.bypass_list.bypass_mode","raw_value":"UNKNOWN","fallback":"always"}}]`,
		`[{"actor_type":"Team","actor_id":9,"extra":true,"normalization":{"path":"repository.settings.rulesets.bypass_list.bypass_mode","raw_value":"UNKNOWN","fallback":"always"}}]`,
	} {
		_, data, _ := writeBypassReport(t, bypassReportChange(t, "UNKNOWN", false, true))
		var wire map[string]any
		json.Unmarshal(data, &wire)
		wire["rulesets"].([]any)[0].(map[string]any)["bypass_mode_normalizations"] = json.RawMessage(raw)
		data, err := json.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		var report PlanReport
		err = json.Unmarshal(data, &report)
		if err == nil {
			_, err = validatePlanReport(report, "owner/current")
		}

		if err == nil {
			t.Fatal("malformed metadata accepted", raw)
		}
	}
}
