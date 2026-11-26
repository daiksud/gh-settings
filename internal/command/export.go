package command

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

func exportCheckout(checkout workspace.Checkout, reader app.Reader, output io.Writer) error {
	diagnostics := exportDiagnostics{output: output}
	target, err := checkout.Repository()
	if err != nil {
		return err
	}
	existing, save, err := prepareCheckoutSave(checkout, output)
	if err != nil {
		return err
	}
	return app.Export(app.ExportCheckout{
		Check:                              github.CheckExportAcquisition,
		Existing:                           existing,
		Info:                               diagnostics.info,
		Repository:                         func() (repository.Repository, error) { return target, nil },
		PrepareSave:                        func() (func(settings.Declaration) error, error) { return save, nil },
		NormalizationWarning:               diagnostics.warnNormalization,
		CommitMessageNormalizationWarning:  diagnostics.warnCommitMessageNormalization,
		SecurityStatusNormalizationWarning: diagnostics.warnSecurityStatusNormalization,
		RulesetNormalizationWarning:        diagnostics.warnRulesetNormalization,
		RulesetRequiredApprovalsNormalizationWarning:      diagnostics.warnRulesetRequiredApprovalsNormalization,
		RulesetMergeMethodsNormalizationWarning:           diagnostics.warnRulesetMergeMethodsNormalization,
		RulesetTeamApprovalsNormalizationWarning:          diagnostics.warnRulesetTeamApprovalsNormalization,
		RulesetScanningAlertsNormalizationWarning:         diagnostics.warnRulesetScanningAlertsNormalization,
		RulesetScanningSecurityAlertsNormalizationWarning: diagnostics.warnRulesetScanningSecurityAlertsNormalization,
		RulesetBypassModeNormalizationWarning:             diagnostics.warnRulesetBypassModeNormalization,
		EnvironmentDeploymentNormalizationWarning:         diagnostics.warnEnvironmentDeploymentNormalization,
		EnvironmentRefTypeNormalizationWarning:            diagnostics.warnEnvironmentRefTypeNormalization,
	}, reader)
}

// Export requires successful diagnostic delivery before saving a projection.
// Apply retains its separate warning callbacks and mutation failure contract.
type exportDiagnostics struct{ output io.Writer }

// Never render upstream messages, response bodies, URLs or supplied values.
func (w exportDiagnostics) info(path string, cause error) error {
	reason := "observation unavailable, incomplete or unsupported"
	var httpError *api.HTTPError
	var networkError net.Error
	if errors.Is(cause, app.ErrExportCodeQLDisabled) {
		_, err := fmt.Fprintf(w.output, "info: export %s: CodeQL default setup is confirmed disabled; existing configuration removed from export\n", path)
		if err != nil {
			return fmt.Errorf("write export info: %w", err)
		}
		return nil
	} else if errors.As(cause, &httpError) {
		reason = fmt.Sprintf("HTTP %d", httpError.StatusCode)
	} else if errors.As(cause, &networkError) {
		if networkError.Timeout() {
			reason = "request timed out"
		} else {
			reason = "network request failed"
		}
	}
	_, err := fmt.Fprintf(w.output, "info: export %s: %s; omitted on first export, existing declaration retained where compatible with acquired settings\n", path, reason)
	if err != nil {
		return fmt.Errorf("write export info: %w", err)
	}
	return nil
}

func (w exportDiagnostics) writeWarning(line string) error {
	if _, err := fmt.Fprintln(w.output, line); err != nil {
		return fmt.Errorf("write export normalization warning: %w", err)
	}
	return nil
}

func (w exportDiagnostics) warnNormalization(normalization settings.ObservationNormalization) error {
	return w.writeWarning(presentation.NormalizationLine(normalization))
}

func (w exportDiagnostics) warnCommitMessageNormalization(normalization settings.CommitMessageNormalization) error {
	return w.writeWarning(presentation.CommitMessageNormalizationLine(normalization))
}

func (w exportDiagnostics) warnSecurityStatusNormalization(record settings.SecurityStatusNormalization) error {
	return w.writeWarning(presentation.SecurityStatusNormalizationLine(record))
}

func (w exportDiagnostics) warnRulesetNormalization(id int64, name string, record settings.ObservationNormalization) error {
	return w.writeWarning(presentation.RulesetNormalizationLine(id, name, record))
}

func (w exportDiagnostics) warnRulesetRequiredApprovalsNormalization(id int64, name string, record settings.RulesetRequiredApprovalsNormalization) error {
	return w.writeWarning(presentation.RulesetRequiredApprovalsNormalizationLine(id, name, record))
}

func (w exportDiagnostics) warnRulesetMergeMethodsNormalization(id int64, name string, record settings.RulesetMergeMethodsNormalization) error {
	return w.writeWarning(presentation.RulesetMergeMethodsNormalizationLine(id, name, record))
}

func (w exportDiagnostics) warnRulesetTeamApprovalsNormalization(id int64, name string, record settings.RulesetTeamApprovalsNormalization) error {
	return w.writeWarning(presentation.RulesetTeamApprovalsNormalizationLine(id, name, record))
}

func (w exportDiagnostics) warnRulesetBypassModeNormalization(id int64, name string, record settings.RulesetBypassModeNormalization) error {
	return w.writeWarning(presentation.RulesetBypassNormalizationLine(id, name, record))
}

func (w exportDiagnostics) warnEnvironmentDeploymentNormalization(name string, record settings.EnvironmentDeploymentNormalization) error {
	return w.writeWarning(presentation.EnvironmentDeploymentNormalizationLine(name, record))
}

func (w exportDiagnostics) warnEnvironmentRefTypeNormalization(name string, pattern settings.EnvironmentPatternState) error {
	return w.writeWarning(presentation.EnvironmentRefTypeNormalizationLine(name, pattern))
}

func (w exportDiagnostics) warnRulesetScanningAlertsNormalization(id int64, name string, record settings.RulesetScanningAlertsNormalization) error {
	return w.writeWarning(presentation.RulesetScanningAlertsNormalizationLine(id, name, record))
}

func (w exportDiagnostics) warnRulesetScanningSecurityAlertsNormalization(id int64, name string, record settings.RulesetScanningSecurityAlertsNormalization) error {
	return w.writeWarning(presentation.RulesetScanningSecurityAlertsNormalizationLine(id, name, record))
}
