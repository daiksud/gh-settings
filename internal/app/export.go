package app

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

// ExportCheckout defers target and destination resolution until export runs.
// PrepareSave resolves the destination before reads and returns its bound saver.
type ExportCheckout struct {
	Existing settings.Declaration
	Info     func(string, error) error
	Check    func(settings.Declaration) error

	Repository                                        TargetResolver
	NormalizationWarning                              func(settings.ObservationNormalization) error
	CommitMessageNormalizationWarning                 func(settings.CommitMessageNormalization) error
	SecurityStatusNormalizationWarning                func(settings.SecurityStatusNormalization) error
	RulesetNormalizationWarning                       func(int64, string, settings.ObservationNormalization) error
	RulesetRequiredApprovalsNormalizationWarning      func(int64, string, settings.RulesetRequiredApprovalsNormalization) error
	RulesetMergeMethodsNormalizationWarning           func(int64, string, settings.RulesetMergeMethodsNormalization) error
	RulesetTeamApprovalsNormalizationWarning          func(int64, string, settings.RulesetTeamApprovalsNormalization) error
	RulesetScanningAlertsNormalizationWarning         func(int64, string, settings.RulesetScanningAlertsNormalization) error
	RulesetScanningSecurityAlertsNormalizationWarning func(int64, string, settings.RulesetScanningSecurityAlertsNormalization) error
	RulesetBypassModeNormalizationWarning             func(int64, string, settings.RulesetBypassModeNormalization) error
	EnvironmentDeploymentNormalizationWarning         func(string, settings.EnvironmentDeploymentNormalization) error
	EnvironmentRefTypeNormalizationWarning            func(string, settings.EnvironmentPatternState) error
	PrepareSave                                       func() (func(settings.Declaration) error, error)
}

// An exportAssignment copies only the values acquired by one family.
type exportAssignment func(*settings.Declaration)

