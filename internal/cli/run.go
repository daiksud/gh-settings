package cli

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/presentation"
	"github.com/daiksud/gh-settings/internal/settings"
)

var errUnsupportedCommand = errors.New("unsupported command or arguments; usage: gh settings <export|validate|plan|apply>")

// ExportOptions selects independently acquired opt-in resources.
type ExportOptions struct {
	ActionsPolicies                                                                                                                                                                                                                                           bool
	PullRequestLimits                                                                                                                                                                                                                                         bool
	CopilotReviewEffort                                                                                                                                                                                                                                       bool
	CopilotInstructions                                                                                                                                                                                                                                       bool
	CopilotMCP                                                                                                                                                                                                                                                bool
	Variables, AgentsVariables                                                                                                                                                                                                                                []string
	Dependabot, Rulesets, Environments, Actions, WorkflowPermissions, SecretScanning, CodeQLConfiguration, CodeQLDefaultSetup, AIScanPullRequests, PrivateReporting, ImmutableReleases, CacheRetention, CacheStorage, Retention, ForkApproval, Archive, Pages bool
}

type ApplyOptions struct{ NoWriteback bool }

// Operations binds shared application operations to their real dependencies.
type Operations struct {
	Export   func(ExportOptions) error
	Validate func() (settings.Declaration, error)
	Plan     func() (app.PlanResult, error)
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
		var options ExportOptions
		for i := 1; i < len(args); i++ {
			flag := args[i]
			switch flag {
			case "--copilot-custom-instructions":
				if options.CopilotInstructions {
					return errUnsupportedCommand
				}
				options.CopilotInstructions = true
			case "--copilot-mcp":
				if options.CopilotMCP {
					return errUnsupportedCommand
				}
				options.CopilotMCP = true
			case "--pull-request-limits":
				if options.PullRequestLimits {
					return errUnsupportedCommand
				}
				options.PullRequestLimits = true
			case "--copilot-review-effort":
				if options.CopilotReviewEffort {
					return errUnsupportedCommand
				}
				options.CopilotReviewEffort = true
			case "--variable", "--agent-variable":
				if i+1 >= len(args) {
					return errUnsupportedCommand
				}
				i++
				name, err := settings.VariableName(args[i])
				if err != nil {
					return err
				}
				selection := &options.Variables
				if flag == "--agent-variable" {
					selection = &options.AgentsVariables
				}
				if slices.Contains(*selection, name) {
					return fmt.Errorf("duplicate repository variable name %q", name)
				}
				*selection = append(*selection, name)
			case "--security":
				if i+1 >= len(args) {
					return errUnsupportedCommand
				}
				i++
				switch args[i] {
				case "dependabot":
					if options.Dependabot {
						return errUnsupportedCommand
					}
					options.Dependabot = true
				case "secret-scanning":
					if options.SecretScanning {
						return errUnsupportedCommand
					}
					options.SecretScanning = true
				case "immutable-releases":
					if options.ImmutableReleases {
						return errUnsupportedCommand
					}
					options.ImmutableReleases = true
				case "codeql-configuration":
					if options.CodeQLConfiguration {
						return errUnsupportedCommand
					}
					options.CodeQLConfiguration = true
				case "codeql-default-setup":
					if options.CodeQLDefaultSetup {
						return errUnsupportedCommand
					}
					options.CodeQLDefaultSetup = true
				case "ai-scan-pull-requests":
					if options.AIScanPullRequests {
						return errUnsupportedCommand
					}
					options.AIScanPullRequests = true
				case "private-vulnerability-reporting":
					if options.PrivateReporting {
						return errUnsupportedCommand
					}
					options.PrivateReporting = true
				default:
					return errUnsupportedCommand
				}

			case "--fork-approval":
				if options.ForkApproval {
					return errUnsupportedCommand
				}
				options.ForkApproval = true
			case "--pages":
				if options.Pages {
					return errUnsupportedCommand
				}
				options.Pages = true
			case "--archive":
				if options.Archive {
					return errUnsupportedCommand
				}
				options.Archive = true
			case "--cache-retention":
				if options.CacheRetention {
					return errUnsupportedCommand
				}
				options.CacheRetention = true
			case "--cache-storage":
				if options.CacheStorage {
					return errUnsupportedCommand
				}
				options.CacheStorage = true
			case "--retention":
				if options.Retention {
					return errUnsupportedCommand
				}
				options.Retention = true
			case "--environments":
				if options.Environments {
					return errUnsupportedCommand
				}
				options.Environments = true
			case "--rulesets":
				if options.Rulesets {
					return errUnsupportedCommand
				}
				options.Rulesets = true
			case "--actions-policies":
				if options.ActionsPolicies {
					return errUnsupportedCommand
				}
				options.ActionsPolicies = true
			case "--actions":
				if options.Actions {
					return errUnsupportedCommand
				}
				options.Actions = true
			case "--workflow-permissions":
				if options.WorkflowPermissions {
					return errUnsupportedCommand
				}
				options.WorkflowPermissions = true
			default:
				return errUnsupportedCommand
			}
		}
		return operations.Export(options)
	}
	var applyOptions ApplyOptions
	if len(args) == 2 && args[0] == "apply" && args[1] == "--no-writeback" {
		applyOptions.NoWriteback = true
		args = args[:1]
	}
	if len(args) != 1 {
		return errUnsupportedCommand
	}
	var data []byte
	var createdIDs []app.CreatedObject
	switch args[0] {
	case "--version", "-v":
		if _, err := fmt.Fprintln(stdout, "gh-settings "+version); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		return nil
	case "--help", "-h":
		if _, err := fmt.Fprintln(stdout, "Usage: gh settings export [--copilot-custom-instructions] [--copilot-mcp] [--pull-request-limits] [--copilot-review-effort] [--variable NAME]... [--agent-variable NAME]... [--pages] [--archive] [--rulesets] [--actions-policies] [--actions] [--workflow-permissions] [--cache-retention] [--cache-storage] [--retention] [--fork-approval] [--security <dependabot|secret-scanning|codeql-configuration|codeql-default-setup|ai-scan-pull-requests|private-vulnerability-reporting|immutable-releases>]...\n       gh settings validate\n       gh settings plan\n       gh settings apply [--no-writeback]\nExport the current repository's basic settings, topics and labels as YAML to .github/settings.yml at the checkout root.\nReplace an existing regular settings file; success is silent.\nUse --copilot-custom-instructions to export the stored Use custom instructions boolean; this does not enable or request Copilot review.\nUse --pull-request-limits to export all three persistent Pull request limits controls; administrator access is required. Temporary restrictions and exemptions remain unmanaged.\nUse --copilot-mcp to export the stored MCP tools boolean; this does not configure servers, grant access or enable reviews.\nUse --copilot-review-effort to export the observable Default, Lite or Balanced review effort; Default does not prove an unavailable stored override is absent.\nUse --variable NAME to export only explicitly named non-sensitive repository Actions variables; repeat for each name. Values are plaintext; repository Variables read is required.\nUse --agent-variable NAME to export only explicitly named non-sensitive repository Agents variables; repeat for each name. Values are plaintext; repository Agent variables read is required.\nUse --pages to include Pages configuration or the Branch none deletion declaration; Pages read is required.\nUse --archive to include the repository lifecycle state (requires repository metadata read).\nUse --rulesets to include supported repository branch/tag Rulesets; complete bypass visibility is required.\nUse --actions-policies to include complete repository Actions event policies; Administration write is required.\nUse --actions to include repository Actions execution policy; Administration read is required.\nUse --workflow-permissions to include default workflow token permissions; Administration read is required.\nUse --security secret-scanning to include both repository Secret scanning controls; complete security visibility is required.\nUse --security immutable-releases for the repository immutable release policy; Administration read is required and GET 404 means disabled.\nUse --security dependabot to include Dependabot alerts and security updates; required permissions are a precondition and GET 404 means disabled.\nUse --security codeql-configuration for existing CodeQL default configuration; complete Administration read is required.\nUse --security codeql-default-setup for CodeQL default setup enablement; Administration read is required.\nUse --security ai-scan-pull-requests for the AI Scan pull request toggle; Code scanning alerts read is required.\nUse --security private-vulnerability-reporting for the public-repository reporting toggle; Metadata read is required.\nUse --cache-retention to include cache retention days; Metadata and Administration read are required.\nUse --cache-storage to include the cache size eviction limit; Actions read is required.\nUse --retention to include repository check/run/status/artifact/log retention; Administration read is required.\nUse --fork-approval for the public contributor approval policy; Administration read is required.\nValidate .github/settings.yml locally.\nPlan differences for declared repository settings.\nApply declared repository settings directly. New label, Ruleset and Actions policy IDs are recorded in YAML unless --no-writeback or GITHUB_ACTIONS=true; no Git commit or push is made."); err != nil {
			return fmt.Errorf("write help: %w", err)
		}

		if _, err := fmt.Fprintln(stdout, "Use --environments with export to include complete supported named Environment policies; Actions read is required. Environment names select whole objects for create-or-update, never numeric-ID writeback or unlisted deletion."); err != nil {
			return fmt.Errorf("write help: %w", err)
		}
		return nil
	case "validate":
		_, err := operations.Validate()
		return err
	case "plan":
		result, err := operations.Plan()
		if err != nil {
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
			return createdIDDiagnostic(applyDiagnostic(err, result), result.CreatedIDs)
		}
		createdIDs = result.CreatedIDs
		if result.Completed() {
			data = []byte("Applied repository settings.\n")
		} else {
			data = []byte("No managed repository settings.\n")
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
		return createdIDDiagnostic(fmt.Errorf("write output: %w", err), createdIDs)
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
	default:
		return collection
	}
}

