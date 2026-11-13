package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func exportPrior(t *testing.T) settings.Declaration {
	t.Helper()
	yes, no := true, false
	days, gb := 7, 10
	permission, approval, effort := "read", "all_external_contributors", "balanced"
	vars, _ := settings.NewVariableDeclarations([]settings.RepositoryVariable{{Name: "KEEP", Value: "private-value"}})
	envs, _ := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "prod", WaitTimer: &days}})
	rules := appRulesets(t, `[]`)
	labels := appLabelDeclarations(t)
	exemptions, _ := settings.NewPullRequestExemptionsDeclaration([]string{"someone"})
	return settings.Declaration{Topics: settings.TopicsDeclaration{Names: &[]string{"keep"}}, Labels: &labels, Rulesets: rules, Environments: &envs, Variables: &vars, AgentsVariables: &vars, EnvironmentVariables: []settings.EnvironmentVariableDeclaration{{Environment: "prod", Variables: vars}}, ActionsPolicies: eventPolicyDeclaration(t, eventAppPolicy(7, "keep")), CopilotInstructions: &yes, CopilotMCP: &yes, CopilotReviewEffort: &effort, PullRequestExemptions: &exemptions, PullRequestLimits: settings.PullRequestLimitsDeclaration{Enabled: &yes}, Actions: settings.ActionsPolicyDeclaration{Enabled: &yes, AllowedActions: appString("all"), SHAPinningRequired: &no, DefaultWorkflowPermissions: &permission, CanApprovePullRequestReviews: &no, CacheRetentionDays: &days, CacheSizeGB: &gb, ForkApprovalPolicy: &approval, RetentionDays: &days}, Pages: settings.PagesDeclaration{}, Security: settings.SecurityDeclaration{CodeQLDefaultSetup: &yes}, Repository: settings.RepositoryDeclaration{Archived: &yes}}
}
func appString(s string) *string { return &s }

// Every independent reader must retain prior intent both on an explicit error
// and an unacquired zero observation, with no normalization from failed reads.
func TestReadableExportIndependentFailuresRetainIntent(t *testing.T) {
	fields := []string{"Topics", "Labels", "ActionsPolicies", "Rulesets", "Environments", "Actions", "WorkflowPermissions", "Retention", "CacheRetention", "CacheStorage", "ForkApproval", "CopilotInstructions", "CopilotMCP", "CopilotReviewEffort", "PullRequestExemptions", "PullRequestLimits", "Archive", "Pages", "Variables", "AgentVariables", "EnvironmentNames", "EnvironmentVariables", "CodeQLForExport"}
	for _, field := range fields {
		for _, zero := range []bool{false, true} {
			if zero && (field == "Topics" || field == "EnvironmentNames") {
				continue
			}
			t.Run(field+map[bool]string{false: "/error", true: "/unacquired"}[zero], func(t *testing.T) {
				prior := exportPrior(t)
				run := func(reader Reader, info func(string, error) error) (settings.Declaration, error) {
					var got settings.Declaration
					err := Export(ExportCheckout{Existing: prior, Info: info, Repository: checkoutRepository, PrepareSave: func() (func(settings.Declaration) error, error) {
						return func(d settings.Declaration) error { got = d; return nil }, nil
					}}, reader)
					return got, err
				}
				reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
				if field == "EnvironmentVariables" {
					reader.EnvironmentNames = func(repository.Repository) ([]string, error) { return []string{"prod"}, nil }
				}
				baseline, err := run(reader, nil)
				if err != nil {
					t.Fatal(err)
				}
				if field == "Topics" {
					baseline.Topics = prior.Topics
				}
				if field == "Labels" {
					baseline.Labels = prior.Labels
				}
				cause := errors.New("sensitive upstream response")
				called := 0
				foundCause := false
				rv := reflect.ValueOf(&reader).Elem().FieldByName(field)
				typ := rv.Type()
				rv.Set(reflect.MakeFunc(typ, func([]reflect.Value) []reflect.Value {
					called++
					result := make([]reflect.Value, typ.NumOut())
					for i := range result {
						result[i] = reflect.Zero(typ.Out(i))
					}
					if !zero {
						result[len(result)-1] = reflect.ValueOf(cause)
					}
					return result
				}))
				got, err := run(reader, func(_ string, e error) error { foundCause = foundCause || errors.Is(e, cause); return nil })
				if err != nil || called == 0 || (!zero && !foundCause) || !reflect.DeepEqual(got, baseline) {
					t.Fatalf("%s changed prior intent or stopped: called=%d cause=%v err=%v", field, called, foundCause, err)
				}
			})
		}
	}
}

