package command

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestExportInfoNeverLogsUpstreamSecrets(t *testing.T) {
	secret := "secret-token-and-value"
	httpFailure := &api.HTTPError{
		StatusCode: 403,
		Message:    secret,
		RequestURL: &url.URL{Scheme: "https", Host: "example.com", RawQuery: "token=" + secret},
	}
	for _, cause := range []error{errors.New(secret), fmt.Errorf("wrapped: %w", httpFailure)} {
		var b strings.Builder
		if err := (exportDiagnostics{output: &b}).info("repository.settings.advanced_security", cause); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(b.String(), secret) || !strings.HasPrefix(b.String(), "info: export ") {
			t.Fatal(b.String())
		}
		if !strings.Contains(b.String(), "existing declaration retained where compatible with acquired settings") {
			t.Fatal("diagnostic must qualify retention when acquired state disables dependent intent", b.String())
		}
		var he *api.HTTPError
		if errors.As(cause, &he) && !strings.Contains(b.String(), "HTTP 403") {
			t.Fatal(b.String())
		}
	}
	if err := (exportDiagnostics{output: exportBrokenWriter{}}).info("resource", errors.New(secret)); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}

type exportBrokenWriter struct{}

func (exportBrokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestExportInfoConfirmedDisabledExplainsRemoval(t *testing.T) {
	var b strings.Builder
	if err := (exportDiagnostics{output: &b}).info("codeql", app.ErrExportCodeQLDisabled); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "confirmed disabled; existing configuration removed from export") || strings.Contains(b.String(), "retained") {
		t.Fatal(b.String())
	}
	if err := (exportDiagnostics{output: exportBrokenWriter{}}).info("codeql", app.ErrExportCodeQLDisabled); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
}

func TestExportWarningAdaptersPropagateOutputFailure(t *testing.T) {
	for _, test := range []struct {
		name  string
		write func(exportDiagnostics) error
	}{
		{"warnNormalization", func(w exportDiagnostics) error { return w.warnNormalization(settings.ObservationNormalization{}) }},
		{"warnCommitMessageNormalization", func(w exportDiagnostics) error {
			return w.warnCommitMessageNormalization(settings.CommitMessageNormalization{})
		}},
		{"warnSecurityStatusNormalization", func(w exportDiagnostics) error {
			return w.warnSecurityStatusNormalization(settings.SecurityStatusNormalization{})
		}},
		{"warnRulesetNormalization", func(w exportDiagnostics) error {
			return w.warnRulesetNormalization(7, "fixture", settings.ObservationNormalization{})
		}},
		{"warnRulesetRequiredApprovalsNormalization", func(w exportDiagnostics) error {
			return w.warnRulesetRequiredApprovalsNormalization(7, "fixture", settings.RulesetRequiredApprovalsNormalization{})
		}},
		{"warnRulesetMergeMethodsNormalization", func(w exportDiagnostics) error {
			return w.warnRulesetMergeMethodsNormalization(7, "fixture", settings.RulesetMergeMethodsNormalization{})
		}},
		{"warnRulesetTeamApprovalsNormalization", func(w exportDiagnostics) error {
			return w.warnRulesetTeamApprovalsNormalization(7, "fixture", settings.RulesetTeamApprovalsNormalization{})
		}},
		{"warnRulesetBypassModeNormalization", func(w exportDiagnostics) error {
			return w.warnRulesetBypassModeNormalization(7, "fixture", settings.RulesetBypassModeNormalization{})
		}},
		{"warnEnvironmentDeploymentNormalization", func(w exportDiagnostics) error {
			return w.warnEnvironmentDeploymentNormalization("fixture", settings.EnvironmentDeploymentNormalization{})
		}},
		{"warnEnvironmentRefTypeNormalization", func(w exportDiagnostics) error {
			return w.warnEnvironmentRefTypeNormalization("fixture", settings.EnvironmentPatternState{})
		}},
		{"warnRulesetScanningAlertsNormalization", func(w exportDiagnostics) error {
			return w.warnRulesetScanningAlertsNormalization(7, "fixture", settings.RulesetScanningAlertsNormalization{})
		}},
		{"warnRulesetScanningSecurityAlertsNormalization", func(w exportDiagnostics) error {
			return w.warnRulesetScanningSecurityAlertsNormalization(7, "fixture", settings.RulesetScanningSecurityAlertsNormalization{})
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.write(exportDiagnostics{output: exportBrokenWriter{}}); !errors.Is(err, io.ErrClosedPipe) || !strings.HasPrefix(err.Error(), "write export normalization warning: ") {
				t.Fatal(err)
			}
		})
	}
	var output strings.Builder
	err := (exportDiagnostics{output: &output}).warnNormalization(settings.ObservationNormalization{Path: settings.IssuePermissionsControl.Path(), RawValue: "FUTURE", Fallback: "all_users"})
	want := "Warning: repository.settings.general.features.issue_permissions.creation_allowed_by: observed \"FUTURE\" is outside UI choices; using \"all_users\" (projection only).\n"
	if err != nil || output.String() != want {
		t.Fatal(err, output.String())
	}
}
