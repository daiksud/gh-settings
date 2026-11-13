package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestExportWarningFailurePreventsSave(t *testing.T) {
	tests := []struct {
		name    string
		install func(*ExportCheckout, func() error)
	}{
		{"commit message", func(c *ExportCheckout, fail func() error) {
			c.CommitMessageNormalizationWarning = func(settings.CommitMessageNormalization) error { return fail() }
		}},
		{"security", func(c *ExportCheckout, fail func() error) {
			c.SecurityStatusNormalizationWarning = func(settings.SecurityStatusNormalization) error { return fail() }
		}},
		{"ruleset enforcement", func(c *ExportCheckout, fail func() error) {
			c.RulesetNormalizationWarning = func(_ int64, _ string, n settings.ObservationNormalization) error {
				if n.Path == settings.RulesetEnforcementPath {
					return fail()
				}
				return nil
			}
		}},
		{"ruleset quality", func(c *ExportCheckout, fail func() error) {
			c.RulesetNormalizationWarning = func(_ int64, _ string, n settings.ObservationNormalization) error {
				if n.Path == settings.RulesetQualitySeverityPath {
					return fail()
				}
				return nil
			}
		}},
		{"ruleset approvals", func(c *ExportCheckout, fail func() error) {
			c.RulesetRequiredApprovalsNormalizationWarning = func(int64, string, settings.RulesetRequiredApprovalsNormalization) error { return fail() }
		}},
		{"ruleset merge methods", func(c *ExportCheckout, fail func() error) {
			c.RulesetMergeMethodsNormalizationWarning = func(int64, string, settings.RulesetMergeMethodsNormalization) error { return fail() }
		}},

		{"ruleset scanning alerts", func(c *ExportCheckout, fail func() error) {
			c.RulesetScanningAlertsNormalizationWarning = func(int64, string, settings.RulesetScanningAlertsNormalization) error { return fail() }
		}},
		{"ruleset scanning security", func(c *ExportCheckout, fail func() error) {
			c.RulesetScanningSecurityAlertsNormalizationWarning = func(int64, string, settings.RulesetScanningSecurityAlertsNormalization) error { return fail() }
		}},
		{"ruleset bypass", func(c *ExportCheckout, fail func() error) {
			c.RulesetBypassModeNormalizationWarning = func(int64, string, settings.RulesetBypassModeNormalization) error { return fail() }
		}},
		{"environment deployment", func(c *ExportCheckout, fail func() error) {
			c.EnvironmentDeploymentNormalizationWarning = func(string, settings.EnvironmentDeploymentNormalization) error { return fail() }
		}},
		{"environment ref type", func(c *ExportCheckout, fail func() error) {
			c.EnvironmentRefTypeNormalizationWarning = func(string, settings.EnvironmentPatternState) error { return fail() }
		}},
	}
	for _, path := range []string{settings.IssuePermissionsControl.Path(), settings.PullRequestPermissionsControl.Path(), settings.ActionsPermissionsPath, settings.WorkflowPermissionsPath, settings.ForkApprovalPath, settings.PagesModePath, settings.PagesFolderPath} {
		tests = append(tests, struct {
			name    string
			install func(*ExportCheckout, func() error)
		}{path, func(c *ExportCheckout, fail func() error) {
			c.NormalizationWarning = func(n settings.ObservationNormalization) error {
				if n.Path == path {
					return fail()
				}
				return nil
			}
		}})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cause := errors.New("warning destination unavailable")
			calls, saves := 0, 0
			c := ExportCheckout{Repository: checkoutRepository, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(settings.Declaration) error { saves++; return nil }, nil
			}, Info: func(_ string, err error) error {
				if errors.Is(err, cause) {
					t.Fatal("warning failure downgraded to unavailable observation")
				}
				return nil
			}}
			test.install(&c, func() error { calls++; return cause })
			err := Export(c, warningFailureReader(t))
			if !errors.Is(err, cause) || calls == 0 || saves != 0 {
				t.Fatal("lost warning error or saved projection", err, calls, saves)
			}
		})
	}
}

func TestExportWarningFailuresSurviveLaterDiagnostics(t *testing.T) {
	for _, securityFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "projection info succeeds", true: "security info fails"}[securityFailure], func(t *testing.T) {
			first, second := errors.New("early warning failed"), errors.New("later diagnostic failed")
			saved, projectionInfo := false, false
			c := ExportCheckout{Repository: checkoutRepository,
				CommitMessageNormalizationWarning:            func(settings.CommitMessageNormalization) error { return first },
				RulesetRequiredApprovalsNormalizationWarning: func(int64, string, settings.RulesetRequiredApprovalsNormalization) error { return second },
				Check: func(d settings.Declaration) error {
					if d.Rulesets != nil {
						return errors.New("unsupported projected policy")
					}
					return nil
				},
				Info: func(path string, cause error) error {
					if errors.Is(cause, first) || errors.Is(cause, second) {
						t.Fatal("delivery failure became acquisition info")
					}
					if path == "repository.settings.rulesets" {
						projectionInfo = true
					}
					if securityFailure && path == "repository.settings.advanced_security.dependabot" {
						return second
					}
					return nil
				},
				PrepareSave: func() (func(settings.Declaration) error, error) {
					return func(settings.Declaration) error { saved = true; return nil }, nil
				},
			}
			err := Export(c, warningFailureReader(t))
			if !errors.Is(err, first) || !errors.Is(err, second) || saved || !projectionInfo {
				t.Fatal("diagnostic cause lost across helpers", err, saved, projectionInfo)
			}
		})
	}
}