// applyDiagnostic uses recorded operation identity, retaining the original
// error and its uncertainty about earlier or failed writes.
func applyDiagnostic(err error, result app.ApplyResult) error {
	for i := len(result.Operations) - 1; i >= 0; i-- {
		op := result.Operations[i]
		if op.Status != app.OperationFailedRead && op.Status != app.OperationFailedUnknown {
			continue
		}
		path := ""
		switch op.Kind {
		case app.CopilotReviewEffortUpdate:
			path = settings.CopilotReviewEffortPath
		case app.CopilotInstructionsUpdate:
			path = settings.CopilotInstructionsPath
		case app.CopilotMCPUpdate:
			path = settings.CopilotMCPPath
		case app.PullRequestLimitsUpdate:
			path = settings.PullRequestLimitsPath
		case app.AgentVariableUpsert:
			path = "repository.settings.agents.variables"
		case app.VariableUpsert:
			path = "repository.settings.actions.variables"
		case app.RepositoryUpdate:
			path = "repository"
		case app.TopicsUpdate:
			path = "repository.about.topics"
		case app.LabelsRead, app.LabelCreate, app.LabelUpdate, app.LabelDelete:
			path = "repository.labels"
		case app.ActionsPoliciesRead, app.ActionsEventPolicyCreate, app.ActionsEventPolicyUpdate:
			path = "repository.settings.actions.policies"
		case app.RulesetsRead, app.RulesetCreate, app.RulesetUpdate, app.RulesetDelete:
			path = "repository.settings.rulesets"
		case app.EnvironmentsRead, app.EnvironmentUpsert:
			path = "repository.settings.environments"
		case app.ActionsPolicyUpdate, app.WorkflowPermissionsUpdate, app.CacheRetentionRead, app.CacheRetentionUpdate, app.CacheStorageRead, app.CacheStorageUpdate, app.ActionsAccessRead, app.ActionsAccessUpdate, app.RetentionRead, app.RetentionUpdate, app.ForkApprovalRead, app.ForkApprovalUpdate, app.PrivateForkRead, app.PrivateForkUpdate:
			path = "repository.settings.actions.general"
		case app.DependabotUpdate:
			path = presentation.SettingPath(op.SecurityField)
		case app.DependabotRead:
			path = "repository.settings.advanced_security.dependabot"
		case app.SecretScanningUpdate:
			path = presentation.SettingPath(op.SecurityField)
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
