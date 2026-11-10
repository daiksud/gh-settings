package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestCopilotReviewEffortPlan(t *testing.T) {
	for _, tc := range []struct {
		name            string
		before, desired string
		mode            string
		changes         int
	}{
		{"explicit balanced", "default", "balanced", "", 1}, {"inherit", "lite", "default", "", 1}, {"matching default", "default", "default", "", 0}, {"denied", "default", "lite", "denied", 0}, {"unread", "default", "lite", "unread", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			result, err := Plan(func() (settings.Declaration, error) {
				return settings.Declaration{CopilotReviewEffort: &tc.desired}, nil
			}, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, Reader{CopilotReviewEffort: func(repo repository.Repository) (settings.CopilotReviewEffortObservation, error) {
				reads++
				if repo.Owner != "owner" || repo.Name != "current" {
					t.Fatal("wrong target")
				}
				if tc.mode == "denied" {
					return settings.CopilotReviewEffortObservation{}, errors.New("denied")
				}
				if tc.mode == "unread" {
					return settings.CopilotReviewEffortObservation{}, nil
				}
				return effortObservation(t, tc.before), nil
			}})
			if (err != nil) != (tc.mode != "") || reads != 1 || len(result.Comparison.CopilotReviewEffort) != tc.changes {
				t.Fatal(result, err, reads)
			}
			if tc.mode != "" {
				if result.Comparison.Managed != 0 || result.Comparison.HasChanges() {
					t.Fatal("partial plan exposed")
				}
				return
			}
			if result.Comparison.Managed != 1 || result.Comparison.HasChanges() != (tc.changes > 0) {
				t.Fatal(result)
			}
			if tc.changes > 0 {
				before, after := result.Comparison.CopilotReviewEffort[0].Values()
				if before != tc.before || after != tc.desired {
					t.Fatal("wrong effort comparison")
				}
			}
		})
	}
	_, err := Plan(func() (settings.Declaration, error) { return settings.Declaration{}, nil }, func() (repository.Repository, error) {
		t.Fatal("resolved omitted setting")
		return repository.Repository{}, nil
	}, Reader{})
	if err != nil {
		t.Fatal(err)
	}
}

type effortReadFailure struct{ error }

func (*effortReadFailure) ReadFailedBeforeWrite() {}
func TestCopilotReviewEffortApply(t *testing.T) {
	target := func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}
	for _, tc := range []struct {
		name    string
		desired string
		wrote   bool
		err     error
		outcome ApplyOutcome
		status  ApplyOperationStatus
	}{
		{"changed default", "default", true, nil, ApplyCompleted, OperationCompleted},
		{"changed lite", "lite", true, nil, ApplyCompleted, OperationCompleted},
		{"matching default", "default", false, nil, ApplyMatchingLabels, OperationUnchanged},
		{"preread failure", "lite", false, &effortReadFailure{errors.New("denied")}, ApplyFailed, OperationFailedRead},
		{"unknown write", "balanced", false, errors.New("write may have applied"), ApplyFailed, OperationFailedUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			result, err := Apply(func() (settings.Declaration, error) {
				return settings.Declaration{CopilotReviewEffort: &tc.desired}, nil
			}, target, Writer{CopilotReviewEffort: func(repo repository.Repository, desired string) (bool, error) {
				calls++
				if desired != tc.desired || repo.Name != "current" {
					t.Fatal("wrong intent/target")
				}
				return tc.wrote, tc.err
			}})
			if (err != nil) != (tc.err != nil) || calls != 1 || result.Outcome != tc.outcome || len(result.Operations) != 1 || result.Operations[0].Status != tc.status || result.Operations[0].Kind != CopilotReviewEffortUpdate {
				t.Fatal(result, err, calls)
			}
		})
	}
	enabled := true
	effort := "lite"
	description := "changed"
	var calls []string
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}, CopilotReviewEffort: &effort, Security: settings.SecurityDeclaration{PrivateVulnerabilityReporting: &enabled}}, nil
	}, target, Writer{
		Repository: func(repository.Repository, settings.RepositoryDeclaration) error {
			calls = append(calls, "repository")
			return nil
		},
		CopilotReviewEffort: func(repository.Repository, string) (bool, error) {
			calls = append(calls, "copilot")
			return false, errors.New("uncertain update")
		},
		ReadPrivateReporting: func(repository.Repository) (settings.PrivateReportingObservation, error) {
			t.Fatal("later resource read after failure")
			return settings.PrivateReportingObservation{}, nil
		},
	})
	if err == nil || result.Outcome != ApplyPartialFailure || len(calls) != 2 || calls[0] != "repository" || calls[1] != "copilot" || result.Operations[0].Status != OperationCompleted {
		t.Fatal(result, err, calls)
	}
	result, err = Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Description: &description}}, nil
	}, target, Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) error { return nil }, CopilotReviewEffort: func(repository.Repository, string) (bool, error) {
		t.Fatal("called omitted setting")
		return false, nil
	}})
	if err != nil || result.Outcome != ApplyCompleted || len(result.Operations) != 1 {
		t.Fatal(result, err)
	}
	archived := true
	_, err = Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &archived}, CopilotReviewEffort: &effort}, nil
	}, target, Writer{ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
		return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
	}, CopilotReviewEffort: func(repository.Repository, string) (bool, error) {
		t.Fatal("updated archived target")
		return false, nil
	}})
	if err == nil {
		t.Fatal("allowed incompatible archived-target update")
	}
}

