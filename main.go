package main

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// version is set with -ldflags at build time.
var version = "dev"

func main() {
	checkout := workspace.NewCheckout(".")
	loadDeclaration := func() (settings.Declaration, error) { return loadCheckoutDeclaration(checkout) }
	reader := app.Reader{Pages: readPages, Archive: readArchive, ObserveRepository: github.ObserveRepository, Topics: readTopics, Labels: readLabels, Rulesets: readRulesets, Actions: readActionsPolicy, WorkflowPermissions: readWorkflowPermissions, SecretScanning: readSecretScanning, PrivateReporting: readPrivateReporting, ImmutableReleases: readImmutableReleases, Retention: readRetention, ForkApproval: readForkApproval, PrivateFork: readPrivateFork}
	reader.Environments = readEnvironments
	if len(os.Args) > 1 && os.Args[1] == "action" {
		os.Exit(actions.Run(os.Args[2:], os.Stdout, os.Stderr, version, func() (app.PlanResult, error) {
			return app.Plan(loadDeclaration, checkout.Repository, reader)
		}))
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version, cli.Operations{
		Export: func(options cli.ExportOptions) error {
			return app.Export(app.ExportCheckout{IncludeEnvironments: options.Environments, IncludePages: options.Pages, IncludeArchive: options.Archive, Repository: checkout.Repository, PrepareSave: func() (func(settings.Declaration) error, error) { return prepareCheckoutSave(checkout) }, IncludeRulesets: options.Rulesets, IncludeActions: options.Actions, IncludeWorkflowPermissions: options.WorkflowPermissions, NormalizationWarning: warnNormalization, CommitMessageNormalizationWarning: warnCommitMessageNormalization, SecurityStatusNormalizationWarning: warnSecurityStatusNormalization, RulesetNormalizationWarning: warnRulesetNormalization, IncludeSecretScanning: options.SecretScanning, IncludePrivateReporting: options.PrivateReporting, IncludeImmutableReleases: options.ImmutableReleases, IncludeRetention: options.Retention, IncludeForkApproval: options.ForkApproval, IncludePrivateFork: options.PrivateFork, ExcludedRulesets: func(count int) {
				fmt.Fprintf(os.Stderr, "Excluded %d repository push Rulesets from export (outside managed branch/tag scope).\n", count)
			}}, reader)
		},
		Validate: func() (settings.Declaration, error) { return app.Validate(loadDeclaration) },
		Plan: func() (app.PlanResult, error) {
			return app.Plan(loadDeclaration, checkout.Repository, reader)
		},
		Apply: func(options cli.ApplyOptions) (app.ApplyResult, error) {
			desired, record, err := prepareCheckoutApply(checkout)
			if err != nil {
				return app.ApplyResult{Outcome: app.ApplyFailed}, err
			}
			if options.NoWriteback || os.Getenv("GITHUB_ACTIONS") == "true" {
				record = nil
			}
			return app.Apply(func() (settings.Declaration, error) { return desired, nil }, checkout.Repository, app.Writer{ReadEnvironments: readEnvironments, Environment: writeEnvironment, CreatedID: record, ReadPages: func(repo repository.Repository) (settings.PagesObservation, error) {
				return readPagesForApply(repo, desired.Pages)
			}, Pages: writePages, ReadArchive: readArchiveForApply, Archive: writeArchive, Repository: writeRepository, Topics: writeTopics, ReadLabels: readLabelsForApply, CreateLabel: createLabel, UpdateLabel: updateLabel, DeleteLabel: deleteLabel, ReadRulesets: readRulesets, Ruleset: writeRuleset, Actions: writeActionsPolicy, WorkflowPermissions: writeWorkflowPermissions, ReadSecretScanning: func(repo repository.Repository) (settings.SecretScanningObservation, error) {
				return readSecretScanningForApply(repo, desired.Security)
			}, SecretScanning: writeSecretScanning, ReadPrivateReporting: readPrivateReporting, ReadImmutableReleases: readImmutableReleases, ImmutableReleases: writeImmutableReleases, PrivateReporting: writePrivateReporting, ReadRetention: readRetention, Retention: writeRetention, ReadForkApproval: readForkApprovalForApply, ForkApproval: writeForkApproval, ReadPrivateFork: readPrivateFork, PrivateFork: writePrivateFork})
		},
	}))
}

func writeRepository(repo repository.Repository, desired settings.RepositoryDeclaration) error {
	return writeRepositoryWithWriters(repo, desired,
		func(repo repository.Repository, restDesired settings.RepositoryDeclaration) error {
			client, err := github.NewWriteClient(repo.Host)
			if err != nil {
				return err
			}
			client.CommitMessageNormalizationWarning = warnCommitMessageNormalization
			return client.ApplyRepository(repo, restDesired)
		},
		func(repo repository.Repository, accessDesired settings.RepositoryDeclaration) error {
			client, err := github.NewRepositoryAccessClient(repo.Host)
			if err != nil {
				return err
			}
			client.NormalizationWarning = warnNormalization
			return client.ApplyRepositoryAccess(repo, accessDesired)
		})
}

func readTopics(repo repository.Repository) ([]string, error) {
	client, err := github.NewReadClient(repo.Host)
	if err != nil {
		return nil, err
	}
	return client.ReadTopics(repo)
}

func writeTopics(repo repository.Repository, desired settings.TopicsDeclaration) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
	}
	return client.ApplyTopics(repo, desired)
}

