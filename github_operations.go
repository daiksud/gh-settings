package main

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

type clientPreWriteFailure struct{ error }

func (*clientPreWriteFailure) ReadFailedBeforeWrite() {}

func writeRepository(repo repository.Repository, desired settings.RepositoryDeclaration) (bool, error) {
	return writeRepositoryWithWriters(repo, desired,
		func(repo repository.Repository, restDesired settings.RepositoryDeclaration) (bool, error) {
			client, err := github.NewWriteClient(repo.Host)
			if err != nil {
				return false, err
			}
			client.CommitMessageNormalizationWarning = warnCommitMessageNormalization
			return client.ApplyRepository(repo, restDesired)
		},
		func(repo repository.Repository, accessDesired settings.RepositoryDeclaration) (bool, error) {
			client, err := github.NewRepositoryAccessClient(repo.Host)
			if err != nil {
				return false, err
			}
			client.NormalizationWarning = warnNormalization
			return client.ApplyRepositoryAccess(repo, accessDesired)
		})
}

func updateLabel(repo repository.Repository, change settings.LabelChange) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
	}
	return client.UpdateLabel(repo, change)
}

func deleteLabel(repo repository.Repository, change settings.LabelChange) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
	}
	return client.DeleteLabel(repo, change)
}

func deleteEnvironment(repo repository.Repository, candidate settings.EnvironmentPruneCandidate) error {
	client, err := github.NewEnvironmentPruneClient(repo.Host)
	if err != nil {
		return &clientPreWriteFailure{err}
	}
	return client.DeleteEnvironment(repo, candidate)
}

func writeEnvironment(repo repository.Repository, desired settings.EnvironmentDeclaration) (bool, error) {
	client, err := github.NewEnvironmentsClient(repo.Host)
	if err != nil {
		return false, err
	}
	client.NormalizationWarning = warnEnvironmentDeploymentNormalization
	client.RefTypeNormalizationWarning = warnEnvironmentRefTypeNormalization
	return client.Update(repo, desired)
}

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

func writeActionsPolicy(repo repository.Repository, desired settings.ActionsPolicyDeclaration) (bool, error) {
	client, err := github.NewActionsPolicyClient(repo.Host)
	if err != nil {
		return false, err
	}
	client.NormalizationWarning = warnNormalization
	return client.ApplyActionsPolicy(repo, desired)
}

func writeWorkflowPermissions(repo repository.Repository, desired settings.ActionsPolicyDeclaration) (bool, error) {
	client, err := github.NewActionsPolicyClient(repo.Host)
	if err != nil {
		return false, err
	}
	client.NormalizationWarning = warnNormalization
	return client.ApplyWorkflowPermissions(repo, desired)
}

func readSecretScanning(repo repository.Repository) (settings.SecretScanningObservation, error) {
	client, err := github.NewSecretScanningClient(repo.Host)
	if err != nil {
		return settings.SecretScanningObservation{}, err
	}
	return client.Read(repo)
}

func readSecretScanningForApply(repo repository.Repository, desired settings.SecurityDeclaration) (settings.SecretScanningObservation, error) {
	observed, err := readSecretScanning(repo)
	if err != nil {
		return settings.SecretScanningObservation{}, err
	}
	if _, _, err := observed.Values(); err != nil {
		return settings.SecretScanningObservation{}, err
	}
	for _, record := range observed.Normalizations() {
		if record.Path == settings.SecretProtectionControl.Path() && desired.SecretScanning != nil || record.Path == settings.PushProtectionControl.Path() && desired.PushProtection != nil {
			warnSecurityStatusNormalization(record)
		}
	}
	return observed, nil
}

func writeSecretScanning(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	client, err := github.NewSecretScanningClient(repo.Host)
	if err != nil {
		return false, err
	}
	client.NormalizationWarning = warnSecurityStatusNormalization
	return client.Update(repo, change)
}

func readForkApproval(repo repository.Repository) (settings.ForkApprovalObservation, error) {
	c, err := github.NewForkWorkflowsClient(repo.Host)
	if err != nil {
		return settings.ForkApprovalObservation{}, err
	}
	return c.ReadApproval(repo)
}

func readForkApprovalForApply(repo repository.Repository) (settings.ForkApprovalObservation, error) {
	observed, err := readForkApproval(repo)
	if err != nil {
		return settings.ForkApprovalObservation{}, err
	}
	if _, err := observed.Value(); err != nil {
		return settings.ForkApprovalObservation{}, err
	}
	if record := observed.Normalization(); record.Path != "" {
		warnNormalization(record)
	}
	return observed, nil
}

