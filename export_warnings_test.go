package main

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestExportWarningAdaptersPropagateOutputFailure(t *testing.T) {
	for _, test := range []struct {
		name  string
		write func(exportWarnings) error
	}{
		{"warnNormalization", func(w exportWarnings) error { return w.warnNormalization(settings.ObservationNormalization{}) }},
		{"warnCommitMessageNormalization", func(w exportWarnings) error {
			return w.warnCommitMessageNormalization(settings.CommitMessageNormalization{})
		}},
		{"warnSecurityStatusNormalization", func(w exportWarnings) error {
			return w.warnSecurityStatusNormalization(settings.SecurityStatusNormalization{})
		}},
		{"warnRulesetNormalization", func(w exportWarnings) error {
			return w.warnRulesetNormalization(7, "fixture", settings.ObservationNormalization{})
		}},
		{"warnRulesetRequiredApprovalsNormalization", func(w exportWarnings) error {
			return w.warnRulesetRequiredApprovalsNormalization(7, "fixture", settings.RulesetRequiredApprovalsNormalization{})
		}},
		{"warnRulesetMergeMethodsNormalization", func(w exportWarnings) error {
			return w.warnRulesetMergeMethodsNormalization(7, "fixture", settings.RulesetMergeMethodsNormalization{})
		}},
		{"warnRulesetTeamApprovalsNormalization", func(w exportWarnings) error {
			return w.warnRulesetTeamApprovalsNormalization(7, "fixture", settings.RulesetTeamApprovalsNormalization{})
		}},
		{"warnRulesetBypassModeNormalization", func(w exportWarnings) error {
			return w.warnRulesetBypassModeNormalization(7, "fixture", settings.RulesetBypassModeNormalization{})
		}},
		{"warnEnvironmentDeploymentNormalization", func(w exportWarnings) error {
			return w.warnEnvironmentDeploymentNormalization("fixture", settings.EnvironmentDeploymentNormalization{})
		}},
		{"warnEnvironmentRefTypeNormalization", func(w exportWarnings) error {
			return w.warnEnvironmentRefTypeNormalization("fixture", settings.EnvironmentPatternState{})
		}},
		{"warnRulesetScanningAlertsNormalization", func(w exportWarnings) error {
			return w.warnRulesetScanningAlertsNormalization(7, "fixture", settings.RulesetScanningAlertsNormalization{})
		}},
		{"warnRulesetScanningSecurityAlertsNormalization", func(w exportWarnings) error {
			return w.warnRulesetScanningSecurityAlertsNormalization(7, "fixture", settings.RulesetScanningSecurityAlertsNormalization{})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.write(exportWarnings{output: exportBrokenWriter{}}); !errors.Is(err, io.ErrClosedPipe) || !strings.HasPrefix(err.Error(), "write export normalization warning: ") {
				t.Fatal(err)
			}
		})
	}
	var output strings.Builder
	err := (exportWarnings{output: &output}).warnNormalization(settings.ObservationNormalization{Path: settings.IssuePermissionsControl.Path(), RawValue: "FUTURE", Fallback: "all_users"})
	want := "Warning: repository.settings.general.features.issue_permissions.creation_allowed_by: observed \"FUTURE\" is outside UI choices; using \"all_users\" (projection only).\n"
	if err != nil || output.String() != want {
		t.Fatal(err, output.String())
	}
}
