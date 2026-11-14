package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

var errUnsupportedCommand = errors.New("unsupported command or arguments; usage: gh settings <export|validate|plan|apply>")

type PlanOptions struct{ Prune settings.PruneScopes }
type ApplyOptions struct {
	NoWriteback bool
	Prune       settings.PruneScopes
}

// Operations binds shared application operations to their real dependencies.
type Operations struct {
	Export   func() error
	Validate func() (settings.Declaration, error)
	Plan     func(PlanOptions) (app.PlanResult, error)
	Apply    func(ApplyOptions) (app.ApplyResult, error)
}

// Run adapts arguments, typed results and errors to terminal output and exit status.
func Run(args []string, stdout, stderr io.Writer, version string, operations Operations) int {
	if err := executeCommand(args, stdout, version, operations); err != nil {
		fmt.Fprintf(stderr, "gh settings: %v\n", err)
		return 1
	}
	return 0
}

func executeCommand(args []string, stdout io.Writer, version string, operations Operations) error {
	if len(args) > 0 && args[0] == "export" {
		if len(args) != 1 {
			switch args[1] {
			case "--security", "--rulesets", "--environments", "--actions", "--actions-policies", "--workflow-permissions", "--cache-retention", "--cache-storage", "--retention", "--fork-approval", "--archive", "--pages", "--copilot-custom-instructions", "--copilot-mcp", "--copilot-review-effort", "--pull-request-limits", "--pull-request-limit-exemptions", "--variable", "--agent-variable", "--environment-variable":
				return fmt.Errorf("export selection options are removed; export now observes supported settings automatically")
			default:
				return errUnsupportedCommand
			}
		}
		return operations.Export()
	}
	var applyOptions ApplyOptions
	var planOptions PlanOptions
	if len(args) > 0 && (args[0] == "plan" || args[0] == "apply") {
		seenPrune := false
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--no-writeback":
				if args[0] != "apply" || applyOptions.NoWriteback {
					return errUnsupportedCommand
				}
				applyOptions.NoWriteback = true
			case "--prune":
				if seenPrune || i+1 >= len(args) {
					return fmt.Errorf("use --prune once with comma-separated scopes")
				}
				seenPrune = true
				i++
				prune, err := settings.ParsePruneScopes(args[i])
				if err != nil {
					return err
				}
				planOptions.Prune, applyOptions.Prune = prune, prune
			default:
				return errUnsupportedCommand
			}
		}
		args = args[:1]
	}
	if len(args) != 1 {
		return errUnsupportedCommand
	}
	var data []byte
	var createdIDs []app.CreatedObject
	var collaborators []app.CollaboratorResult
	var collaboratorDeletions []settings.CollaboratorDeleteResult
	switch args[0] {
	case "--version", "-v":
		if _, err := fmt.Fprintln(stdout, "gh-settings "+version); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		return nil
	case "--help", "-h":
		_, err := fmt.Fprintln(stdout, "Usage: gh settings export\n       gh settings validate\n       gh settings plan [--prune SCOPES]\n       gh settings apply [--no-writeback] [--prune SCOPES]\nExport all supported readable settings to .github/settings.yml. Independent unavailable settings emit info and retain valid prior declarations; initial export omits them. Variables are plaintext. Secret values are never read. Refuse invalid or concurrently changed files.\nPlan is read-only; apply changes declared settings. --prune adds unlisted deletions for present collections. Supported scopes: "+strings.Join(settings.PruneScopeNames(), ", ")+". Standalone all selects every supported scope.")
		if err != nil {
			return fmt.Errorf("write help: %w", err)
		}
		return nil
	case "validate":
		_, err := operations.Validate()
		return err
	case "plan":
		result, err := operations.Plan(planOptions)
		if err != nil {
			return err
		}
		if err := settings.ValidateAutolinkChanges(result.Comparison.Autolinks); err != nil {
			return err
		}
		if err := settings.ValidateClassicChanges(result.Comparison.BranchProtectionRules); err != nil {
			return err
		}
		if err := presentation.ValidateRulesetChanges(result.Comparison.Rulesets); err != nil {
			return fmt.Errorf("repository.settings.rulesets: %w", err)
		}
		if err := presentation.ValidateActionsEventPolicyChanges(result.Comparison.ActionsPolicies); err != nil {
			return fmt.Errorf("repository.settings.actions.policies: %w", err)
		}
		data = RenderComparison(result.Comparison)
	case "apply":
		result, err := operations.Apply(applyOptions)
		if err != nil {
			for _, notice := range result.Notices {
				err = fmt.Errorf("%w; note: %s", err, notice)
			}
			return collaboratorDeletionDiagnostic(collaboratorDiagnostic(createdIDDiagnostic(applyDiagnostic(err, result), result.CreatedIDs), result.Collaborators), result.CollaboratorDeletions)
		}
		createdIDs = result.CreatedIDs
		collaborators = result.Collaborators
		collaboratorDeletions = result.CollaboratorDeletions
		if result.Completed() {
			data = []byte("Applied repository settings.\n")
		} else {
			data = []byte("No managed repository settings.\n")
		}
		for _, person := range result.Collaborators {
			data = append(data, []byte(collaboratorResultLine(person)+"\n")...)
		}
		for _, deletion := range result.CollaboratorDeletions {
			data = append(data, []byte(collaboratorDeletionResultLine(deletion)+"\n")...)
		}
		for _, object := range result.CreatedIDs {
			data = append(data, []byte(fmt.Sprintf("Created %s %q: id %d\n", createdCollectionPath(object.Collection), object.Name, object.ID))...)
		}
		for _, notice := range result.Notices {
			data = append(data, []byte("Note: "+notice+"\n")...)
		}
	default:
		return errUnsupportedCommand
	}
	if _, err := stdout.Write(data); err != nil {
		return collaboratorDeletionDiagnostic(collaboratorDiagnostic(createdIDDiagnostic(fmt.Errorf("write output: %w", err), createdIDs), collaborators), collaboratorDeletions)
	}
	return nil
}

