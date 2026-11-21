package main

import (
	"fmt"
	"os"

	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

func warnNormalization(normalization settings.ObservationNormalization) {
	fmt.Fprintln(os.Stderr, presentation.NormalizationLine(normalization))
}

func warnCommitMessageNormalization(normalization settings.CommitMessageNormalization) {
	fmt.Fprintln(os.Stderr, presentation.CommitMessageNormalizationLine(normalization))
}

func warnSecurityStatusNormalization(record settings.SecurityStatusNormalization) {
	fmt.Fprintln(os.Stderr, presentation.SecurityStatusNormalizationLine(record))
}

func warnRulesetNormalization(id int64, name string, record settings.ObservationNormalization) {
	fmt.Fprintln(os.Stderr, presentation.RulesetNormalizationLine(id, name, record))
}

func warnRulesetRequiredApprovalsNormalization(id int64, name string, record settings.RulesetRequiredApprovalsNormalization) {
	fmt.Fprintln(os.Stderr, presentation.RulesetRequiredApprovalsNormalizationLine(id, name, record))
}

func warnRulesetMergeMethodsNormalization(id int64, name string, record settings.RulesetMergeMethodsNormalization) {
	fmt.Fprintln(os.Stderr, presentation.RulesetMergeMethodsNormalizationLine(id, name, record))
}

func warnRulesetTeamApprovalsNormalization(id int64, name string, record settings.RulesetTeamApprovalsNormalization) {
	fmt.Fprintln(os.Stderr, presentation.RulesetTeamApprovalsNormalizationLine(id, name, record))
}

func warnRulesetBypassModeNormalization(id int64, name string, record settings.RulesetBypassModeNormalization) {
	fmt.Fprintln(os.Stderr, presentation.RulesetBypassNormalizationLine(id, name, record))
}

func warnEnvironmentDeploymentNormalization(name string, record settings.EnvironmentDeploymentNormalization) {
	fmt.Fprintln(os.Stderr, presentation.EnvironmentDeploymentNormalizationLine(name, record))
}

func warnEnvironmentRefTypeNormalization(name string, pattern settings.EnvironmentPatternState) {
	fmt.Fprintln(os.Stderr, presentation.EnvironmentRefTypeNormalizationLine(name, pattern))
}

func warnRulesetScanningAlertsNormalization(id int64, name string, record settings.RulesetScanningAlertsNormalization) {
	fmt.Fprintln(os.Stderr, presentation.RulesetScanningAlertsNormalizationLine(id, name, record))
}

func warnRulesetScanningSecurityAlertsNormalization(id int64, name string, record settings.RulesetScanningSecurityAlertsNormalization) {
	fmt.Fprintln(os.Stderr, presentation.RulesetScanningSecurityAlertsNormalizationLine(id, name, record))
}
