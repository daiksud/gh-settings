package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func exemptionTarget() (repository.Repository, error) {
	return repository.Repository{Owner: "owner", Name: "current"}, nil
}
func TestPullRequestExemptionsPlanAndEmptyRemoval(t *testing.T) {
	for _, names := range [][]string{{"alice"}, {}} {
		desired, _ := settings.NewPullRequestExemptionsDeclaration(names)
		load := func() (settings.Declaration, error) {
			return settings.Declaration{PullRequestExemptions: &desired}, nil
		}
		result, err := Plan(load, exemptionTarget, Reader{PullRequestExemptions: func(repository.Repository) (settings.PullRequestExemptionsObservation, error) {
			return settings.NewPullRequestExemptionsObservation([]settings.PullRequestExemptionUser{{ID: 2, Login: "bob", Type: "User"}})
		}})
		if err != nil || result.Comparison.Managed != 1 || len(result.Comparison.PullRequestExemptions) != 1 || !result.Comparison.HasChanges() {
			t.Fatal(result, err)
		}
		before, after := result.Comparison.PullRequestExemptions[0].Values()
		if !reflect.DeepEqual(before, []string{"bob"}) || !reflect.DeepEqual(after, names) {
			t.Fatal(before, after)
		}
	}
	_, err := Plan(func() (settings.Declaration, error) { return settings.Declaration{}, nil }, func() (repository.Repository, error) {
		t.Fatal("resolved omitted list")
		return repository.Repository{}, nil
	}, Reader{})
	if err != nil {
		t.Fatal(err)
	}
}
func TestPullRequestExemptionsApplyProgress(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wrote   bool
		failure error
		status  ApplyOperationStatus
		outcome ApplyOutcome
	}{
		{"changed", true, nil, OperationCompleted, ApplyCompleted},
		{"matching", false, nil, OperationUnchanged, ApplyMatchingLabels},
		{"prewrite", false, &mcpReadFailure{errors.New("denied")}, OperationFailedRead, ApplyFailed},
		{"unknown", false, errors.New("uncertain"), OperationFailedUnknown, ApplyFailed},
		{"partial-removal", true, errors.New("addition failed after removal"), OperationFailedUnknown, ApplyPartialFailure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			desired, _ := settings.NewPullRequestExemptionsDeclaration([]string{})
			result, err := Apply(func() (settings.Declaration, error) {
				return settings.Declaration{PullRequestExemptions: &desired}, nil
			}, exemptionTarget, Writer{PullRequestExemptions: func(_ repository.Repository, d settings.PullRequestExemptionsDeclaration) (bool, error) {
				if len(d.Logins()) != 0 {
					t.Fatal(d)
				}
				return tc.wrote, tc.failure
			}})
			if (err != nil) != (tc.failure != nil) || len(result.Operations) != 1 || result.Operations[0].Kind != PullRequestExemptionsUpdate || result.Operations[0].Status != tc.status {
				t.Fatal(result, err)
			}
			// Existing matching-resource success uses ApplyMatchingLabels for no-write outcomes.
			if result.Outcome != tc.outcome {
				t.Fatal(result.Outcome, tc.outcome)
			}
			if err != nil && strings.Contains(err.Error(), "earlier successful writes remain applied") != (tc.outcome == ApplyPartialFailure) {
				t.Fatal("diagnostic does not match confirmed progress", result, err)
			}
		})
	}
}

func TestPullRequestExemptionsFailureDistinguishesRemoteAndLocalProgress(t *testing.T) {
	desired, _ := settings.NewPullRequestExemptionsDeclaration([]string{})
	for _, prior := range []struct {
		name       string
		operations []ApplyOperation
		partial    bool
	}{
		{"none", nil, false},
		{"local identity binding", []ApplyOperation{{Kind: CollaboratorIdentityBinding, Status: OperationCompleted}}, false},
		{"read only", []ApplyOperation{{Kind: LabelsRead, Status: OperationCompleted}}, false},
		{"confirmed remote update", []ApplyOperation{{Kind: RepositoryUpdate, Status: OperationCompleted}}, true},
	} {
		t.Run(prior.name, func(t *testing.T) {
			result := ApplyResult{Operations: prior.operations}
			cause := &mcpReadFailure{errors.New("denied before write")}
			_, err := applyPullRequestExemptions(&result, repository.Repository{}, desired, Writer{
				PullRequestExemptions: func(repository.Repository, settings.PullRequestExemptionsDeclaration) (bool, error) {
					return false, cause
				},
			})
			if !errors.Is(err, cause) || (result.Outcome == ApplyPartialFailure) != prior.partial {
				t.Fatal(result, err)
			}
		})
	}
}
func TestPullRequestExemptionsExportIsolation(t *testing.T) {
	for _, mode := range []string{"selected", "empty"} {
		t.Run(mode, func(t *testing.T) {
			reader := exportFixtureReader(t, settings.RepositoryState{}, func(string, repository.Repository) error { return nil })
			reads := 0
			reader.PullRequestExemptions = func(repository.Repository) (settings.PullRequestExemptionsObservation, error) {
				reads++
				if mode == "denied" {
					return settings.PullRequestExemptionsObservation{}, errors.New("denied")
				}
				if mode == "unacquired" {
					return settings.PullRequestExemptionsObservation{}, nil
				}
				users := []settings.PullRequestExemptionUser{}
				if mode != "empty" {
					users = append(users, settings.PullRequestExemptionUser{ID: 1, Login: "alice", Type: "User"})
				}
				return settings.NewPullRequestExemptionsObservation(users)
			}
			saved := false
			err := Export(ExportCheckout{Repository: exemptionTarget, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					saved = true
					if !d.PullRequestLimits.Empty() {
						t.Fatal("exemption export fabricated cap intent")
					}
					if mode == "ordinary" {
						if d.PullRequestExemptions != nil {
							t.Fatal(d)
						}
					} else if d.PullRequestExemptions == nil || mode == "empty" && len(d.PullRequestExemptions.Logins()) != 0 {
						t.Fatal(d)
					}
					return nil
				}, nil
			}}, reader)
			fail := mode == "denied" || mode == "unacquired"
			if (err != nil) != fail || saved == fail || mode == "ordinary" && reads != 0 || mode != "ordinary" && reads != 1 {
				t.Fatal(err, saved, reads)
			}
		})
	}
}

func TestPullRequestExemptionsArchiveAndPriorWrite(t *testing.T) {
	desired, _ := settings.NewPullRequestExemptionsDeclaration([]string{})
	archived := true
	_, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{PullRequestExemptions: &desired, Repository: settings.RepositoryDeclaration{Archived: &archived}}, nil
	}, exemptionTarget, Writer{ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
		return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
	}, PullRequestExemptions: func(repository.Repository, settings.PullRequestExemptionsDeclaration) (bool, error) {
		t.Fatal("exemptions written on archived target")
		return false, nil
	}})
	if err == nil {
		t.Fatal("archived target accepted")
	}
	description := "changed"
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{PullRequestExemptions: &desired, Repository: settings.RepositoryDeclaration{Description: &description}}, nil
	}, exemptionTarget, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) { return true, nil }, PullRequestExemptions: func(repository.Repository, settings.PullRequestExemptionsDeclaration) (bool, error) {
		return false, errors.New("uncertain")
	}})
	if err == nil || result.Outcome != ApplyPartialFailure || result.Operations[0].Status != OperationCompleted || strings.Count(err.Error(), "earlier successful writes remain applied") != 1 {
		t.Fatal(result, err)
	}
}
