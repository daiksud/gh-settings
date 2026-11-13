package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

type customAppCompletedFailure struct{ error }

func (customAppCompletedFailure) EnvironmentWritesCompleted() {}

func TestEnvironmentAppProgressDoesNotInventCreationWriteback(t *testing.T) {
	ids := []int64{2}
	d, _ := settings.NewEnvironmentDeclarations([]settings.EnvironmentDeclaration{{Name: "prod", CustomRules: &ids}})
	zero, off, reviewers := 0, false, []settings.EnvironmentReviewer{}
	observed, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 7, CustomRules: []settings.EnvironmentCustomRule{}, Policy: settings.EnvironmentDeclaration{Name: "prod", WaitTimer: &zero, PreventSelfReview: &off, CanAdminsBypass: &off, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"matching", "success", "unknown", "confirmed"} {
		t.Run(mode, func(t *testing.T) {
			result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Environments: &d}, nil }, func() (repository.Repository, error) { return repository.Repository{}, nil }, Writer{ReadEnvironments: func(repository.Repository, []string) (settings.EnvironmentObservation, error) { return observed, nil }, Environment: func(_ repository.Repository, p settings.EnvironmentDeclaration) (bool, error) {
				if p.CustomRules == nil || p.WaitTimer != nil {
					t.Fatal("lost partial App intent", p)
				}
				switch mode {
				case "matching":
					return false, nil
				case "success":
					return true, nil
				case "confirmed":
					return false, customAppCompletedFailure{errors.New("binding rule ID 102; later removal failed")}
				default:
					return false, errors.New("binding rule ID 102; independent readback failed")
				}
			}, CreatedID: func(string, string, int64) error { t.Fatal("binding entered YAML writeback"); return nil }})
			if len(result.CreatedIDs) != 0 {
				t.Fatal(result)
			}
			if mode == "matching" && (err != nil || result.Operations[1].Status != OperationUnchanged) {
				t.Fatal(result, err)
			}
			if mode == "success" && (err != nil || !result.Completed()) {
				t.Fatal(result, err)
			}
			if mode == "unknown" && (err == nil || result.Outcome != ApplyFailed) {
				t.Fatal(result, err)
			}
			if mode == "confirmed" && (err == nil || result.Outcome != ApplyPartialFailure) {
				t.Fatal(result, err)
			}
		})
	}
}