func TestReadableExportVariableInventoriesAndPerEnvironmentRetention(t *testing.T) {
	old := exportPrior(t)
	for _, mode := range []string{"complete", "failed", "unsupported-empty"} {
		t.Run(mode, func(t *testing.T) {
			r := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			inventory := func() (settings.VariableInventory, error) {
				switch mode {
				case "failed":
					return settings.VariableInventory{}, errors.New("later page failed")
				case "unacquired":
					return settings.VariableInventory{}, nil
				case "unsupported-empty":
					return settings.NewVariableInventory([]settings.RepositoryVariable{{Name: "EMPTY", Value: ""}})
				}
				return settings.NewVariableInventory([]settings.RepositoryVariable{{Name: "B", Value: "two"}, {Name: "A", Value: "one"}})
			}
			r.Variables = func(repository.Repository) (settings.VariableInventory, error) { return inventory() }
			r.AgentVariables = func(repository.Repository) (settings.VariableInventory, error) {
				return settings.NewVariableInventory(nil)
			}
			r.EnvironmentNames = func(repository.Repository) ([]string, error) { return []string{"prod", "staging"}, nil }
			r.EnvironmentVariables = func(_ repository.Repository, name string) (settings.VariableInventory, error) {
				if name == "prod" {
					return settings.VariableInventory{}, errors.New("denied")
				}
				return settings.NewEnvironmentVariableInventory(name, 4, []settings.RepositoryVariable{{Name: "B", Value: "other"}})
			}
			var got settings.Declaration
			err := Export(ExportCheckout{Existing: old, Repository: checkoutRepository, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error { got = d; return nil }, nil
			}}, r)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "complete" {
				if got.Variables.Len() != 2 || got.Variables.Values()[0].Name != "A" {
					t.Fatal("inventory not exported")
				}
			} else if !reflect.DeepEqual(got.Variables, old.Variables) {
				t.Fatal("unknown/unsupported inventory erased prior values")
			}
			if got.AgentsVariables == nil || got.AgentsVariables.Len() != 0 {
				t.Fatal("complete empty inventory lost")
			}
			if len(got.EnvironmentVariables) != 2 || !reflect.DeepEqual(got.EnvironmentVariables[0], old.EnvironmentVariables[0]) || got.EnvironmentVariables[1].Variables.Values()[0].Value != "other" {
				t.Fatal("one Environment failure stopped another or erased intent")
			}
		})
	}
}

func TestReadableExportInfoFailureAndProjectionFailure(t *testing.T) {
	r := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
	r.Rulesets = func(repository.Repository) (settings.RulesetObservation, error) {
		return settings.NewRulesetObservation(nil, 0)
	}
	prior := exportPrior(t)
	var saved settings.Declaration
	cause := errors.New("diagnostic unavailable")
	err := Export(ExportCheckout{Existing: prior, Repository: checkoutRepository, Info: func(string, error) error { return cause }, PrepareSave: func() (func(settings.Declaration) error, error) {
		return func(d settings.Declaration) error { saved = d; return nil }, nil
	}}, r)
	if !errors.Is(err, cause) || !saved.Empty() {
		t.Fatal("diagnostic failure saved a file", err)
	}
	seen := false
	err = Export(ExportCheckout{Existing: prior, Repository: checkoutRepository, Check: func(d settings.Declaration) error {
		if d.Rulesets != prior.Rulesets {
			return errors.New("unsupported projection")
		}
		return nil
	}, Info: func(path string, e error) error {
		if path == "repository.settings.rulesets" {
			seen = true
		}
		return nil
	}, PrepareSave: func() (func(settings.Declaration) error, error) {
		return func(d settings.Declaration) error { saved = d; return nil }, nil
	}}, r)
	if err != nil || !seen || saved.Rulesets != prior.Rulesets {
		t.Fatal("failed family projection escaped", err, seen)
	}
}
