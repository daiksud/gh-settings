package bindings

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func writeRuleset(repo repository.Repository, change settings.RulesetChange) (int64, error) {
	client, err := github.NewRulesetsClient(repo.Host)
	if err != nil {
		return 0, err
	}
	client.RulesetNormalizationWarning = warnRulesetNormalization
	client.RulesetRequiredApprovalsNormalizationWarning = warnRulesetRequiredApprovalsNormalization
	client.RulesetMergeMethodsNormalizationWarning = warnRulesetMergeMethodsNormalization
	client.RulesetTeamApprovalsNormalizationWarning = warnRulesetTeamApprovalsNormalization
	client.RulesetScanningAlertsNormalizationWarning = warnRulesetScanningAlertsNormalization
	client.RulesetScanningSecurityAlertsNormalizationWarning = warnRulesetScanningSecurityAlertsNormalization
	client.RulesetBypassModeNormalizationWarning = warnRulesetBypassModeNormalization
	return client.ApplyRuleset(repo, change)
}

func writeClassicProtection(repo repository.Repository, selection settings.ClassicSelection) (settings.ClassicWriteResult, error) {
	client, err := github.NewClassicProtectionClient(repo.Host)
	if err != nil {
		return settings.ClassicWriteResult{}, &clientPreWriteFailure{err}
	}
	return client.Apply(repo, selection)
}

func deleteClassicProtection(repo repository.Repository, candidate settings.ClassicPruneCandidate) error {
	client, err := github.NewClassicProtectionClient(repo.Host)
	if err != nil {
		return &clientPreWriteFailure{err}
	}
	return client.Delete(repo, candidate)
}
