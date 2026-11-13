package main

import (
	"fmt"
	"io"

	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Export requires successful diagnostic delivery before saving a projection.
// Apply retains its separate warning callbacks and mutation failure contract.
type exportWarnings struct{ output io.Writer }

func (w exportWarnings) write(line string) error {
	if _, err := fmt.Fprintln(w.output, line); err != nil {
		return fmt.Errorf("write export normalization warning: %w", err)
	}
	return nil
}

func (w exportWarnings) warnNormalization(normalization settings.ObservationNormalization) error {
	return w.write(presentation.NormalizationLine(normalization))
}

func (w exportWarnings) warnCommitMessageNormalization(normalization settings.CommitMessageNormalization) error {
	return w.write(presentation.CommitMessageNormalizationLine(normalization))
}

func (w exportWarnings) warnSecurityStatusNormalization(record settings.SecurityStatusNormalization) error {
	return w.write(presentation.SecurityStatusNormalizationLine(record))
}

func (w exportWarnings) warnRulesetNormalization(id int64, name string, record settings.ObservationNormalization) error {
	return w.write(presentation.RulesetNormalizationLine(id, name, record))
}

func (w exportWarnings) warnRulesetRequiredApprovalsNormalization(id int64, name string, record settings.RulesetRequiredApprovalsNormalization) error {
	return w.write(presentation.RulesetRequiredApprovalsNormalizationLine(id, name, record))
}

func (w exportWarnings) warnRulesetMergeMethodsNormalization(id int64, name string, record settings.RulesetMergeMethodsNormalization) error {
	return w.write(presentation.RulesetMergeMethodsNormalizationLine(id, name, record))
}

func (w exportWarnings) warnRulesetTeamApprovalsNormalization(id int64, name string, record settings.RulesetTeamApprovalsNormalization) error {
	return w.write(presentation.RulesetTeamApprovalsNormalizationLine(id, name, record))
}

func (w exportWarnings) warnRulesetBypassModeNormalization(id int64, name string, record settings.RulesetBypassModeNormalization) error {
	return w.write(presentation.RulesetBypassNormalizationLine(id, name, record))
}

func (w exportWarnings) warnEnvironmentDeploymentNormalization(name string, record settings.EnvironmentDeploymentNormalization) error {
	return w.write(presentation.EnvironmentDeploymentNormalizationLine(name, record))
}

func (w exportWarnings) warnEnvironmentRefTypeNormalization(name string, pattern settings.EnvironmentPatternState) error {
	return w.write(presentation.EnvironmentRefTypeNormalizationLine(name, pattern))
}

func (w exportWarnings) warnRulesetScanningAlertsNormalization(id int64, name string, record settings.RulesetScanningAlertsNormalization) error {
	return w.write(presentation.RulesetScanningAlertsNormalizationLine(id, name, record))
}

func (w exportWarnings) warnRulesetScanningSecurityAlertsNormalization(id int64, name string, record settings.RulesetScanningSecurityAlertsNormalization) error {
	return w.write(presentation.RulesetScanningSecurityAlertsNormalizationLine(id, name, record))
}