// Export saves acquired and retained settings after independent family reads.
func Export(checkout ExportCheckout, read Reader) error {
	repo, err := checkout.Repository()
	if err != nil {
		return err
	}
	save, err := checkout.PrepareSave()
	if err != nil {
		return err
	}
	declaration := checkout.Existing
	declaration.Repository, err = exportRepository(checkout, read, repo)
	// Delivery failures must bypass observe's unavailable-family retention path.
	// A projection is never saved after its required diagnostic could not be sent.
	diagnosticError := err
	observe := func(path string, available bool, acquire func() (exportAssignment, error)) {
		var assign exportAssignment
		err := fmt.Errorf("reader unavailable")
		if available {
			assign, err = acquire()
			if err == nil && checkout.Check != nil {
				var acquired settings.Declaration
				assign(&acquired)
				err = checkout.Check(acquired)
			}
		}
		if err != nil {
			if checkout.Info != nil {
				diagnosticError = errors.Join(diagnosticError, checkout.Info(path, err))
			}
			return
		}
		assign(&declaration)
	}
	observe("repository.about.topics", read.Topics != nil, func() (exportAssignment, error) {
		names, err := read.Topics(repo)
		if err != nil {
			return nil, err
		}
		names = settings.SortedTopics(names)
		return func(d *settings.Declaration) { d.Topics.Names = &names }, nil
	})
	observe(settings.WebhooksPath, read.Webhooks != nil, func() (exportAssignment, error) {
		observed, err := read.Webhooks(repo, nil)
		if err != nil {
			return nil, err
		}
		value, err := observed.Declaration()
		return func(d *settings.Declaration) { d.Webhooks = &value }, err
	})
	observe(settings.DeployKeysPath, read.DeployKeys != nil, func() (exportAssignment, error) {
		observed, err := read.DeployKeys(repo)
		if err != nil {
			return nil, err
		}
		value, err := observed.Declaration()
		return func(d *settings.Declaration) { d.DeployKeys = &value }, err
	})
	observe("repository.settings.collaborators", read.Collaborators != nil, func() (exportAssignment, error) {
		observed, err := read.Collaborators(repo)
		if err != nil {
			return nil, err
		}
		desired, err := observed.Declaration()
		return func(d *settings.Declaration) { d.Collaborators = &desired }, err
	})
	observe("repository.labels", read.Labels != nil, func() (exportAssignment, error) {
		labelObservation, err := read.Labels(repo)
		if err != nil {
			return nil, err
		}
		labels, err := labelObservation.Values()
		if err != nil {
			return nil, err
		}
		labelDeclarations := make([]settings.LabelDeclaration, 0, len(labels))
		for _, label := range labels {
			id, color, description := label.ID, label.Color, label.Description
			labelDeclarations = append(labelDeclarations, settings.LabelDeclaration{ID: &id, Name: label.Name, Color: &color, Description: &description})
		}
		validated, err := settings.NewLabelDeclarations(labelDeclarations)
		return func(d *settings.Declaration) { d.Labels = &validated }, err
	})
	observe("repository.settings.actions.policies", read.ActionsPolicies != nil, func() (exportAssignment, error) {
		observed, err := read.ActionsPolicies(repo)
		if err != nil {
			return nil, fmt.Errorf("read Actions policies: %w", err)
		}
		policies, err := settings.ExportActionsPolicies(observed)
		if err != nil {
			return nil, fmt.Errorf("export Actions policies: %w", err)
		}
		return func(d *settings.Declaration) { d.ActionsPolicies = &policies }, nil
	})
	observe(settings.AutolinksPath, read.Autolinks != nil, func() (exportAssignment, error) {
		observed, err := read.Autolinks(repo)
		if err != nil {
			return nil, err
		}
		values, err := settings.ExportAutolinks(observed)
		return func(d *settings.Declaration) { d.Autolinks = &values }, err
	})
	observe(settings.ClassicProtectionPath, read.BranchProtectionRules != nil, func() (exportAssignment, error) {
		observed, err := read.BranchProtectionRules(repo)
		if err != nil {
			return nil, err
		}
		rules, err := settings.ExportClassicRules(observed)
		return func(d *settings.Declaration) { d.BranchProtectionRules = &rules }, err
	})
	observe("repository.settings.rulesets", read.Rulesets != nil, func() (exportAssignment, error) {
		observed, err := read.Rulesets(repo)
		if err != nil {
			return nil, err
		}
		desired, err := observed.ExportDeclarations()
		if err != nil {
			return nil, err
		}
		if checkout.RulesetNormalizationWarning != nil || checkout.RulesetRequiredApprovalsNormalizationWarning != nil || checkout.RulesetBypassModeNormalizationWarning != nil || checkout.RulesetMergeMethodsNormalizationWarning != nil || checkout.RulesetTeamApprovalsNormalizationWarning != nil || checkout.RulesetScanningAlertsNormalizationWarning != nil || checkout.RulesetScanningSecurityAlertsNormalizationWarning != nil {
			values, err := observed.Values()
			if err != nil {
				return nil, err
			}
			slices.SortFunc(values, func(a, b settings.RulesetState) int { return strings.Compare(a.Policy.Name, b.Policy.Name) })
			for _, value := range values {
				if record := value.EnforcementNormalization; record != (settings.ObservationNormalization{}) && checkout.RulesetNormalizationWarning != nil {
					diagnosticError = errors.Join(diagnosticError, checkout.RulesetNormalizationWarning(value.ID, value.Policy.Name, record))
				}
				if record := value.RequiredApprovalsNormalization; record != (settings.RulesetRequiredApprovalsNormalization{}) && checkout.RulesetRequiredApprovalsNormalizationWarning != nil {
					diagnosticError = errors.Join(diagnosticError, checkout.RulesetRequiredApprovalsNormalizationWarning(value.ID, value.Policy.Name, record))
				}
				if record := value.MergeMethodsNormalization; !record.IsZero() && checkout.RulesetMergeMethodsNormalizationWarning != nil {
					diagnosticError = errors.Join(diagnosticError, checkout.RulesetMergeMethodsNormalizationWarning(value.ID, value.Policy.Name, record))
				}
				if checkout.RulesetTeamApprovalsNormalizationWarning != nil {
					for _, record := range settings.OrderedRulesetTeamApprovalsNormalizations(value.TeamApprovalsNormalizations) {
						diagnosticError = errors.Join(diagnosticError, checkout.RulesetTeamApprovalsNormalizationWarning(value.ID, value.Policy.Name, record))
					}
				}
				if checkout.RulesetScanningAlertsNormalizationWarning != nil {
					for _, record := range settings.OrderedRulesetScanningAlertsNormalizations(value.ScanningAlertsNormalizations) {
						diagnosticError = errors.Join(diagnosticError, checkout.RulesetScanningAlertsNormalizationWarning(value.ID, value.Policy.Name, record))
					}
				}
				if checkout.RulesetScanningSecurityAlertsNormalizationWarning != nil {
					for _, record := range settings.OrderedRulesetScanningSecurityAlertsNormalizations(value.ScanningSecurityAlertsNormalizations) {
						diagnosticError = errors.Join(diagnosticError, checkout.RulesetScanningSecurityAlertsNormalizationWarning(value.ID, value.Policy.Name, record))
					}
				}
				if record := value.QualitySeverityNormalization; record != (settings.ObservationNormalization{}) && checkout.RulesetNormalizationWarning != nil {
					diagnosticError = errors.Join(diagnosticError, checkout.RulesetNormalizationWarning(value.ID, value.Policy.Name, record))
				}
				if checkout.RulesetBypassModeNormalizationWarning != nil {
					for _, record := range settings.OrderedRulesetBypassModeNormalizations(value.BypassModeNormalizations) {
						diagnosticError = errors.Join(diagnosticError, checkout.RulesetBypassModeNormalizationWarning(value.ID, value.Policy.Name, record))
					}
				}
			}
		}
		return func(d *settings.Declaration) { d.Rulesets = &desired }, nil
	})
	observe("repository.settings.environments", read.Environments != nil, func() (exportAssignment, error) {
		observed, err := read.Environments(repo, nil)
		if err != nil {
			return nil, err
		}
		values, err := observed.Values()
		if err != nil {
			return nil, err
		}
		slices.SortFunc(values, func(a, b settings.EnvironmentState) int {
			return strings.Compare(settings.EnvironmentIdentity(a.Policy.Name), settings.EnvironmentIdentity(b.Policy.Name))
		})
		for _, value := range values {
			if checkout.EnvironmentDeploymentNormalizationWarning != nil {
				if record := value.DeploymentModeNormalization; record != (settings.EnvironmentDeploymentNormalization{}) {
					diagnosticError = errors.Join(diagnosticError, checkout.EnvironmentDeploymentNormalizationWarning(value.Policy.Name, record))
				}
			}
			if mode := value.Policy.DeploymentBranchPolicy; mode != nil && mode.CustomBranchPolicies && checkout.EnvironmentRefTypeNormalizationWarning != nil {
				patterns := slices.Clone(value.Patterns)
				slices.SortFunc(patterns, func(a, b settings.EnvironmentPatternState) int { return cmp.Compare(a.ID, b.ID) })
				for _, pattern := range patterns {
					if record := pattern.RefTypeNormalization; record != (settings.ObservationNormalization{}) {
						diagnosticError = errors.Join(diagnosticError, checkout.EnvironmentRefTypeNormalizationWarning(value.Policy.Name, pattern))
					}
				}
			}
		}
		entries := make([]settings.EnvironmentDeclaration, 0, len(values))
		for _, value := range values {
			policy, err := value.EffectivePolicy()
			if err != nil {
				return nil, err
			}
			entries = append(entries, policy)
		}
		environments, err := settings.NewEnvironmentDeclarations(entries)
		return func(d *settings.Declaration) { d.Environments = &environments }, err
	})
	observe("repository.settings.actions.general.actions_permissions", read.Actions != nil, func() (exportAssignment, error) {
		observed, err := read.Actions(repo, true)
		if err != nil {
			return nil, err
		}
		values, err := observed.Declaration()
		if err != nil {
			return nil, err
		}
		if observed.Enabled && observed.AllowedActionsNormalization != (settings.ObservationNormalization{}) && checkout.NormalizationWarning != nil {
			diagnosticError = errors.Join(diagnosticError, checkout.NormalizationWarning(observed.AllowedActionsNormalization))
		}
		return func(d *settings.Declaration) {
			d.Actions.Enabled = values.Enabled
			d.Actions.AllowedActions = values.AllowedActions
			d.Actions.SHAPinningRequired = values.SHAPinningRequired
			d.Actions.SelectedActions = values.SelectedActions
		}, nil
	})
	observe("repository.settings.actions.general.workflow_permissions", read.WorkflowPermissions != nil, func() (exportAssignment, error) {
		state, err := read.WorkflowPermissions(repo)
		if err != nil {
			return nil, err
		}
		values, err := (settings.ActionsPolicyDeclaration{}).WithWorkflowPermissions(state)
		if err != nil {
			return nil, err
		}
		if state.Normalization.Path != "" && checkout.NormalizationWarning != nil {
			diagnosticError = errors.Join(diagnosticError, checkout.NormalizationWarning(state.Normalization))
		}
		return func(d *settings.Declaration) {
			d.Actions.DefaultWorkflowPermissions = values.DefaultWorkflowPermissions
			d.Actions.CanApprovePullRequestReviews = values.CanApprovePullRequestReviews
		}, nil
	})
	observe("repository.settings.actions.general.artifact_and_log_retention", read.Retention != nil, func() (exportAssignment, error) {
		observed, err := read.Retention(repo)
		if err != nil {
			return nil, err
		}
		retention, err := observed.Declaration()
		return func(d *settings.Declaration) { d.Actions.RetentionDays = retention.RetentionDays }, err
	})
	observe("repository.settings.actions.general.cache_retention", read.CacheRetention != nil, func() (exportAssignment, error) {
		observed, err := read.CacheRetention(repo)
		if err != nil {
			return nil, err
		}
		values, err := observed.Declaration()
		return func(d *settings.Declaration) { d.Actions.CacheRetentionDays = values.CacheRetentionDays }, err
	})
	observe("repository.settings.actions.general.cache_storage", read.CacheStorage != nil, func() (exportAssignment, error) {
		observed, err := read.CacheStorage(repo)
		if err != nil {
			return nil, err
		}
		values, err := observed.Declaration()
		return func(d *settings.Declaration) { d.Actions.CacheSizeGB = values.CacheSizeGB }, err
	})

	observe("repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors", read.ForkApproval != nil, func() (exportAssignment, error) {
		o, err := read.ForkApproval(repo)
		if err != nil {
			return nil, err
		}
		v, err := o.Value()
		if err != nil {
			return nil, err
		}
		if normalization := o.Normalization(); normalization.Path != "" && checkout.NormalizationWarning != nil {
			diagnosticError = errors.Join(diagnosticError, checkout.NormalizationWarning(normalization))
		}
		return func(d *settings.Declaration) { d.Actions.ForkApprovalPolicy = &v }, nil
	})

	observe("repository.settings.copilot.code_review.use_custom_instructions", read.CopilotInstructions != nil, func() (exportAssignment, error) {
		observed, err := read.CopilotInstructions(repo)
		if err != nil {
			return nil, err
		}
		enabled, err := observed.Value()
		return func(d *settings.Declaration) { d.CopilotInstructions = &enabled }, err
	})
	observe(settings.RunnersPath, read.AllRunners != nil, func() (exportAssignment, error) {
		observed, err := read.AllRunners(repo)
		if err != nil {
			return nil, err
		}
		values, err := observed.Values()
		return func(d *settings.Declaration) { d.Runners = &values }, err
	})
	declaration.OIDCSubject, err = exportOIDCSubject(checkout, read, repo, declaration.OIDCSubject)
	diagnosticError = errors.Join(diagnosticError, err)
	observe("repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests", read.CopilotMCP != nil, func() (exportAssignment, error) {
		observed, err := read.CopilotMCP(repo)
		if err != nil {
			return nil, err
		}
		enabled, err := observed.Value()
		return func(d *settings.Declaration) { d.CopilotMCP = &enabled }, err
	})
	observe(settings.PullRequestExemptionsPath, read.PullRequestExemptions != nil, func() (exportAssignment, error) {
		observed, err := read.PullRequestExemptions(repo)
		if err != nil {
			return nil, err
		}
		logins, err := observed.Logins()
		if err != nil {
			return nil, err
		}
		desired, err := settings.NewPullRequestExemptionsDeclaration(logins)
		return func(d *settings.Declaration) { d.PullRequestExemptions = &desired }, err
	})
	observe(settings.PullRequestLimitsPath, read.PullRequestLimits != nil, func() (exportAssignment, error) {
		observed, err := read.PullRequestLimits(repo)
		if err != nil {
			return nil, err
		}
		values, err := observed.Values()
		return func(d *settings.Declaration) {
			d.PullRequestLimits = settings.PullRequestLimitsDeclaration{Enabled: &values.Enabled, Maximum: &values.Maximum, IncludeDrafts: &values.IncludeDrafts}
		}, err
	})
	observe("repository.settings.copilot.code_review.review_effort_level", read.CopilotReviewEffort != nil, func() (exportAssignment, error) {
		observed, err := read.CopilotReviewEffort(repo)
		if err != nil {
			return nil, err
		}
		enabled, err := observed.Value()
		return func(d *settings.Declaration) { d.CopilotReviewEffort = &enabled }, err
	})
	declaration.Security, err = exportSecurity(checkout, read, repo)
	if err != nil {
		return errors.Join(diagnosticError, err)
	}
	observe("repository.settings.general.danger_zone.archive_this_repository", read.Archive != nil, func() (exportAssignment, error) {
		state, err := read.Archive(repo)
		if err == nil {
			err = state.Validate(repo)
		}
		if err != nil {
			return nil, err
		}
		desired := state.Archived
		return func(d *settings.Declaration) { d.Repository.Archived = &desired }, nil
	})
	observe("repository.settings.pages", read.Pages != nil, func() (exportAssignment, error) {
		o, err := read.Pages(repo)
		if err != nil {
			return nil, err
		}
		pages, err := o.Declaration()
		if err != nil {
			return nil, err
		}
		if record := o.ModeNormalization(); record != (settings.ObservationNormalization{}) && checkout.NormalizationWarning != nil {
			diagnosticError = errors.Join(diagnosticError, checkout.NormalizationWarning(record))
		}
		if record := o.FolderNormalization(); record != (settings.ObservationNormalization{}) && checkout.NormalizationWarning != nil {
			diagnosticError = errors.Join(diagnosticError, checkout.NormalizationWarning(record))
		}
		return func(d *settings.Declaration) { d.Pages = pages }, nil
	})

	observe(settings.ActionsVariablesPath, read.Variables != nil, func() (exportAssignment, error) {
		inventory, err := read.Variables(repo)
		if err != nil {
			return nil, err
		}
		values, err := exportVariableInventory(inventory)
		return func(d *settings.Declaration) { d.Variables = &values }, err
	})
	observe(settings.AgentsVariablesPath, read.AgentVariables != nil, func() (exportAssignment, error) {
		inventory, err := read.AgentVariables(repo)
		if err != nil {
			return nil, err
		}
		values, err := exportVariableInventory(inventory)
		return func(d *settings.Declaration) { d.AgentsVariables = &values }, err
	})
	declaration.EnvironmentVariables, err = exportAllEnvironmentVariables(checkout, read, repo)
	diagnosticError = errors.Join(diagnosticError, err)
	if diagnosticError != nil {
		return diagnosticError
	}
	return save(declaration)
}
