// Package bindings connects application operations to clients for the selected GitHub host.
package bindings

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

// Bind each method to the selected repository's host when it is called.
// Local-only commands never construct a client or need GitHub credentials.
func bindRead[Client, Result any](
	newClient func(string) (Client, error),
	read func(Client, repository.Repository) (Result, error),
) func(repository.Repository) (Result, error) {
	return func(repo repository.Repository) (Result, error) {
		client, err := newClient(repo.Host)
		if err != nil {
			var zero Result
			return zero, err
		}
		return read(client, repo)
	}
}

func bindInput[Client, Input, Result any](
	newClient func(string) (Client, error),
	operation func(Client, repository.Repository, Input) (Result, error),
) func(repository.Repository, Input) (Result, error) {
	return func(repo repository.Repository, input Input) (Result, error) {
		client, err := newClient(repo.Host)
		if err != nil {
			var zero Result
			return zero, err
		}
		return operation(client, repo, input)
	}
}

// Reader binds the GitHub clients used to observe repository settings.
func Reader() app.Reader {
	return app.Reader{
		ObserveRepository: github.ObserveRepository,
		// Lifecycle reads use the redirect-rejecting write client.
		Archive:               bindRead(github.NewWriteClient, (*github.Client).ReadArchive),
		Topics:                bindRead(github.NewReadClient, (*github.Client).ReadTopics),
		Labels:                bindRead(github.NewReadClient, (*github.Client).ReadLabels),
		Rulesets:              bindRead(github.NewRulesetsClient, (*github.Client).ReadRulesets),
		BranchProtectionRules: bindRead(github.NewClassicProtectionClient, (*github.ClassicProtectionClient).Read),
		Autolinks:             bindRead(github.NewAutolinksClient, (*github.Client).ReadAutolinks),

		Collaborators:       bindRead(github.NewCollaboratorsClient, (*github.Client).ReadCollaborators),
		ResolveCollaborator: resolveCollaborator,
		CollaboratorPrune:   bindInput(github.NewCollaboratorsClient, (*github.Client).ReadCollaboratorPrune),
		DeployKeys:          bindRead(github.NewDeployKeysClient, (*github.Client).ReadDeployKeys),
		Webhooks:            bindInput(github.NewWebhooksClient, (*github.Client).ReadWebhooks),
		WebhookPrune:        bindRead(github.NewWebhooksClient, (*github.Client).ReadWebhookPrune),

		Actions:             bindInput(github.NewActionsPolicyClient, (*github.ActionsPolicyClient).ReadActionsPolicy),
		WorkflowPermissions: bindRead(github.NewActionsPolicyClient, (*github.ActionsPolicyClient).ReadWorkflowPermissions),
		ActionsAccess:       bindRead(github.NewActionsAccessClient, (*github.ActionsAccessClient).Read),
		CacheStorage:        bindRead(github.NewCacheStorageClient, (*github.CacheStorageClient).Read),
		CacheRetention:      bindRead(github.NewCacheRetentionClient, (*github.CacheRetentionClient).Read),
		Retention:           bindRead(github.NewRetentionClient, (*github.RetentionClient).Read),
		ForkApproval:        readForkApproval,
		PrivateFork:         bindRead(github.NewForkWorkflowsClient, (*github.ForkWorkflowsClient).ReadPrivate),
		ActionsPolicies:     bindRead(github.NewActionsPoliciesClient, (*github.Client).ReadActionsPolicies),

		AllRunners:  bindRead(github.NewRunnersClient, (*github.RunnersClient).ReadAll),
		Runners:     bindInput(github.NewRunnersClient, (*github.RunnersClient).ReadSelected),
		RunnerPrune: bindRead(github.NewRunnersClient, (*github.RunnersClient).ReadPrune),
		OIDCSubject: bindRead(github.NewOIDCSubjectClient, (*github.OIDCSubjectClient).Read),

		Variables:            bindRead(github.NewVariablesClient, (*github.Client).ReadVariables),
		Variable:             bindInput(github.NewVariablesClient, (*github.Client).ReadVariable),
		AgentVariables:       bindRead(github.NewVariablesClient, (*github.Client).ReadAgentVariables),
		AgentVariable:        bindInput(github.NewVariablesClient, (*github.Client).ReadAgentVariable),
		EnvironmentNames:     bindRead(github.NewEnvironmentsClient, (*github.EnvironmentsClient).Names),
		Environments:         bindInput(github.NewEnvironmentsClient, (*github.EnvironmentsClient).Read),
		EnvironmentPrune:     bindInput(github.NewEnvironmentPruneClient, (*github.EnvironmentsClient).ReadEnvironmentPrune),
		EnvironmentVariables: bindInput(github.NewVariablesClient, (*github.Client).ReadEnvironmentVariables),
		EnvironmentVariable:  readEnvironmentVariable,

		PullRequestLimits:     bindRead(github.NewPullRequestLimitsClient, (*github.PullRequestLimitsClient).Read),
		PullRequestExemptions: bindRead(github.NewPullRequestLimitsClient, (*github.PullRequestLimitsClient).ReadExemptions),
		CopilotReviewEffort:   bindRead(github.NewCopilotReviewEffortClient, (*github.CopilotReviewEffortClient).Read),
		CopilotInstructions:   bindRead(github.NewCopilotInstructionsClient, (*github.CopilotInstructionsClient).Read),
		CopilotContext:        bindRead(github.NewCopilotContextClient, (*github.CopilotContextClient).Read),

		CodeQLDefaultSetup:  bindRead(github.NewCodeQLDefaultSetupClient, (*github.CodeQLDefaultSetupClient).Read),
		CodeQLConfiguration: bindRead(github.NewCodeQLDefaultSetupClient, (*github.CodeQLDefaultSetupClient).ReadConfiguration),
		CodeQLForExport:     readCodeQLForExport,
		AIScanPullRequests:  bindRead(github.NewAIScanPullRequestsClient, (*github.AIScanPullRequestsClient).Read),
		AIScanRepositories:  bindRead(github.NewAIScanRepositoriesClient, (*github.AIScanRepositoriesClient).Read),
		SecretScanning:      readSecretScanning,
		Dependabot:          readDependabot,
		PrivateReporting:    bindRead(github.NewPrivateReportingClient, (*github.PrivateReportingClient).Read),
		ImmutableReleases:   bindRead(github.NewImmutableReleasesClient, (*github.ImmutableReleasesClient).Read),
		Pages:               readPages,
	}
}

