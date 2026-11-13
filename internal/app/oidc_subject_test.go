package app

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func appOIDC(defaults bool) settings.OIDCSubjectTemplate {
	v := settings.OIDCSubjectTemplate{UseDefaultTemplate: &defaults}
	if !defaults {
		keys := []string{"repo", "context"}
		v.SubjectClaimTemplate = &keys
	}
	return v
}
func TestOIDCPlanAndApplyProgress(t *testing.T) {
	d := appOIDC(false)
	before, _ := settings.NewOIDCSubjectObservation(appOIDC(true))
	plan, err := Plan(func() (settings.Declaration, error) { return settings.Declaration{OIDCSubject: &d}, nil }, checkoutRepository, Reader{OIDCSubject: func(repository.Repository) (settings.OIDCSubjectObservation, error) { return before, nil }})
	if err != nil || plan.Comparison.Managed != 1 || len(plan.Comparison.OIDCSubject) != 1 {
		t.Fatal(plan, err)
	}
	for _, earlier := range []bool{false, true} {
		for _, mode := range []string{"unchanged", "changed", "prewrite", "uncertain"} {
			desired := settings.Declaration{OIDCSubject: &d, Repository: settings.RepositoryDeclaration{Description: appString("description")}}
			result, err := Apply(func() (settings.Declaration, error) { return desired, nil }, checkoutRepository, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) { return earlier, nil }, OIDCSubject: func(_ repository.Repository, v settings.OIDCSubjectTemplate) (bool, error) {
				if !settings.OIDCSubjectMatches(v, d) {
					t.Fatal(v)
				}
				switch mode {
				case "prewrite":
					return false, &workflowReadFailure{errors.New("read failed")}
				case "uncertain":
					return false, errors.New("effects uncertain")
				}
				return mode == "changed", nil
			}})
			if mode == "unchanged" || mode == "changed" {
				if err != nil || !result.Completed() {
					t.Fatal(result, err)
				}
				continue
			}
			want := ApplyFailed
			if earlier {
				want = ApplyPartialFailure
			}
			status := OperationFailedUnknown
			if mode == "prewrite" {
				status = OperationFailedRead
			}
			if err == nil || result.Outcome != want || result.Operations[len(result.Operations)-1].Status != status {
				t.Fatal(earlier, mode, result, err)
			}
		}
	}
}
func TestOIDCExportIndependentRetention(t *testing.T) {
	for _, unavailable := range []bool{false, true} {
		prior := appOIDC(false)
		var saved settings.Declaration
		err := Export(ExportCheckout{Repository: checkoutRepository, Existing: settings.Declaration{OIDCSubject: &prior}, PrepareSave: func() (func(settings.Declaration) error, error) {
			return func(d settings.Declaration) error { saved = d; return nil }, nil
		}}, Reader{OIDCSubject: func(repository.Repository) (settings.OIDCSubjectObservation, error) {
			if unavailable {
				return settings.OIDCSubjectObservation{}, errors.New("unavailable")
			}
			return settings.NewOIDCSubjectObservation(appOIDC(true))
		}, Topics: func(repository.Repository) ([]string, error) { return []string{"acquired"}, nil }, WorkflowPermissions: func(repository.Repository) (settings.WorkflowPermissionsState, error) {
			return settings.WorkflowPermissionsState{DefaultWorkflowPermissions: "read", CanApprovePullRequestReviews: false}, nil
		}})
		if err != nil || saved.OIDCSubject == nil || *saved.OIDCSubject.UseDefaultTemplate == unavailable || saved.Topics.Names == nil || saved.Actions.DefaultWorkflowPermissions == nil || *saved.Actions.DefaultWorkflowPermissions != "read" || saved.Actions.CanApprovePullRequestReviews == nil || *saved.Actions.CanApprovePullRequestReviews {
			t.Fatal(saved, err)
		}
	}
}

func TestOIDCValidationAndArchiveOrdering(t *testing.T) {
	invalid := settings.OIDCSubjectTemplate{}
	called := false
	_, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{OIDCSubject: &invalid}, nil }, func() (repository.Repository, error) { called = true; return repository.Repository{}, nil }, Writer{})
	if err == nil || called {
		t.Fatal("invalid template reached target", err)
	}
	for _, scenario := range []string{"unarchive first", "archive last", "already archived", "failure stops later work"} {
		t.Run(scenario, func(t *testing.T) {
			desired := appOIDC(false)
			archive := scenario != "unarchive first"
			initial := scenario == "unarchive first" || scenario == "already archived"
			sequence := []string{}
			result, err := Apply(func() (settings.Declaration, error) {
				return settings.Declaration{OIDCSubject: &desired, Repository: settings.RepositoryDeclaration{Archived: &archive}}, nil
			}, checkoutRepository, Writer{
				ReadArchive: func(repo repository.Repository) (settings.ArchiveState, error) {
					sequence = append(sequence, "read")
					return settings.ArchiveState{ID: 1, FullName: repo.Owner + "/" + repo.Name, Archived: initial}, nil
				},
				Archive: func(_ repository.Repository, _ settings.ArchiveState, value bool) error {
					sequence = append(sequence, fmt.Sprint("archive:", value))
					return nil
				},
				OIDCSubject: func(repository.Repository, settings.OIDCSubjectTemplate) (bool, error) {
					sequence = append(sequence, "oidc")
					if scenario == "failure stops later work" {
						return false, errors.New("uncertain")
					}
					return true, nil
				},
			})
			want := []string{"read", "oidc", "read", "archive:true"}
			switch scenario {
			case "unarchive first":
				want = []string{"read", "archive:false", "oidc"}
			case "already archived":
				want = []string{"read"}
			case "failure stops later work":
				want = []string{"read", "oidc"}
			}
			fail := scenario == "already archived" || scenario == "failure stops later work"
			if (err != nil) != fail || !reflect.DeepEqual(sequence, want) || result.Completed() == fail {
				t.Fatal(sequence, result, err)
			}
		})
	}
}
