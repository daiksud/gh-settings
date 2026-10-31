package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"testing"
)

func TestScanningAlertsExportUsesIndependentSinkAfterCompleteAcquisition(t *testing.T) {
	for _, mode := range []string{"selected", "nil sink", "unrequested", "canonical", "read failure", "unacquired", "UI failure"} {
		t.Run(mode, func(t *testing.T) {
			state := exportEnforcementState(t, 7, "scan", "", false)
			tools := []settings.CodeScanningTool{{Tool: "z tool", AlertsThreshold: "errors", SecurityAlertsThreshold: "all"}, {Tool: "CodeQL", AlertsThreshold: "errors", SecurityAlertsThreshold: "high_or_higher"}}
			*state.Policy.Rules = append(*state.Policy.Rules, settings.RulesetRule{Type: "code_scanning", Parameters: &settings.RuleParameters{CodeScanningTools: &tools}})
			state.ScanningAlertsNormalizations = []settings.RulesetScanningAlertsNormalization{{Tool: "z tool", Path: settings.RulesetScanningAlertsPath, RawValue: "future", Fallback: "errors"}, {Tool: "CodeQL", Path: settings.RulesetScanningAlertsPath, RawValue: "", Fallback: "errors"}}
			if mode == "canonical" {
				state.ScanningAlertsNormalizations = nil
			}
			if mode == "UI failure" {
				always := "always"
				state.Policy.BypassActors = &[]settings.BypassActor{{ActorType: "OrganizationAdmin", BypassMode: &always}}
			}
			observed, err := settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
			if err != nil {
				t.Fatal(err)
			}
			original, _ := observed.Values()
			cause := errors.New("scanning acquisition failed")
			reads, saves := 0, 0
			var warnings []string
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reader.Rulesets = func(repository.Repository) (settings.RulesetObservation, error) {
				reads++
				if mode == "read failure" {
					return observed, cause
				}
				if mode == "unacquired" {
					return settings.RulesetObservation{}, nil
				}
				return observed, nil
			}
			checkout := ExportCheckout{Repository: checkoutRepository, IncludeRulesets: mode != "unrequested", PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					ui, err := github.UIFromExportDeclaration(d)
					if err != nil {
						return err
					}
					saves++
					if mode != "unrequested" {
						tools := *(*ui.Settings.Rulesets)[0].Rules.RequireCodeScanningResults.CodeScanningTools
						if len(tools) != 2 || tools[0].ToolName != "CodeQL" || tools[1].ToolName != "z tool" || tools[0].Alerts != "errors" || tools[1].Alerts != "errors" || tools[0].SecurityAlerts != "high_or_higher" || tools[1].SecurityAlerts != "all" {
							t.Fatal(tools)
						}
					}
					return nil
				}, nil
			}}
			if mode != "nil sink" {
				checkout.RulesetScanningAlertsNormalizationWarning = func(id int64, name string, n settings.RulesetScanningAlertsNormalization) {
					if id != 7 || name != "scan" {
						t.Fatal(id, name)
					}
					warnings = append(warnings, n.Tool+":"+n.RawValue)
					n.Tool = "corrupt"
				}
			}
			err = Export(checkout, reader)
			failure := mode == "read failure" || mode == "unacquired" || mode == "UI failure"
			if (err != nil) != failure || failure && saves != 0 || !failure && saves != 1 {
				t.Fatal(err, saves)
			}
			if mode == "unrequested" && reads != 0 {
				t.Fatal("unrequested read")
			}
			var want []string
			if mode == "selected" || mode == "UI failure" {
				want = []string{"CodeQL:", "z tool:future"}
			}
			if !reflect.DeepEqual(warnings, want) {
				t.Fatal(warnings, want)
			}
			after, _ := observed.Values()
			if !reflect.DeepEqual(original, after) {
				t.Fatal("aliased observation")
			}
		})
	}
}
