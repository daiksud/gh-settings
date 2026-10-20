package main

import (
	"fmt"
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// version is set with -ldflags at build time.
var version = "dev"

func main() {
	checkout := workspace.NewCheckout(".")
	reader := app.Reader{ObserveRepository: github.ObserveRepository, Topics: readTopics, Labels: readLabels, Rulesets: readRulesets, Actions: readActionsPolicy, WorkflowPermissions: readWorkflowPermissions, SecretScanning: readSecretScanning, PrivateReporting: readPrivateReporting, Retention: readRetention, ForkApproval: readForkApproval, PrivateFork: readPrivateFork}
	if len(os.Args) > 1 && os.Args[1] == "action" {
		os.Exit(actions.Run(os.Args[2:], os.Stdout, os.Stderr, version, func() (app.PlanResult, error) {
			return app.Plan(checkout.Declaration, checkout.Repository, reader)
		}))
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version, cli.Operations{
		Export: func(options cli.ExportOptions) error {
			return app.Export(app.ExportCheckout{Repository: checkout.Repository, PrepareSave: checkout.PrepareSave, IncludeRulesets: options.Rulesets, IncludeActions: options.Actions, IncludeWorkflowPermissions: options.WorkflowPermissions, IncludeSecretScanning: options.SecretScanning, IncludePrivateReporting: options.PrivateReporting, IncludeRetention: options.Retention, IncludeForkApproval: options.ForkApproval, IncludePrivateFork: options.PrivateFork, ExcludedRulesets: func(count int) {
				fmt.Fprintf(os.Stderr, "Excluded %d repository push Rulesets from export (outside managed branch/tag scope).\n", count)
			}}, reader)
		},
		Validate: func() (settings.Declaration, error) { return app.Validate(checkout.Declaration) },
		Plan: func() (app.PlanResult, error) {
			return app.Plan(checkout.Declaration, checkout.Repository, reader)
		},
		Apply: func() (app.ApplyResult, error) {
			return app.Apply(checkout.Declaration, checkout.Repository, app.Writer{Repository: writeRepository, Topics: writeTopics, ReadLabels: readLabelsForApply, CreateLabel: createLabel, UpdateLabel: updateLabel, DeleteLabel: deleteLabel, ReadRulesets: readRulesets, Ruleset: writeRuleset, Actions: writeActionsPolicy, WorkflowPermissions: writeWorkflowPermissions, ReadSecretScanning: readSecretScanning, SecretScanning: writeSecretScanning, ReadPrivateReporting: readPrivateReporting, PrivateReporting: writePrivateReporting, ReadRetention: readRetention, Retention: writeRetention, ReadForkApproval: readForkApproval, ForkApproval: writeForkApproval, ReadPrivateFork: readPrivateFork, PrivateFork: writePrivateFork})
		},
	}))
}

func writeRepository(repo repository.Repository, desired settings.RepositoryDeclaration) error {
	restDesired := desired
	restDesired.HasPullRequests = nil
	restDesired.PullRequestCreationPolicy = nil
	restDesired.IssueCreationPolicy = nil
	if !restDesired.Empty() {
		client, err := github.NewWriteClient(repo.Host)
		if err != nil {
			return err
		}
		if err := client.ApplyRepository(repo, restDesired); err != nil {
			return err
		}
	}
	if desired.RepositoryAccessEmpty() {
		return nil
	}
	client, err := github.NewRepositoryAccessClient(repo.Host)
	if err == nil {
		err = client.ApplyRepositoryAccess(repo, desired)
	}
	if err != nil && !restDesired.Empty() {
		if desired.IssueCreationPolicy != nil {
			return fmt.Errorf("apply repository access (earlier repository writes remain applied): %w", err)
		}
		return fmt.Errorf("apply PR access (earlier repository writes remain applied): %w", err)
	}
	return err
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
func createLabel(repo repository.Repository, change settings.LabelChange) error {
	client, err := github.NewWriteClient(repo.Host)
	if err != nil {
		return err
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
func writeRuleset(repo repository.Repository, change settings.RulesetChange) error {
	client, err := github.NewRulesetsClient(repo.Host)
	if err != nil {
		return err
	}
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
	return client.ApplyWorkflowPermissions(repo, desired)
}

func readSecretScanning(repo repository.Repository) (settings.SecretScanningObservation, error) {
	client, err := github.NewSecretScanningClient(repo.Host)
	if err != nil {
		return settings.SecretScanningObservation{}, err
	}
	return client.Read(repo)
}
func writeSecretScanning(repo repository.Repository, change settings.SecurityChange) (bool, error) {
	client, err := github.NewSecretScanningClient(repo.Host)
	if err != nil {
		return false, err
	}
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
	return c.UpdateApproval(repo, v)
}
func writePrivateFork(repo repository.Repository, d settings.PrivateForkDeclaration) (bool, error) {
	c, err := github.NewForkWorkflowsClient(repo.Host)
	if err != nil {
		return false, err
	}
	return c.UpdatePrivate(repo, d)
}