func createdIDDiagnostic(err error, objects []app.CreatedObject) error {
	for _, object := range objects {
		err = fmt.Errorf("%w; reported %s %q id %d (retain this ID before retrying)", err, createdCollectionPath(object.Collection), object.Name, object.ID)
	}
	return err
}

func createdCollectionPath(collection string) string {
	switch collection {
	case "labels":
		return "repository.labels"
	case "actions_policies":
		return "repository.settings.actions.policies"
	case "rulesets":
		return "repository.settings.rulesets"
	case "deploy_keys":
		return settings.DeployKeysPath
	case "autolinks":
		return settings.AutolinksPath
	case "webhooks":
		return settings.WebhooksPath
	case "branch_protection_rules":
		return settings.ClassicProtectionPath
	default:
		return collection
	}
}

// applyDiagnostic uses recorded operation identity, retaining the original
// error and its uncertainty about earlier or failed writes.
func applyDiagnostic(err error, result app.ApplyResult) error {
	for i := len(result.Operations) - 1; i >= 0; i-- {
		op := result.Operations[i]
		if op.Status != app.OperationFailedRead && op.Status != app.OperationFailedUnknown && op.Status != app.OperationFailedLocal {
			continue
		}
		path := ""
		switch op.Kind {
		case app.WebhooksRead, app.WebhookUpdate, app.WebhookCreate:
			path = settings.WebhooksPath
		case app.DeployKeysRead, app.DeployKeyCreate, app.DeployKeyDelete:
			path = settings.DeployKeysPath
		case app.CopilotReviewEffortUpdate:
			path = settings.CopilotReviewEffortPath
		case app.CopilotInstructionsUpdate:
			path = settings.CopilotInstructionsPath
		case app.RunnersRead, app.RunnerLabelsUpdate:
			path = settings.RunnersPath
		case app.OIDCSubjectUpdate:
			path = settings.OIDCSubjectPath
		case app.CopilotMCPUpdate:
			path = settings.CopilotMCPPath
		case app.PullRequestExemptionsUpdate:
			path = settings.PullRequestExemptionsPath
		case app.PullRequestLimitsUpdate:
			path = settings.PullRequestLimitsPath
		case app.EnvironmentVariableUpsert, app.EnvironmentVariableDelete, app.EnvironmentVariablesRead:
			path = fmt.Sprintf("repository.settings.environments[%q].variables[%q]", op.EnvironmentName, op.VariableName)
		case app.AgentVariableUpsert, app.AgentVariableDelete, app.AgentVariablesRead:
			path = "repository.settings.agents.variables"
		case app.VariableUpsert, app.VariableDelete, app.VariablesRead:
			path = "repository.settings.actions.variables"
		case app.RepositoryUpdate:
			path = "repository"
		case app.CollaboratorsRead, app.CollaboratorReconcile, app.CollaboratorIdentityBinding, app.CollaboratorCancelInvitation, app.CollaboratorRemove:
			path = "repository.settings.collaborators"
		case app.TopicsUpdate:
			path = "repository.about.topics"
		case app.LabelsRead, app.LabelCreate, app.LabelUpdate, app.LabelDelete:
			path = "repository.labels"
		case app.ActionsPoliciesRead, app.ActionsEventPolicyCreate, app.ActionsEventPolicyUpdate, app.ActionsEventPolicyDelete:
			path = "repository.settings.actions.policies"
		case app.RulesetsRead, app.RulesetCreate, app.RulesetUpdate, app.RulesetDelete:
			path = "repository.settings.rulesets"
		case app.AutolinksRead, app.AutolinkCreate, app.AutolinkDelete:
			path = settings.AutolinksPath
		case app.ClassicProtectionRead, app.ClassicProtectionCreate, app.ClassicProtectionUpdate, app.ClassicProtectionDelete:
			path = settings.ClassicProtectionPath
		case app.EnvironmentsRead, app.EnvironmentUpsert, app.EnvironmentDelete:
			path = "repository.settings.environments"
		case app.ActionsPolicyUpdate, app.WorkflowPermissionsUpdate, app.CacheRetentionRead, app.CacheRetentionUpdate, app.CacheStorageRead, app.CacheStorageUpdate, app.ActionsAccessRead, app.ActionsAccessUpdate, app.RetentionRead, app.RetentionUpdate, app.ForkApprovalRead, app.ForkApprovalUpdate, app.PrivateForkRead, app.PrivateForkUpdate:
			path = "repository.settings.actions.general"
		case app.DependabotUpdate:
			path = presentation.SettingPath(op.SecurityField)
		case app.DependabotRead:
			path = "repository.settings.advanced_security.dependabot"
		case app.SecretScanningUpdate:
			path = presentation.SettingPath(op.SecurityField)
		case app.AIScanRepositoriesRead, app.AIScanRepositoriesUpdate:
			path = "repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_repositories"
		case app.AIScanPullRequestsRead, app.AIScanPullRequestsUpdate:
			path = "repository.settings.advanced_security.code_security.ai_scan.ai_scan_for_pull_requests"
		case app.CodeQLConfigurationRead, app.CodeQLConfigurationUpdate:
			path = settings.CodeQLConfigurationPath
		case app.CodeQLDefaultSetupRead, app.CodeQLDefaultSetupUpdate:
			path = "repository.settings.advanced_security.code_security.codeql_analysis.default_setup"
		case app.SecretScanningRead, app.PrivateReportingRead, app.PrivateReportingUpdate:
			path = "repository.settings.advanced_security"
		case app.ImmutableReleasesRead, app.ImmutableReleasesUpdate:
			path = "repository.settings.general.releases.immutable_release"
		case app.PagesRead, app.PagesUpdate, app.PagesCreate:
			path = "repository.settings.pages"
		case app.ArchiveRead, app.ArchiveUnarchive, app.ArchiveUpdate:
			path = "repository.settings.general.danger_zone.archive_this_repository"
		}
		if path != "" {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	return err
}