// Writer binds the GitHub clients used to apply the declaration and record created IDs.
func Writer(read app.Reader, desired settings.Declaration, record func(string, string, int64) error) app.Writer {
	var bind func(string, int64) error
	if record != nil {
		bind = func(username string, id int64) error { return record("collaborators", username, id) }
	}
	return app.Writer{
		ObserveRepository: read.ObserveRepository,
		Repository:        writeRepository,
		ReadArchive:       read.Archive,
		Archive:           writeArchive,
		ReadTopics:        read.Topics,
		Topics:            bindInput(github.NewWriteClient, (*github.Client).ApplyTopics),
		ReadLabels:        bindRead(github.NewWriteClient, (*github.Client).ReadLabels),
		CreateLabel:       bindInput(github.NewWriteClient, (*github.Client).CreateLabel),
		UpdateLabel:       updateLabel,
		DeleteLabel:       deleteLabel,
		CreatedID:         record,

		ReadRulesets:               read.Rulesets,
		Ruleset:                    writeRuleset,
		ReadBranchProtectionRules:  read.BranchProtectionRules,
		BranchProtectionRule:       writeClassicProtection,
		DeleteBranchProtectionRule: deleteClassicProtection,
		ReadAutolinks:              read.Autolinks,
		CreateAutolink:             createAutolink,
		DeleteAutolink:             deleteAutolink,

		ReadCollaborators:        read.Collaborators,
		ResolveCollaborator:      read.ResolveCollaborator,
		Collaborator:             reconcileCollaborator,
		BindCollaboratorID:       bind,
		ReadCollaboratorPrune:    read.CollaboratorPrune,
		RefreshCollaboratorPrune: refreshCollaboratorPrune,
		DeleteCollaborator:       deleteCollaborator,

		ReadDeployKeys:   read.DeployKeys,
		CreateDeployKey:  createDeployKey,
		DeleteDeployKey:  deleteDeployKey,
		ReadWebhooks:     read.Webhooks,
		Webhook:          bindInput(github.NewWebhooksClient, (*github.Client).ApplyWebhook),
		CreateWebhook:    createWebhook,
		ReadWebhookPrune: read.WebhookPrune,
		DeleteWebhook:    deleteWebhook,

		Actions:                 writeActionsPolicy,
		ReadWorkflowPermissions: read.WorkflowPermissions,
		WorkflowPermissions:     writeWorkflowPermissions,
		ReadActionsAccess:       read.ActionsAccess,
		ActionsAccess:           bindInput(github.NewActionsAccessClient, (*github.ActionsAccessClient).Update),
		ReadCacheRetention:      read.CacheRetention,
		CacheRetention:          bindInput(github.NewCacheRetentionClient, (*github.CacheRetentionClient).Update),
		ReadCacheStorage:        read.CacheStorage,
		CacheStorage:            bindInput(github.NewCacheStorageClient, (*github.CacheStorageClient).Update),
		ReadRetention:           read.Retention,
		Retention:               bindInput(github.NewRetentionClient, (*github.RetentionClient).Update),
		ReadForkApproval:        readForkApprovalForApply,
		ForkApproval:            writeForkApproval,
		ReadPrivateFork:         read.PrivateFork,
		PrivateFork:             bindInput(github.NewForkWorkflowsClient, (*github.ForkWorkflowsClient).UpdatePrivate),
		ReadActionsPolicies:     read.ActionsPolicies,
		ActionsEventPolicy:      bindInput(github.NewActionsPoliciesClient, (*github.Client).WriteActionsEventPolicy),

		ReadRunners:             read.Runners,
		RunnerLabels:            bindInput(github.NewRunnersClient, (*github.RunnersClient).Update),
		ReadRunnerPrune:         read.RunnerPrune,
		DeleteRunnerPrune:       deleteRunnerPrune,
		VerifyArchivedRunners:   verifyArchivedRunners,
		OIDCSubject:             bindInput(github.NewOIDCSubjectClient, (*github.OIDCSubjectClient).Update),
		ReadArchivedOIDCSubject: bindInput(github.NewOIDCSubjectClient, (*github.OIDCSubjectClient).ReadArchived),

		ReadVariables:             read.Variables,
		Variable:                  upsertVariable,
		DeleteVariable:            deleteVariable,
		ReadArchivedVariable:      read.Variable,
		ReadAgentVariables:        read.AgentVariables,
		AgentVariable:             upsertAgentVariable,
		DeleteAgentVariable:       deleteAgentVariable,
		ReadArchivedAgentVariable: read.AgentVariable,

		ReadEnvironments:                 read.Environments,
		Environment:                      writeEnvironment,
		ReadEnvironmentPrune:             read.EnvironmentPrune,
		DeleteEnvironment:                deleteEnvironment,
		ReadEnvironmentVariables:         read.EnvironmentVariables,
		EnvironmentVariable:              upsertEnvironmentVariable,
		DeleteEnvironmentVariable:        deleteEnvironmentVariable,
		ReadArchivedEnvironmentVariables: readArchivedEnvironmentVariables,

		ReadPullRequestLimits:     read.PullRequestLimits,
		PullRequestLimits:         bindInput(github.NewPullRequestLimitsClient, (*github.PullRequestLimitsClient).Update),
		ReadPullRequestExemptions: read.PullRequestExemptions,
		PullRequestExemptions:     updatePullRequestExemptions,
		CopilotInstructions:       bindInput(github.NewCopilotInstructionsClient, (*github.CopilotInstructionsClient).Update),
		CopilotMCP:                bindInput(github.NewCopilotContextClient, (*github.CopilotContextClient).UpdateMCP),
		CopilotPublicSearch:       bindInput(github.NewCopilotContextClient, (*github.CopilotContextClient).UpdatePublicSearch),
		ReadCopilotReviewEffort:   read.CopilotReviewEffort,
		CopilotReviewEffort:       bindInput(github.NewCopilotReviewEffortClient, (*github.CopilotReviewEffortClient).Update),
		ReadArchivedCopilot:       readArchivedCopilot,

		ReadCodeQLDefaultSetup:  read.CodeQLDefaultSetup,
		CodeQLDefaultSetup:      bindInput(github.NewCodeQLDefaultSetupClient, (*github.CodeQLDefaultSetupClient).Update),
		ReadCodeQLConfiguration: read.CodeQLConfiguration,
		CodeQLConfiguration:     bindInput(github.NewCodeQLDefaultSetupClient, (*github.CodeQLDefaultSetupClient).UpdateConfiguration),
		ReadAIScanRepositories:  read.AIScanRepositories,
		AIScanRepositories:      bindInput(github.NewAIScanRepositoriesClient, (*github.AIScanRepositoriesClient).Update),
		ReadAIScanPullRequests:  read.AIScanPullRequests,
		AIScanPullRequests:      bindInput(github.NewAIScanPullRequestsClient, (*github.AIScanPullRequestsClient).Update),
		ReadSecretScanning: func(repo repository.Repository) (settings.SecretScanningObservation, error) {
			return readSecretScanningForApply(repo, desired.Security)
		},
		SecretScanning:        writeSecretScanning,
		ReadDependabot:        read.Dependabot,
		Dependabot:            writeDependabot,
		ReadPrivateReporting:  read.PrivateReporting,
		PrivateReporting:      bindInput(github.NewPrivateReportingClient, (*github.PrivateReportingClient).Update),
		ReadImmutableReleases: read.ImmutableReleases,
		ImmutableReleases:     bindInput(github.NewImmutableReleasesClient, (*github.ImmutableReleasesClient).Update),
		ReadPages: func(repo repository.Repository) (settings.PagesObservation, error) {
			return readPagesForApply(repo, desired.Pages)
		},
		Pages: writePages,
	}
}

type clientPreWriteFailure struct{ error }

func (*clientPreWriteFailure) ReadFailedBeforeWrite() {}
