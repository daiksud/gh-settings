package app

import (
	"errors"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"testing"
)

func appRunner(id int64, labels ...string) settings.RunnerLabels {
	if labels == nil {
		labels = []string{}
	}
	return settings.RunnerLabels{ID: id, CustomLabels: &labels}
}
func runnerObservation(t *testing.T, values ...settings.RunnerLabels) settings.RunnerObservation {
	t.Helper()
	states := []settings.RunnerState{}
	for _, v := range values {
		states = append(states, settings.RunnerState{Value: v, ReadOnlyLabels: []string{"self-hosted"}})
	}
	o, e := settings.NewRunnerObservation(states)
	if e != nil {
		t.Fatal(e)
	}
	return o
}
func TestRunnerPlanAndEmptyOwnership(t *testing.T) {
	d := []settings.RunnerLabels{appRunner(23, "gpu"), appRunner(24)}
	calls := 0
	p, e := Plan(func() (settings.Declaration, error) { return settings.Declaration{Runners: &d}, nil }, checkoutRepository, Reader{Runners: func(_ repository.Repository, want []settings.RunnerLabels) (settings.RunnerObservation, error) {
		calls++
		if !reflect.DeepEqual(want, d) {
			t.Fatal(want)
		}
		return runnerObservation(t, appRunner(23, "old"), appRunner(24, "old")), nil
	}})
	if e != nil || p.Comparison.Managed != 1 || len(p.Comparison.Runners) != 2 || calls != 1 {
		t.Fatal(p, e, calls)
	}
	empty := []settings.RunnerLabels{}
	for _, v := range []*[]settings.RunnerLabels{nil, &empty} {
		p, e = Plan(func() (settings.Declaration, error) { return settings.Declaration{Runners: v}, nil }, checkoutRepository, Reader{})
		if e != nil || p.Comparison.Managed != 0 {
			t.Fatal(p, e)
		}
		r, e := Apply(func() (settings.Declaration, error) { return settings.Declaration{Runners: v}, nil }, checkoutRepository, Writer{})
		if e != nil || len(r.Operations) != 0 {
			t.Fatal(r, e)
		}
	}
}
func TestRunnerApplyPreparationAndProgress(t *testing.T) {
	for _, mode := range []string{"unchanged", "changed", "prepare denied", "prepare missing", "collision", "second prewrite", "second uncertain", "first uncertain"} {
		t.Run(mode, func(t *testing.T) {
			d := []settings.RunnerLabels{appRunner(23, "gpu"), appRunner(24, "gpu"), appRunner(25, "gpu")}
			sequence := []int64{}
			result, e := Apply(func() (settings.Declaration, error) { return settings.Declaration{Runners: &d}, nil }, checkoutRepository, Writer{ReadRunners: func(repository.Repository, []settings.RunnerLabels) (settings.RunnerObservation, error) {
				sequence = append(sequence, 0)
				if mode == "prepare denied" {
					return settings.RunnerObservation{}, errors.New("unavailable")
				}
				if mode == "prepare missing" {
					return runnerObservation(t, appRunner(23)), nil
				}
				if mode == "collision" {
					o, _ := settings.NewRunnerObservation([]settings.RunnerState{{Value: appRunner(23)}, {Value: appRunner(24), ReadOnlyLabels: []string{"GPU"}}, {Value: appRunner(25)}})
					return o, nil
				}
				return runnerObservation(t, appRunner(23), appRunner(24), appRunner(25)), nil
			}, RunnerLabels: func(_ repository.Repository, v settings.RunnerLabels) (bool, error) {
				sequence = append(sequence, v.ID)
				if v.ID == 24 && mode == "second prewrite" {
					return false, &workflowReadFailure{errors.New("before write")}
				}
				if (v.ID == 24 && mode == "second uncertain") || (v.ID == 23 && mode == "first uncertain") {
					return false, errors.New("effects unknown")
				}
				return mode != "unchanged", nil
			}})
			fail := mode != "changed" && mode != "unchanged"
			if (e != nil) != fail {
				t.Fatal(result, e)
			}
			switch mode {
			case "prepare denied", "prepare missing", "collision":
				if !reflect.DeepEqual(sequence, []int64{0}) || result.Outcome != ApplyFailed {
					t.Fatal(sequence, result)
				}
			case "second prewrite", "second uncertain":
				want := OperationFailedUnknown
				if mode == "second prewrite" {
					want = OperationFailedRead
				}
				if !reflect.DeepEqual(sequence, []int64{0, 23, 24}) || result.Outcome != ApplyPartialFailure || result.Operations[2].Status != want {
					t.Fatal(sequence, result)
				}
			case "first uncertain":
				if result.Outcome != ApplyFailed || !reflect.DeepEqual(sequence, []int64{0, 23}) {
					t.Fatal(result, sequence)
				}
			default:
				if !result.Completed() || len(result.Operations) != 4 {
					t.Fatal(result)
				}
			}
		})
	}
}
func TestRunnerExportRetentionAndDiagnostics(t *testing.T) {
	for _, mode := range []string{"complete", "empty", "unavailable", "first unavailable", "invalid observation", "diagnostic failure"} {
		t.Run(mode, func(t *testing.T) {
			prior := []settings.RunnerLabels{appRunner(23, "prior")}
			existing := settings.Declaration{Runners: &prior}
			if mode == "first unavailable" {
				existing = settings.Declaration{}
			}
			saved := false
			var got settings.Declaration
			infos := 0
			e := Export(ExportCheckout{Repository: checkoutRepository, Existing: existing, Info: func(string, error) error {
				infos++
				if mode == "diagnostic failure" {
					return errors.New("stderr")
				}
				return nil
			}, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error { saved = true; got = d; return nil }, nil
			}}, Reader{AllRunners: func(repository.Repository) (settings.RunnerObservation, error) {
				switch mode {
				case "complete":
					return runnerObservation(t, appRunner(23, "acquired")), nil
				case "empty":
					return runnerObservation(t), nil
				case "invalid observation":
					return settings.RunnerObservation{}, nil
				}
				return settings.RunnerObservation{}, errors.New("unavailable")
			}, Topics: func(repository.Repository) ([]string, error) { return []string{"sibling"}, nil }})
			if mode == "diagnostic failure" {
				if e == nil || saved {
					t.Fatal(e, saved)
				}
				return
			}
			if e != nil || !saved || got.Topics.Names == nil {
				t.Fatal(got, e)
			}
			if mode == "first unavailable" {
				if got.Runners != nil {
					t.Fatal(got)
				}
				return
			}
			if got.Runners == nil {
				t.Fatal(got)
			}
			if mode == "empty" {
				if len(*got.Runners) != 0 {
					t.Fatal(got)
				}
				return
			}
			want := "prior"
			if mode == "complete" {
				want = "acquired"
			}
			if (*(*got.Runners)[0].CustomLabels)[0] != want {
				t.Fatal(got)
			}
		})
	}
}
func TestRunnerUnarchiveBeforeFamilyPreparation(t *testing.T) {
	d := []settings.RunnerLabels{appRunner(23, "gpu")}
	archive := false
	sequence := []string{}
	result, e := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Runners: &d, Repository: settings.RepositoryDeclaration{Archived: &archive}}, nil
	}, checkoutRepository, Writer{ReadArchive: func(r repository.Repository) (settings.ArchiveState, error) {
		return settings.ArchiveState{ID: 1, FullName: r.Owner + "/" + r.Name, Archived: true}, nil
	}, Archive: func(repository.Repository, settings.ArchiveState, bool) error {
		sequence = append(sequence, "unarchive")
		return nil
	}, ReadRunners: func(repository.Repository, []settings.RunnerLabels) (settings.RunnerObservation, error) {
		sequence = append(sequence, "prepare")
		return runnerObservation(t, appRunner(23)), nil
	}, RunnerLabels: func(repository.Repository, settings.RunnerLabels) (bool, error) {
		sequence = append(sequence, "write")
		return true, nil
	}})
	if e != nil || !result.Completed() || !reflect.DeepEqual(sequence, []string{"unarchive", "prepare", "write"}) {
		t.Fatal(result, sequence, e)
	}
}
