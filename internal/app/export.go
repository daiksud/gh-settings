package app

import (
	"cmp"
	"slices"
	"strings"

	"github.com/daiksud/gh-settings/internal/settings"
)

// ExportCheckout defers target and destination resolution until export runs.
// PrepareSave resolves the destination before reads and returns its bound saver.
type ExportCheckout struct {
	IncludeArchive                               bool
	IncludePages                                 bool
	IncludeForkApproval                          bool
	IncludePrivateFork                           bool
	Repository                                   TargetResolver
	IncludeActions                               bool
	IncludeWorkflowPermissions                   bool
	IncludeRulesets                              bool
	IncludeEnvironments                          bool
	IncludeSecretScanning                        bool
	IncludePrivateReporting                      bool
	IncludeImmutableReleases                     bool
	IncludeRetention                             bool
	ExcludedRulesets                             func(int)
	NormalizationWarning                         func(settings.ObservationNormalization)
	CommitMessageNormalizationWarning            func(settings.CommitMessageNormalization)
	SecurityStatusNormalizationWarning           func(settings.SecurityStatusNormalization)
	RulesetNormalizationWarning                  func(int64, string, settings.ObservationNormalization)
	RulesetRequiredApprovalsNormalizationWarning func(int64, string, settings.RulesetRequiredApprovalsNormalization)
	RulesetMergeMethodsNormalizationWarning      func(int64, string, settings.RulesetMergeMethodsNormalization)
	RulesetTeamApprovalsNormalizationWarning     func(int64, string, settings.RulesetTeamApprovalsNormalization)
	RulesetScanningAlertsNormalizationWarning    func(int64, string, settings.RulesetScanningAlertsNormalization)
	RulesetBypassModeNormalizationWarning        func(int64, string, settings.RulesetBypassModeNormalization)
	EnvironmentDeploymentNormalizationWarning    func(string, settings.EnvironmentDeploymentNormalization)
	EnvironmentRefTypeNormalizationWarning       func(string, settings.EnvironmentPatternState)
	PrepareSave                                  func() (func(settings.Declaration) error, error)
}

