package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"testing"
)

func pruneRunnerState(id int64, custom string) settings.RunnerPruneState {
	labels := []settings.RunnerPruneLabel{{Name: "self-hosted", Kind: "read-only"}}
	if custom != "" {
		labels = append(labels, settings.RunnerPruneLabel{Name: custom, Kind: "custom"})
	}
	return settings.RunnerPruneState{ID: id, Name: "runner", OS: "linux", Status: "online", Labels: labels, ListLabels: labels, DetailLabels: labels, EndpointLabels: labels}
}
func pruneRunnerInventory() settings.RunnerPruneObservation {
	return settings.RunnerPruneObservation{Target: settings.RunnerPruneTarget{ID: 1, OwnerID: 2}, Entries: []settings.RunnerPruneState{pruneRunnerState(1, "old"), pruneRunnerState(2, "candidate"), pruneRunnerState(3, "another")}}
}
func TestRunnerPrunePlanPresentAndOwnership(t *testing.T) {
	p, _ := settings.ParsePruneScopes("runners")
	empty := []settings.RunnerLabels{}
	keep := []settings.RunnerLabels{appRunner(1, "new")}
	calls := 0
	read := Reader{RunnerPrune: func(repository.Repository) (settings.RunnerPruneObservation, error) {
		calls++
		return pruneRunnerInventory(), nil
	}}
	for _, tc := range []struct {
		desired                   *[]settings.RunnerLabels
		managed, updates, deletes int
	}{{nil, 0, 0, 0}, {&empty, 1, 0, 3}, {&keep, 1, 1, 2}} {
		result, e := Plan(func() (settings.Declaration, error) { return settings.Declaration{Runners: tc.desired}, nil }, checkoutRepository, read, p)
		if e != nil || result.Comparison.Managed != tc.managed || len(result.Comparison.Runners) != tc.updates || len(result.Comparison.RunnerDeletions) != tc.deletes {
			t.Fatal(result, e)
		}
	}
	if calls != 2 {
		t.Fatal(calls)
	}
}
func TestRunnerPruneGlobalPreflightBeforeOtherWrites(t *testing.T) {
	p, _ := settings.ParsePruneScopes("runners")
	empty := []settings.RunnerLabels{}
	description := "change"
	active := false
	writes := 0
	for _, mode := range []string{"busy final candidate", "incomplete", "archived"} {
		t.Run(mode, func(t *testing.T) {
			_, err := Apply(func() (settings.Declaration, error) {
				return settings.Declaration{Runners: &empty, Repository: settings.RepositoryDeclaration{Description: &description, Archived: &active}}, nil
			}, checkoutRepository, Writer{ReadRunnerPrune: func(repository.Repository) (settings.RunnerPruneObservation, error) {
				v := pruneRunnerInventory()
				switch mode {
				case "busy final candidate":
					v.Entries[2].Busy = true
				case "incomplete":
					return v, errors.New("incomplete")
				case "archived":
					v.Target.Archived = true
					v.Entries[2].Status = "unknown"
				}
				return v, nil
			}, DeleteRunnerPrune: func(repository.Repository, settings.RunnerPruneObservation, []int64, int64) (bool, error) {
				writes++
				return true, nil
			}, RunnerLabels: func(repository.Repository, settings.RunnerLabels) (bool, error) { writes++; return true, nil }, ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
				t.Fatal("read archive before all candidates")
				return settings.ArchiveState{}, nil
			}, Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) { writes++; return true, nil }}, p)
			if err == nil || writes != 0 {
				t.Fatal(err, writes)
			}
		})
	}
}
func TestRunnerPruneOwnOverlayFixedIDsAndProgress(t *testing.T) {
	for _, mode := range []string{"success", "foreign label drift", "own label unconfirmed", "second delete uncertain"} {
		t.Run(mode, func(t *testing.T) {
			desired := []settings.RunnerLabels{appRunner(1, "new")}
			initial := pruneRunnerInventory()
			current := initial
			events := []string{}
			result := ApplyResult{Outcome: ApplyFailed}
			writer := Writer{ReadRunnerPrune: func(repository.Repository) (settings.RunnerPruneObservation, error) {
				events = append(events, "read")
				return current, nil
			}, RunnerLabels: func(repository.Repository, settings.RunnerLabels) (bool, error) {
				events = append(events, "labels")
				current.Entries = append([]settings.RunnerPruneState{}, current.Entries...)
				if mode != "own label unconfirmed" {
					current.Entries[0] = pruneRunnerState(1, "new")
				}
				if mode == "foreign label drift" {
					current.Entries[1] = pruneRunnerState(2, "foreign")
				}
				return true, nil
			}, DeleteRunnerPrune: func(_ repository.Repository, expected settings.RunnerPruneObservation, ids []int64, id int64) (bool, error) {
				events = append(events, "delete")
				if id == 2 && !reflect.DeepEqual(ids, []int64{2, 3}) {
					t.Fatal(ids)
				}
				if id == 3 && !reflect.DeepEqual(ids, []int64{3}) {
					t.Fatal(ids)
				}
				if settings.VerifyRunnerPrune(expected, current, ids, nil) != nil {
					t.Fatal("lost verified label overlay")
				}
				if id == 3 && mode == "second delete uncertain" {
					return true, errors.New("effects unknown")
				}
				current, _ = current.Without(id)
				return true, nil
			}}
			wrote, err := applyRunnerPrune(&result, repository.Repository{}, desired, initial, false, writer)
			if mode == "success" {
				if err != nil || !wrote || !reflect.DeepEqual(events, []string{"read", "labels", "read", "delete", "delete"}) {
					t.Fatal(events, err)
				}
				return
			}
			if err == nil || !wrote || result.Outcome != ApplyPartialFailure {
				t.Fatal(result, err)
			}
			if mode != "second delete uncertain" && !reflect.DeepEqual(events, []string{"read", "labels", "read"}) {
				t.Fatal(events)
			}
			if mode == "second delete uncertain" && result.Operations[len(result.Operations)-1].Status != OperationFailedUnknown {
				t.Fatal(result)
			}
		})
	}
}

func TestRunnerPruneAfterConfirmedUnarchive(t *testing.T) {
	initial := pruneRunnerInventory()
	initial.Target.Archived = true
	active := initial.AfterUnarchive()
	desired := []settings.RunnerLabels{appRunner(1, "old"), appRunner(2, "candidate"), appRunner(3, "another")}
	calls := 0
	for _, confirmed := range []bool{false, true} {
		result := ApplyResult{Outcome: ApplyFailed}
		_, err := applyRunnerPrune(&result, repository.Repository{}, desired, initial, confirmed, Writer{ReadRunnerPrune: func(repository.Repository) (settings.RunnerPruneObservation, error) { return active, nil }, RunnerLabels: func(repository.Repository, settings.RunnerLabels) (bool, error) { calls++; return false, nil }})
		if confirmed && err != nil {
			t.Fatal(err)
		}
		if !confirmed && (err == nil || calls != 0) {
			t.Fatal("unconfirmed archive overlay", err, calls)
		}
	}
}
