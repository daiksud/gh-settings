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
	NormalizationWarning                              func(settings.ObservationNormalization)
	CommitMessageNormalizationWarning                 func(settings.CommitMessageNormalization)
	SecurityStatusNormalizationWarning                func(settings.SecurityStatusNormalization)
	RulesetNormalizationWarning                       func(int64, string, settings.ObservationNormalization)
	RulesetRequiredApprovalsNormalizationWarning      func(int64, string, settings.RulesetRequiredApprovalsNormalization)
	RulesetMergeMethodsNormalizationWarning           func(int64, string, settings.RulesetMergeMethodsNormalization)
	RulesetTeamApprovalsNormalizationWarning          func(int64, string, settings.RulesetTeamApprovalsNormalization)
	RulesetScanningAlertsNormalizationWarning         func(int64, string, settings.RulesetScanningAlertsNormalization)
	RulesetScanningSecurityAlertsNormalizationWarning func(int64, string, settings.RulesetScanningSecurityAlertsNormalization)
	RulesetBypassModeNormalizationWarning             func(int64, string, settings.RulesetBypassModeNormalization)
	EnvironmentDeploymentNormalizationWarning         func(string, settings.EnvironmentDeploymentNormalization)
	EnvironmentRefTypeNormalizationWarning            func(string, settings.EnvironmentPatternState)
	PrepareSave                                       func() (func(settings.Declaration) error, error)
}

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
	infoError := err
	observe := func(path string, available bool, acquire func(*settings.Declaration) error) {
		candidate := declaration
		err := fmt.Errorf("reader unavailable")
		if available {
			err = acquire(&candidate)
			if err == nil && checkout.Check != nil {
				family, familyErr := exportAcquiredFamily(path, candidate)
				if familyErr != nil {
					err = familyErr
				} else {
					err = checkout.Check(family)
				}
			}
		}
		if err != nil {
			if checkout.Info != nil {
				infoError = errors.Join(infoError, checkout.Info(path, err))
			}
			return
		}
		declaration = candidate
	}
	observe("repository.about.topics", read.Topics != nil, func(candidate *settings.Declaration) error {
		names, err := read.Topics(repo)
		if err != nil {
			return err
		}
		names = settings.SortedTopics(names)
		candidate.Topics.Names = &names
		return nil
	})
	observe("repository.labels", read.Labels != nil, func(candidate *settings.Declaration) error {
		labelObservation, err := read.Labels(repo)
		if err != nil {
			return err
		}
		labels, err := labelObservation.Values()
		if err != nil {
			return err
		}
		labelDeclarations := make([]settings.LabelDeclaration, 0, len(labels))
		for _, label := range labels {
			id, color, description := label.ID, label.Color, label.Description
			labelDeclarations = append(labelDeclarations, settings.LabelDeclaration{ID: &id, Name: label.Name, Color: &color, Description: &description})
		}
		validated, err := settings.NewLabelDeclarations(labelDeclarations)
		if err != nil {
			return err
		}
		candidate.Labels = &validated
		return nil
	})
	observe("repository.settings.actions.policies", read.ActionsPolicies != nil, func(candidate *settings.Declaration) error {
		observed, err := read.ActionsPolicies(repo)
		if err != nil {
			return fmt.Errorf("read Actions policies: %w", err)
		}
		policies, err := settings.ExportActionsPolicies(observed)
		if err != nil {
			return fmt.Errorf("export Actions policies: %w", err)
		}
		candidate.ActionsPolicies = &policies
		return nil
	})
	observe("repository.settings.rulesets", read.Rulesets != nil, func(candidate *settings.Declaration) error {
		observed, err := read.Rulesets(repo)
		if err != nil {
			return err
		}
		desired, err := observed.ExportDeclarations()
		if err != nil {
			return err
		}
		candidate.Rulesets = &desired
		if checkout.RulesetNormalizationWarning != nil || checkout.RulesetRequiredApprovalsNormalizationWarning != nil || checkout.RulesetBypassModeNormalizationWarning != nil || checkout.RulesetMergeMethodsNormalizationWarning != nil || checkout.RulesetTeamApprovalsNormalizationWarning != nil || checkout.RulesetScanningAlertsNormalizationWarning != nil || checkout.RulesetScanningSecurityAlertsNormalizationWarning != nil {
			values, err := observed.Values()
			if err != nil {
				return err
			}
			slices.SortFunc(values, func(a, b settings.RulesetState) int { return strings.Compare(a.Policy.Name, b.Policy.Name) })
			for _, value := range values {
				if record := value.EnforcementNormalization; record != (settings.ObservationNormalization{}) && checkout.RulesetNormalizationWarning != nil {
					checkout.RulesetNormalizationWarning(value.ID, value.Policy.Name, record)
				}
				if record := value.RequiredApprovalsNormalization; record != (settings.RulesetRequiredApprovalsNormalization{}) && checkout.RulesetRequiredApprovalsNormalizationWarning != nil {
					checkout.RulesetRequiredApprovalsNormalizationWarning(value.ID, value.Policy.Name, record)
				}
				if record := value.MergeMethodsNormalization; !record.IsZero() && checkout.RulesetMergeMethodsNormalizationWarning != nil {
					checkout.RulesetMergeMethodsNormalizationWarning(value.ID, value.Policy.Name, record)
				}
				if checkout.RulesetTeamApprovalsNormalizationWarning != nil {
					for _, record := range settings.OrderedRulesetTeamApprovalsNormalizations(value.TeamApprovalsNormalizations) {
						checkout.RulesetTeamApprovalsNormalizationWarning(value.ID, value.Policy.Name, record)
					}
				}
				if checkout.RulesetScanningAlertsNormalizationWarning != nil {
					for _, record := range settings.OrderedRulesetScanningAlertsNormalizations(value.ScanningAlertsNormalizations) {
						checkout.RulesetScanningAlertsNormalizationWarning(value.ID, value.Policy.Name, record)
					}
				}
				if checkout.RulesetScanningSecurityAlertsNormalizationWarning != nil {
					for _, record := range settings.OrderedRulesetScanningSecurityAlertsNormalizations(value.ScanningSecurityAlertsNormalizations) {
						checkout.RulesetScanningSecurityAlertsNormalizationWarning(value.ID, value.Policy.Name, record)
					}
				}
				if record := value.QualitySeverityNormalization; record != (settings.ObservationNormalization{}) && checkout.RulesetNormalizationWarning != nil {
					checkout.RulesetNormalizationWarning(value.ID, value.Policy.Name, record)
				}
				if checkout.RulesetBypassModeNormalizationWarning != nil {
					for _, record := range settings.OrderedRulesetBypassModeNormalizations(value.BypassModeNormalizations) {
						checkout.RulesetBypassModeNormalizationWarning(value.ID, value.Policy.Name, record)
					}
				}
			}
		}
		return nil
	})
	observe("repository.settings.environments", read.Environments != nil, func(candidate *settings.Declaration) error {
		observed, err := read.Environments(repo, nil)
		if err != nil {
			return err
		}
		values, err := observed.Values()
		if err != nil {
			return err
		}
		slices.SortFunc(values, func(a, b settings.EnvironmentState) int {
			return strings.Compare(settings.EnvironmentIdentity(a.Policy.Name), settings.EnvironmentIdentity(b.Policy.Name))
		})
		for _, value := range values {
			if checkout.EnvironmentDeploymentNormalizationWarning != nil {
				if record := value.DeploymentModeNormalization; record != (settings.EnvironmentDeploymentNormalization{}) {
					checkout.EnvironmentDeploymentNormalizationWarning(value.Policy.Name, record)
				}
			}
			if mode := value.Policy.DeploymentBranchPolicy; mode != nil && mode.CustomBranchPolicies && checkout.EnvironmentRefTypeNormalizationWarning != nil {
				patterns := slices.Clone(value.Patterns)
				slices.SortFunc(patterns, func(a, b settings.EnvironmentPatternState) int { return cmp.Compare(a.ID, b.ID) })
				for _, pattern := range patterns {
					if record := pattern.RefTypeNormalization; record != (settings.ObservationNormalization{}) {
						checkout.EnvironmentRefTypeNormalizationWarning(value.Policy.Name, pattern)
					}
				}
			}
		}
		entries := make([]settings.EnvironmentDeclaration, 0, len(values))
		for _, value := range values {
			policy, err := value.EffectivePolicy()
			if err != nil {
				return err
			}
			entries = append(entries, policy)
		}
		d, err := settings.NewEnvironmentDeclarations(entries)
		if err != nil {
			return err
		}
		candidate.Environments = &d
		return nil
	})
	observe("repository.settings.actions.general.actions_permissions", read.Actions != nil, func(candidate *settings.Declaration) error {
		observed, err := read.Actions(repo, true)
		if err != nil {
			return err
		}
		d, err := observed.Declaration()
		if err != nil {
			return err
		}
		candidate.Actions.Enabled, candidate.Actions.AllowedActions, candidate.Actions.SHAPinningRequired, candidate.Actions.SelectedActions = d.Enabled, d.AllowedActions, d.SHAPinningRequired, d.SelectedActions
		if observed.Enabled && observed.AllowedActionsNormalization != (settings.ObservationNormalization{}) && checkout.NormalizationWarning != nil {
			checkout.NormalizationWarning(observed.AllowedActionsNormalization)
		}
		return nil
	})
	observe("repository.settings.actions.general.workflow_permissions", read.WorkflowPermissions != nil, func(candidate *settings.Declaration) error {
		state, err := read.WorkflowPermissions(repo)
		if err != nil {
			return err
		}
		candidate.Actions, err = candidate.Actions.WithWorkflowPermissions(state)
		if err != nil {
			return err
		}
		if state.Normalization.Path != "" && checkout.NormalizationWarning != nil {
			checkout.NormalizationWarning(state.Normalization)
		}
		return nil
	})
	observe("repository.settings.actions.general.artifact_and_log_retention", read.Retention != nil, func(candidate *settings.Declaration) error {
		observed, err := read.Retention(repo)
		if err != nil {
			return err
		}
		retention, err := observed.Declaration()
		if err != nil {
			return err
		}
		candidate.Actions.RetentionDays = retention.RetentionDays
		return nil
	})
	observe("repository.settings.actions.general.cache_retention", read.CacheRetention != nil, func(candidate *settings.Declaration) error {
		observed, err := read.CacheRetention(repo)
		if err != nil {
			return err
		}
		d, err := observed.Declaration()
		if err != nil {
			return err
		}
		candidate.Actions.CacheRetentionDays = d.CacheRetentionDays
		return nil
	})
	observe("repository.settings.actions.general.cache_storage", read.CacheStorage != nil, func(candidate *settings.Declaration) error {
		observed, err := read.CacheStorage(repo)
		if err != nil {
			return err
		}
		d, err := observed.Declaration()
		if err != nil {
			return err
		}
		candidate.Actions.CacheSizeGB = d.CacheSizeGB
		return nil
	})

	observe("repository.settings.actions.general.approval_for_running_fork_pull_request_workflows_from_contributors", read.ForkApproval != nil, func(candidate *settings.Declaration) error {
		o, err := read.ForkApproval(repo)
		if err != nil {
			return err
		}
		v, err := o.Value()
		if err != nil {
			return err
		}
		candidate.Actions.ForkApprovalPolicy = &v
		if normalization := o.Normalization(); normalization.Path != "" && checkout.NormalizationWarning != nil {
			checkout.NormalizationWarning(normalization)
		}
		return nil
	})

	observe("repository.settings.copilot.code_review.use_custom_instructions", read.CopilotInstructions != nil, func(candidate *settings.Declaration) error {
		observed, err := read.CopilotInstructions(repo)
		if err != nil {
			return err
		}
		enabled, err := observed.Value()
		if err != nil {
			return err
		}
		candidate.CopilotInstructions = &enabled
		return nil
	})
	observe("repository.settings.copilot.code_review.allow_copilot_to_use_mcp_tools_when_reviewing_pull_requests", read.CopilotMCP != nil, func(candidate *settings.Declaration) error {
		observed, err := read.CopilotMCP(repo)
		if err != nil {
			return err
		}
		enabled, err := observed.Value()
		if err != nil {
			return err
		}
		candidate.CopilotMCP = &enabled
		return nil
	})
	observe("repository.settings.interaction_limits.pull_request_limits.exemption_list", read.PullRequestExemptions != nil, func(candidate *settings.Declaration) error {
		observed, err := read.PullRequestExemptions(repo)
		if err != nil {
			return err
		}
		logins, err := observed.Logins()
		if err != nil {
			return err
		}
		desired, err := settings.NewPullRequestExemptionsDeclaration(logins)
		if err != nil {
			return err
		}
		candidate.PullRequestExemptions = &desired
		return nil
	})
	observe("repository.settings.interaction_limits.pull_request_limits", read.PullRequestLimits != nil, func(candidate *settings.Declaration) error {
		observed, err := read.PullRequestLimits(repo)
		if err != nil {
			return err
		}
		values, err := observed.Values()
		if err != nil {
			return err
		}
		candidate.PullRequestLimits = settings.PullRequestLimitsDeclaration{Enabled: &values.Enabled, Maximum: &values.Maximum, IncludeDrafts: &values.IncludeDrafts}
		return nil
	})
	observe("repository.settings.copilot.code_review.review_effort_level", read.CopilotReviewEffort != nil, func(candidate *settings.Declaration) error {
		observed, err := read.CopilotReviewEffort(repo)
		if err != nil {
			return err
		}
		enabled, err := observed.Value()
		if err != nil {
			return err
		}
		candidate.CopilotReviewEffort = &enabled
		return nil
	})
	declaration.Security, err = exportSecurity(checkout, read, repo)
	if err != nil {
		return err
	}
	observe("repository.settings.general.danger_zone.archive_this_repository", read.Archive != nil, func(candidate *settings.Declaration) error {
		state, err := read.Archive(repo)
		if err == nil {
			err = state.Validate(repo)
		}
		if err != nil {
			return err
		}
		desired := state.Archived
		candidate.Repository.Archived = &desired
		return nil
	})
	observe("repository.settings.pages", read.Pages != nil, func(candidate *settings.Declaration) error {
		o, err := read.Pages(repo)
		if err != nil {
			return err
		}
		candidate.Pages, err = o.Declaration()
		if err != nil {
			return err
		}
		if record := o.ModeNormalization(); record != (settings.ObservationNormalization{}) && checkout.NormalizationWarning != nil {
			checkout.NormalizationWarning(record)
		}
		if record := o.FolderNormalization(); record != (settings.ObservationNormalization{}) && checkout.NormalizationWarning != nil {
			checkout.NormalizationWarning(record)
		}
		return nil
	})

	observe("repository.settings.actions.variables", read.Variables != nil, func(candidate *settings.Declaration) error {
		inventory, err := read.Variables(repo)
		if err != nil {
			return err
		}
		d, err := exportVariableInventory(inventory)
		if err != nil {
			return err
		}
		candidate.Variables = &d
		return nil
	})
	observe("repository.settings.agents.variables", read.AgentVariables != nil, func(candidate *settings.Declaration) error {
		inventory, err := read.AgentVariables(repo)
		if err != nil {
			return err
		}
		d, err := exportVariableInventory(inventory)
		if err != nil {
			return err
		}
		candidate.AgentsVariables = &d
		return nil
	})
	declaration.EnvironmentVariables, err = exportAllEnvironmentVariables(checkout, read, repo)
	infoError = errors.Join(infoError, err)
	if infoError != nil {
		return infoError
	}
	return save(declaration)
}