func writeForkApproval(repo repository.Repository, v string) (bool, error) {
	c, err := github.NewForkWorkflowsClient(repo.Host)
	if err != nil {
		return false, err
	}
	c.NormalizationWarning = warnNormalization
	return c.UpdateApproval(repo, v)
}

func writeArchive(repo repository.Repository, previous settings.ArchiveState, archived bool) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
	}
	return client.UpdateArchive(repo, previous, archived)
}

func readPages(repo repository.Repository) (settings.PagesObservation, error) {
	c, err := github.NewPagesClient(repo.Host)
	if err != nil {
		return settings.PagesObservation{}, err
	}
	return c.Read(repo)
}

func writePages(repo repository.Repository, d settings.PagesDeclaration) (bool, error) {
	c, err := github.NewPagesClient(repo.Host)
	if err != nil {
		return false, err
	}
	c.NormalizationWarning = warnNormalization
	return c.Update(repo, d)
}

func readPagesForApply(repo repository.Repository, d settings.PagesDeclaration) (settings.PagesObservation, error) {
	o, err := readPages(repo)
	if err != nil {
		return settings.PagesObservation{}, err
	}
	if absent, err := o.Absent(); err != nil {
		return settings.PagesObservation{}, err
	} else if absent || d.Delete() {
		return o, nil
	}
	if record := o.ModeNormalization(); d.BuildType != nil && record != (settings.ObservationNormalization{}) {
		warnNormalization(record)
	}
	if record := o.FolderNormalization(); d.Source != nil && record != (settings.ObservationNormalization{}) {
		warnNormalization(record)
	}
	return o, nil
}

func upsertVariable(repo repository.Repository, value settings.RepositoryVariable) (bool, error) {
	client, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return false, &clientPreWriteFailure{err}
	}
	return client.UpsertVariable(repo, value)
}

func upsertAgentVariable(repo repository.Repository, value settings.RepositoryVariable) (bool, error) {
	client, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return false, &clientPreWriteFailure{err}
	}
	return client.UpsertAgentVariable(repo, value)
}

func readEnvironmentVariable(repo repository.Repository, environment, name string) (settings.VariableObservation, error) {
	client, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return settings.VariableObservation{}, err
	}
	return client.ReadEnvironmentVariable(repo, environment, name)
}

func upsertEnvironmentVariable(repo repository.Repository, environment string, value settings.RepositoryVariable) (bool, error) {
	client, err := github.NewVariablesClient(repo.Host)
	if err != nil {
		return false, &clientPreWriteFailure{err}
	}
	return client.UpsertEnvironmentVariable(repo, environment, value)
}

func updatePullRequestExemptions(repo repository.Repository, desired settings.PullRequestExemptionsDeclaration) (bool, error) {
	client, err := github.NewPullRequestLimitsClient(repo.Host)
	if err != nil {
		return false, &clientPreWriteFailure{err}
	}
	return client.UpdateExemptions(repo, desired)
}

func readCodeQLForExport(repo repository.Repository) (settings.CodeQLDefaultSetupObservation, settings.CodeQLConfigurationObservation, error) {
	client, err := github.NewCodeQLDefaultSetupClient(repo.Host)
	if err != nil {
		return settings.CodeQLDefaultSetupObservation{}, settings.CodeQLConfigurationObservation{}, err
	}
	return client.ReadForExport(repo)
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

func verifyArchivedRunners(repo repository.Repository, desired []settings.RunnerLabels, archive settings.ArchiveState) error {
	client, err := github.NewRunnersClient(repo.Host)
	if err != nil {
		return err
	}
	return client.VerifyArchivedMatching(repo, desired, archive)
}

func deleteRunnerPrune(repo repository.Repository, expected settings.RunnerPruneObservation, candidates []int64, id int64) (bool, error) {
	c, e := github.NewRunnersClient(repo.Host)
	if e != nil {
		return false, e
	}
	return c.DeletePrune(repo, expected, candidates, id)
}

// readArchivedCopilot supplies strict selected-toggle evidence for archived apply.
func readArchivedCopilot(repo repository.Repository, instructions, mcp bool) (settings.ArchivedCopilotToggles, error) {
	client, err := github.NewArchivedCopilotClient(repo.Host)
	if err != nil {
		return settings.ArchivedCopilotToggles{}, err
	}
	return client.Read(repo, instructions, mcp)
}