func readLabels(repo repository.Repository) (settings.LabelObservation, error) {
	client, err := github.NewReadClient(repo.Host)
	if err != nil {
		return settings.LabelObservation{}, err
	}
	return client.ReadLabels(repo)
}
func readLabelsForApply(repo repository.Repository) (settings.LabelObservation, error) {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return settings.LabelObservation{}, err
	}
	return client.ReadLabels(repo)
}
func createLabel(repo repository.Repository, change settings.LabelChange) (int64, error) {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return 0, err
	}
	return client.CreateLabel(repo, change)
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

func readRulesets(repo repository.Repository) (settings.RulesetObservation, error) {
	client, err := github.NewRulesetsClient(repo.Host)
	if err != nil {
		return settings.RulesetObservation{}, err
	}
	return client.ReadRulesets(repo)
}

func readEnvironments(repo repository.Repository, names []string) (settings.EnvironmentObservation, error) {
	client, err := github.NewEnvironmentsClient(repo.Host)
	if err != nil {
		return settings.EnvironmentObservation{}, err
	}
	return client.Read(repo, names)
}
func writeEnvironment(repo repository.Repository, desired settings.EnvironmentDeclaration) (bool, error) {
	client, err := github.NewEnvironmentsClient(repo.Host)
	if err != nil {
		return false, err
	}
	return client.Update(repo, desired)
}
func writeRuleset(repo repository.Repository, change settings.RulesetChange) (int64, error) {
	client, err := github.NewRulesetsClient(repo.Host)
	if err != nil {
		return 0, err
	}
	client.RulesetNormalizationWarning = warnRulesetNormalization
	return client.ApplyRuleset(repo, change)
}

func readActionsPolicy(repo repository.Repository, selected bool) (settings.ActionsPolicyState, error) {
	client, err := github.NewActionsPolicyClient(repo.Host)
	if err != nil {
		return settings.ActionsPolicyState{}, err
	}
	return client.ReadActionsPolicy(repo, selected)
}
func writeActionsPolicy(repo repository.Repository, desired settings.ActionsPolicyDeclaration) error {
	client, err := github.NewActionsPolicyClient(repo.Host)
	if err != nil {
		return err
	}
	client.NormalizationWarning = warnNormalization
	return client.ApplyActionsPolicy(repo, desired)
}

func readWorkflowPermissions(repo repository.Repository) (settings.WorkflowPermissionsState, error) {
	client, err := github.NewActionsPolicyClient(repo.Host)
	if err != nil {
		return settings.WorkflowPermissionsState{}, err
	}
	return client.ReadWorkflowPermissions(repo)
}
func writeWorkflowPermissions(repo repository.Repository, desired settings.ActionsPolicyDeclaration) error {
	client, err := github.NewActionsPolicyClient(repo.Host)
	if err != nil {
		return err
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

func readPrivateReporting(repo repository.Repository) (settings.PrivateReportingObservation, error) {
	client, err := github.NewPrivateReportingClient(repo.Host)
	if err != nil {
		return settings.PrivateReportingObservation{}, err
	}
	return client.Read(repo)
}
func writePrivateReporting(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	client, err := github.NewPrivateReportingClient(repo.Host)
	if err != nil {
		return false, err
	}
	return client.Update(repo, change)
}

func readImmutableReleases(repo repository.Repository) (settings.ImmutableReleasesObservation, error) {
	client, err := github.NewImmutableReleasesClient(repo.Host)
	if err != nil {
		return settings.ImmutableReleasesObservation{}, err
	}
	return client.Read(repo)
}
func writeImmutableReleases(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	client, err := github.NewImmutableReleasesClient(repo.Host)
	if err != nil {
		return false, err
	}
	return client.Update(repo, change)
}

func readRetention(repo repository.Repository) (settings.RetentionObservation, error) {
	client, err := github.NewRetentionClient(repo.Host)
	if err != nil {
		return settings.RetentionObservation{}, err
	}
	return client.Read(repo)
}
func writeRetention(repo repository.Repository, days int) (bool, error) {
	client, err := github.NewRetentionClient(repo.Host)
	if err != nil {
		return false, err
	}
	return client.Update(repo, days)
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
func readPrivateFork(repo repository.Repository) (settings.PrivateForkObservation, error) {
	c, err := github.NewForkWorkflowsClient(repo.Host)
	if err != nil {
		return settings.PrivateForkObservation{}, err
	}
	return c.ReadPrivate(repo)
}
func writeForkApproval(repo repository.Repository, v string) (bool, error) {
	c, err := github.NewForkWorkflowsClient(repo.Host)
	if err != nil {
		return false, err
	}
	c.NormalizationWarning = warnNormalization
	return c.UpdateApproval(repo, v)
}
func writePrivateFork(repo repository.Repository, d settings.PrivateForkDeclaration) (bool, error) {
	c, err := github.NewForkWorkflowsClient(repo.Host)
	if err != nil {
		return false, err
	}
	return c.UpdatePrivate(repo, d)
}

func readArchive(repo repository.Repository) (settings.ArchiveState, error) {
	client, err := github.NewWriteClient(repo.Host) // Reject redirects even for lifecycle reads.
	if err != nil {
		return settings.ArchiveState{}, err
	}
	return client.ReadArchive(repo)
}
func readArchiveForApply(repo repository.Repository) (settings.ArchiveState, error) {
	return readArchive(repo)
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
	if _, err := o.Values(); err != nil {
		return settings.PagesObservation{}, err
	}
	if record := o.ModeNormalization(); d.BuildType != nil && record != (settings.ObservationNormalization{}) {
		warnNormalization(record)
	}
	if record := o.FolderNormalization(); d.Source != nil && record != (settings.ObservationNormalization{}) {
		warnNormalization(record)
	}
	return o, nil
}

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