// Export saves all observed fields only after a complete read.
func Export(checkout ExportCheckout, read Reader) error {
	repo, err := checkout.Repository()
	if err != nil {
		return err
	}
	save, err := checkout.PrepareSave()
	if err != nil {
		return err
	}
	observed, err := read.ObserveRepository(repo, settings.AllRepositoryFields())
	if err != nil {
		return err
	}
	repositoryDeclaration, err := observed.Declaration()
	if err != nil {
		return err
	}
	if checkout.NormalizationWarning != nil {
		for _, normalization := range observed.CreationPolicyNormalizations() {
			if normalization.Path == settings.IssuePermissionsControl.Path() && !*repositoryDeclaration.HasIssues {
				continue
			}
			if normalization.Path == settings.PullRequestPermissionsControl.Path() && !*repositoryDeclaration.HasPullRequests {
				continue
			}
			checkout.NormalizationWarning(normalization)
		}
	}
	if checkout.CommitMessageNormalizationWarning != nil {
		for _, normalization := range observed.CommitMessageNormalizations() {
			if normalization.Path == settings.SquashCommitMenu.Path() && !*repositoryDeclaration.AllowSquashMerge {
				continue
			}
			if normalization.Path == settings.MergeCommitMenu.Path() && !*repositoryDeclaration.AllowMergeCommit {
				continue
			}
			checkout.CommitMessageNormalizationWarning(normalization)
		}
	}
	declaration := settings.Declaration{Repository: repositoryDeclaration}
	names, err := read.Topics(repo)
	if err != nil {
		return err
	}
	names = settings.SortedTopics(names)
	declaration.Topics.Names = &names
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
	declaration.Labels = &validated
	if checkout.IncludeRulesets {
		observed, err := read.Rulesets(repo)
		if err != nil {
			return err
		}
		desired, err := observed.ExportDeclarations()
		if err != nil {
			return err
		}
		declaration.Rulesets = &desired
		if checkout.RulesetNormalizationWarning != nil || checkout.RulesetRequiredApprovalsNormalizationWarning != nil || checkout.RulesetBypassModeNormalizationWarning != nil || checkout.RulesetMergeMethodsNormalizationWarning != nil || checkout.RulesetTeamApprovalsNormalizationWarning != nil || checkout.RulesetScanningAlertsNormalizationWarning != nil {
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
		if observed.ExcludedPush() > 0 && checkout.ExcludedRulesets != nil {
			checkout.ExcludedRulesets(observed.ExcludedPush())
		}
	}
	if checkout.IncludeEnvironments {
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
		declaration.Environments = &d
	}
	if checkout.IncludeActions {
		observed, err := read.Actions(repo, true)
		if err != nil {
			return err
		}
		declaration.Actions, err = observed.Declaration()
		if err != nil {
			return err
		}
		if observed.Enabled && observed.AllowedActionsNormalization != (settings.ObservationNormalization{}) && checkout.NormalizationWarning != nil {
			checkout.NormalizationWarning(observed.AllowedActionsNormalization)
		}
	}
	if checkout.IncludeWorkflowPermissions {
		state, err := read.WorkflowPermissions(repo)
		if err != nil {
			return err
		}
		declaration.Actions, err = declaration.Actions.WithWorkflowPermissions(state)
		if err != nil {
			return err
		}
		if state.Normalization.Path != "" && checkout.NormalizationWarning != nil {
			checkout.NormalizationWarning(state.Normalization)
		}
	}
	if checkout.IncludeRetention {
		observed, err := read.Retention(repo)
		if err != nil {
			return err
		}
		retention, err := observed.Declaration()
		if err != nil {
			return err
		}
		declaration.Actions.RetentionDays = retention.RetentionDays
	}
	if checkout.IncludeForkApproval {
		o, err := read.ForkApproval(repo)
		if err != nil {
			return err
		}
		v, err := o.Value()
		if err != nil {
			return err
		}
		declaration.Actions.ForkApprovalPolicy = &v
		if normalization := o.Normalization(); normalization.Path != "" && checkout.NormalizationWarning != nil {
			checkout.NormalizationWarning(normalization)
		}
	}
	if checkout.IncludePrivateFork {
		o, err := read.PrivateFork(repo)
		if err != nil {
			return err
		}
		d, err := o.Declaration()
		if err != nil {
			return err
		}
		declaration.Actions.PrivateFork = d
	}
	if checkout.IncludeSecretScanning {
		observed, err := read.SecretScanning(repo)
		if err != nil {
			return err
		}
		if _, _, err := observed.Values(); err != nil {
			return err
		}
		if checkout.SecurityStatusNormalizationWarning != nil {
			for _, record := range observed.Normalizations() {
				checkout.SecurityStatusNormalizationWarning(record)
			}
		}
		declaration.Security, err = observed.Declaration()
		if err != nil {
			return err
		}
	}
	if checkout.IncludePrivateReporting {
		observed, err := read.PrivateReporting(repo)
		if err != nil {
			return err
		}
		enabled, err := observed.Value()
		if err != nil {
			return err
		}
		declaration.Security.PrivateVulnerabilityReporting = &enabled
	}
	if checkout.IncludeImmutableReleases {
		observed, err := read.ImmutableReleases(repo)
		if err != nil {
			return err
		}
		enabled, _, _, err := observed.Values()
		if err != nil {
			return err
		}
		declaration.Security.ImmutableReleases = &enabled
	}
	if checkout.IncludeArchive {
		state, err := read.Archive(repo)
		if err == nil {
			err = state.Validate(repo)
		}
		if err != nil {
			return err
		}
		desired := state.Archived
		declaration.Repository.Archived = &desired
	}
	if checkout.IncludePages {
		o, err := read.Pages(repo)
		if err != nil {
			return err
		}
		declaration.Pages, err = o.Declaration()
		if err != nil {
			return err
		}
		if record := o.ModeNormalization(); record != (settings.ObservationNormalization{}) && checkout.NormalizationWarning != nil {
			checkout.NormalizationWarning(record)
		}
		if record := o.FolderNormalization(); record != (settings.ObservationNormalization{}) && checkout.NormalizationWarning != nil {
			checkout.NormalizationWarning(record)
		}
	}
	return save(declaration)
}