func warningFailureReader(t *testing.T) Reader {
	t.Helper()
	base := settings.RepositoryState{DefaultBranch: "main", HasIssues: true}
	r := exportFixtureReader(t, base, func(string, repository.Repository) error { return nil })
	merge := settings.MergeMethodsState{AllowMergeCommit: true, MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE", SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitNormalization: settings.CommitMessageNormalization{Path: settings.MergeCommitMenu.Path(), RawTitle: "MERGE_MESSAGE", RawMessage: "BLANK", Fallback: "default_message"}}
	pr := settings.PRAccessState{HasPullRequests: true, PullRequestCreationPolicy: "ALL", Normalization: settings.ObservationNormalization{Path: settings.PullRequestPermissionsControl.Path(), RawValue: "FUTURE", Fallback: "all_users"}}
	issue := settings.IssueCreationPolicyState{Policy: "ALL", Normalization: settings.ObservationNormalization{Path: settings.IssuePermissionsControl.Path(), RawValue: "FUTURE", Fallback: "all_users"}}
	o := settings.NewRepositoryObservation(&base, &merge, &pr).WithIssueCreationPolicy(issue).WithSponsorships(settings.SponsorshipsState{})
	r.ObserveRepository = func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) { return o, nil }
	r.Actions = func(repository.Repository, bool) (settings.ActionsPolicyState, error) {
		return settings.ActionsPolicyState{Enabled: true, AllowedActions: "all", AllowedActionsNormalization: settings.ObservationNormalization{Path: settings.ActionsPermissionsPath, RawValue: "FUTURE", Fallback: string(settings.DefaultActionsPermissionsChoice)}}, nil
	}
	r.WorkflowPermissions = func(repository.Repository) (settings.WorkflowPermissionsState, error) {
		return github.NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "FUTURE"}), nil
	}
	r.ForkApproval = func(repository.Repository) (settings.ForkApprovalObservation, error) {
		return settings.NewForkApprovalObservationWithNormalization("first_time_contributors", settings.ObservationNormalization{Path: settings.ForkApprovalPath, RawValue: "FUTURE", Fallback: string(settings.DefaultForkApprovalChoice)})
	}
	r.SecretScanning = func(repository.Repository) (settings.SecretScanningObservation, error) {
		return settings.NewSecretScanningObservationWithNormalizations(false, false, []settings.SecurityStatusNormalization{{Path: settings.SecretProtectionControl.Path(), RawValue: "FUTURE", Fallback: false}})
	}
	r.Pages = func(repository.Repository) (settings.PagesObservation, error) {
		return settings.NewPagesObservationWithModeNormalization(settings.PagesState{BuildType: "legacy", Source: &settings.PagesSource{Branch: "main", Path: "/"}}, settings.ObservationNormalization{Path: settings.PagesModePath, RawValue: "FUTURE", Fallback: "deploy_from_a_branch"}, settings.ObservationNormalization{Path: settings.PagesFolderPath, RawValue: "/future", Fallback: "root"})
	}
	r.Environments = func(repository.Repository, []string) (settings.EnvironmentObservation, error) {
		return settings.NewEnvironmentObservation([]settings.EnvironmentState{
			exportEnvironmentModeState(t, 1, "mode", false),
			exportEnvironmentRefTypeState(t, 2, "patterns", []settings.EnvironmentPatternState{{ID: 3, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "FUTURE", Fallback: "branch"}}}),
		})
	}
	state := exportApprovalsState(t, 7, "policy", 11, true)
	p := (*state.Policy.Rules)[0].Parameters
	p.AllowedMergeMethods = &[]string{"merge", "squash", "rebase"}
	state.MergeMethodsNormalization = settings.RulesetMergeMethodsNormalization{Path: settings.RulesetMergeMethodsPath, RawValue: []string{"FUTURE"}, Fallback: []string{"merge", "squash", "rebase"}}

	quality := exportQualityState(t, 7, "policy", "FUTURE", false)
	*state.Policy.Rules = append(*state.Policy.Rules, (*quality.Policy.Rules)[0], settings.RulesetRule{Type: "code_scanning", Parameters: &settings.RuleParameters{CodeScanningTools: &[]settings.CodeScanningTool{{Tool: "CodeQL", AlertsThreshold: "errors", SecurityAlertsThreshold: "high_or_higher"}}}})
	state.QualitySeverityNormalization = quality.QualitySeverityNormalization
	state.ScanningAlertsNormalizations = []settings.RulesetScanningAlertsNormalization{{Tool: "CodeQL", Path: settings.RulesetScanningAlertsPath, RawValue: "FUTURE", Fallback: "errors"}}
	state.ScanningSecurityAlertsNormalizations = []settings.RulesetScanningSecurityAlertsNormalization{{Tool: "CodeQL", Path: settings.RulesetScanningSecurityAlertsPath, RawValue: "FUTURE", Fallback: "high_or_higher"}}
	bypass := exportBypassState(t, 7, "policy", true)
	state.Policy.BypassActors = bypass.Policy.BypassActors
	state.BypassModeNormalizations = bypass.BypassModeNormalizations
	rulesets, err := settings.NewRulesetObservation([]settings.RulesetState{state}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rulesets.ExportDeclarations(); err != nil {
		t.Fatal(err)
	}
	r.Rulesets = func(repository.Repository) (settings.RulesetObservation, error) { return rulesets, nil }
	return r
}