func TestCopilotReviewEffortExport(t *testing.T) {
	for _, tc := range []struct {
		name    string
		include bool
		enabled string
		mode    string
	}{
		{"ordinary", false, "default", ""}, {"lite", true, "lite", ""}, {"default", true, "default", ""}, {"balanced", true, "balanced", ""}, {"denied", true, "default", "denied"}, {"unread", true, "default", "unread"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := exportFixtureReader(t, settings.RepositoryState{}, func(string, repository.Repository) error { return nil })
			reads, saves := 0, 0
			reader.CopilotReviewEffort = func(repo repository.Repository) (settings.CopilotReviewEffortObservation, error) {
				reads++
				if repo.Name != "current" {
					t.Fatal("wrong target")
				}
				if tc.mode == "denied" {
					return settings.CopilotReviewEffortObservation{}, errors.New("denied")
				}
				if tc.mode == "unread" {
					return settings.CopilotReviewEffortObservation{}, nil
				}
				return effortObservation(t, tc.enabled), nil
			}
			err := Export(ExportCheckout{IncludeCopilotReviewEffort: tc.include, Repository: func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error {
					saves++
					if (d.CopilotReviewEffort != nil) != tc.include || tc.include && *d.CopilotReviewEffort != tc.enabled {
						t.Fatal(d)
					}
					return nil
				}, nil
			}}, reader)
			if (err != nil) != (tc.mode != "") || reads != map[bool]int{false: 0, true: 1}[tc.include] || (saves == 1) != (tc.mode == "") {
				t.Fatal(err, reads, saves)
			}
		})
	}
}

func effortObservation(t *testing.T, value string) settings.CopilotReviewEffortObservation {
	t.Helper()
	var configured *string
	effective := "balanced"
	if value != "default" {
		configured = &value
		effective = value
	}
	o, err := settings.NewCopilotReviewEffortObservation(configured, effective, []string{"lite", "balanced", "max"})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestCopilotControlsSelectivePlan(t *testing.T) {
	for _, includeEffort := range []bool{false, true} {
		for _, includeInstructions := range []bool{false, true} {
			effort, instructions := "balanced", false
			desired := settings.Declaration{}
			if includeEffort {
				desired.CopilotReviewEffort = &effort
			}
			if includeInstructions {
				desired.CopilotInstructions = &instructions
			}
			effortReads, instructionsReads := 0, 0
			result, err := Plan(func() (settings.Declaration, error) { return desired, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, Reader{
				CopilotReviewEffort: func(repository.Repository) (settings.CopilotReviewEffortObservation, error) {
					effortReads++
					return effortObservation(t, "default"), nil
				},
				CopilotInstructions: func(repository.Repository) (settings.CopilotInstructionsObservation, error) {
					instructionsReads++
					return settings.NewCopilotInstructionsObservation(true), nil
				},
			})
			expected := 0
			if includeEffort {
				expected++
			}
			if includeInstructions {
				expected++
			}
			if err != nil || result.Comparison.Managed != expected || (effortReads == 1) != includeEffort || (instructionsReads == 1) != includeInstructions {
				t.Fatal(result, err, effortReads, instructionsReads)
			}
		}
	}
}
