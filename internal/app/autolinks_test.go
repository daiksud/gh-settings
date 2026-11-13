package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func appAutolink(id int64, prefix string) settings.Autolink {
	v := settings.Autolink{ReferencePrefix: prefix, TargetURL: "https://test/<num>", ReferenceFormat: "numeric"}
	if id > 0 {
		v.ID = &id
	}
	return v
}
func appAutolinks() settings.AutolinkObservation {
	return settings.AutolinkObservation{Target: settings.AutolinkTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, Entries: []settings.AutolinkState{{Value: appAutolink(7, "OLD-")}}}
}
func TestAutolinkPruneGlobalGateBeforeUnrelatedWrite(t *testing.T) {
	values := []settings.Autolink{}
	title := "new"
	no := false
	scopes, _ := settings.ParsePruneScopes("autolinks")
	_, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Autolinks: &values, Repository: settings.RepositoryDeclaration{Archived: &no, Description: &title}}, nil
	}, func() (repository.Repository, error) { return repository.Repository{}, nil }, Writer{ReadAutolinks: func(repository.Repository) (settings.AutolinkObservation, error) {
		return settings.AutolinkObservation{}, errors.New("unavailable")
	}, ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
		t.Fatal("archive read after failed global gate")
		return settings.ArchiveState{}, nil
	}, Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
		t.Fatal("unrelated write after failed global gate")
		return false, nil
	}}, scopes)
	if err == nil {
		t.Fatal("unsafe global gate")
	}
}
func TestAutolinkApplyOwnInventoryRecordingAndFailures(t *testing.T) {
	for _, mode := range []string{"success", "no-op", "creation-unknown", "creation-prewrite", "writeback", "delete-prewrite", "delete-unknown", "global-drift", "own-unarchive"} {
		t.Run(mode, func(t *testing.T) {
			values := []settings.Autolink{appAutolink(0, "NEW-")}
			if mode == "no-op" {
				values = []settings.Autolink{appAutolink(7, "OLD-")}
			}
			initial := appAutolinks()
			current := initial.Clone()
			d := settings.Declaration{Autolinks: &values}
			if mode == "own-unarchive" {
				initial.Target.Archived = true
				current.Target.Archived = true
				no := false
				d.Repository.Archived = &no
			}
			calls := []string{}
			reads := 0
			writer := Writer{ReadAutolinks: func(repository.Repository) (settings.AutolinkObservation, error) {
				reads++
				calls = append(calls, "read")
				if mode == "global-drift" && reads > 1 {
					current.Entries[0].Value.TargetURL += "drift"
				}
				return current.Clone(), nil
			}, ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
				return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: true}, nil
			}, Archive: func(repository.Repository, settings.ArchiveState, bool) error {
				calls = append(calls, "unarchive")
				current.Target.Archived = false
				return nil
			}, CreateAutolink: func(_ repository.Repository, expected settings.AutolinkObservation, value settings.Autolink) (settings.AutolinkWriteResult, error) {
				calls = append(calls, "create")
				if err := settings.VerifyAutolinkInventory(expected, current); err != nil {
					t.Fatal(err)
				}
				if mode == "creation-prewrite" {
					return settings.AutolinkWriteResult{}, errors.New("read failed")
				}
				if mode == "creation-unknown" {
					return settings.AutolinkWriteResult{Attempted: true, RecoveryID: 8}, errors.New("unconfirmed")
				}
				state := settings.AutolinkState{Value: appAutolink(8, "NEW-")}
				var err error
				current, err = current.WithCreated(state)
				if err != nil {
					t.Fatal(err)
				}
				return settings.AutolinkWriteResult{Attempted: true, RecoveryID: 8, Verified: &state}, nil
			}, CreatedID: func(collection, prefix string, id int64) error {
				calls = append(calls, "record")
				if collection != "autolinks" || prefix != "NEW-" || id != 8 {
					t.Fatal(collection, prefix, id)
				}
				if mode == "writeback" {
					return errors.New("file changed")
				}
				return nil
			}, DeleteAutolink: func(_ repository.Repository, expected settings.AutolinkObservation, id int64) (bool, error) {
				calls = append(calls, "delete")
				if err := settings.VerifyAutolinkInventory(expected, current); err != nil {
					t.Fatal("own creation missing from fixed expected inventory", err)
				}
				if id != 7 {
					t.Fatal("candidate widened", id)
				}
				if mode == "delete-prewrite" {
					return false, errors.New("client failed")
				}
				if mode == "delete-unknown" {
					return true, errors.New("readback failed")
				}
				current = current.Without(id)
				return true, nil
			}}
			scopes, _ := settings.ParsePruneScopes("autolinks")
			result, err := Apply(func() (settings.Declaration, error) { return d, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, writer, scopes)
			good := mode == "success" || mode == "no-op" || mode == "own-unarchive"
			if (err == nil) != good {
				t.Fatal(result, err, calls)
			}
			if good && mode != "no-op" {
				want := []string{"read", "read", "create", "record", "delete"}
				if mode == "own-unarchive" {
					want = []string{"read", "unarchive", "read", "create", "record", "delete"}
				}
				if !reflect.DeepEqual(calls, want) {
					t.Fatal(calls)
				}
			}
			if mode == "no-op" && result.Outcome != ApplyMatchingLabels {
				t.Fatal(result)
			}
			if mode == "writeback" || mode == "creation-unknown" || mode == "creation-prewrite" || mode == "global-drift" {
				for _, call := range calls {
					if call == "delete" {
						t.Fatal("deleted before successful declared work", calls)
					}
				}
			}
			if mode == "creation-unknown" && (len(result.CreatedIDs) != 1 || result.CreatedIDs[0].ID != 8) {
				t.Fatal("recovery lost", result)
			}
			for _, op := range result.Operations {
				if op.Kind == AutolinkDelete && mode == "delete-prewrite" && op.Status != OperationFailedRead {
					t.Fatal(op)
				}
				if op.Kind == AutolinkDelete && mode == "delete-unknown" && op.Status != OperationFailedUnknown {
					t.Fatal(op)
				}
			}
			if (mode == "delete-prewrite" || mode == "delete-unknown" || mode == "writeback") && result.Outcome != ApplyPartialFailure {
				t.Fatal("earlier confirmed creation lost", result)
			}
		})
	}
}
func TestAutolinkExportIndependentEmptyAndUnavailable(t *testing.T) {
	for _, prior := range []bool{false, true} {
		for _, mode := range []string{"complete", "empty", "unavailable"} {
			t.Run(mode+map[bool]string{false: "-initial", true: "-prior"}[prior], func(t *testing.T) {
				existing := settings.Declaration{}
				if prior {
					values := []settings.Autolink{appAutolink(9, "PRIOR-")}
					existing.Autolinks = &values
				}
				var saved settings.Declaration
				infos := []string{}
				err := Export(ExportCheckout{Existing: existing, Repository: func() (repository.Repository, error) { return repository.Repository{}, nil }, PrepareSave: func() (func(settings.Declaration) error, error) {
					return func(d settings.Declaration) error { saved = d; return nil }, nil
				}, Info: func(path string, _ error) error { infos = append(infos, path); return nil }}, Reader{Autolinks: func(repository.Repository) (settings.AutolinkObservation, error) {
					o := appAutolinks()
					if mode == "empty" {
						o.Entries = []settings.AutolinkState{}
					}
					if mode == "unavailable" {
						return settings.AutolinkObservation{}, errors.New("denied")
					}
					return o, nil
				}})
				if err != nil {
					t.Fatal(err)
				}
				if mode == "unavailable" {
					if !reflect.DeepEqual(saved.Autolinks, existing.Autolinks) {
						t.Fatal(saved)
					}
					return
				}
				if saved.Autolinks == nil || (mode == "empty" && len(*saved.Autolinks) != 0) || (mode == "complete" && *(*saved.Autolinks)[0].ID != 7) {
					t.Fatal(saved)
				}
			})
		}
	}
}

func TestAutolinkSecondDeletionRetainsConfirmedProgress(t *testing.T) {
	current := appAutolinks()
	current.Entries = append(current.Entries, settings.AutolinkState{Value: appAutolink(9, "SECOND-")})
	desired := []settings.Autolink{}
	scope, _ := settings.ParsePruneScopes("autolinks")
	deletions := 0
	result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Autolinks: &desired}, nil }, func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}, Writer{ReadAutolinks: func(repository.Repository) (settings.AutolinkObservation, error) { return current.Clone(), nil }, DeleteAutolink: func(_ repository.Repository, expected settings.AutolinkObservation, id int64) (bool, error) {
		if err := settings.VerifyAutolinkInventory(expected, current); err != nil {
			t.Fatal("verified own deletion not accounted for", err)
		}
		deletions++
		if deletions == 2 {
			return true, errors.New("unverified deletion")
		}
		current = current.Without(id)
		return true, nil
	}}, scope)
	if err == nil || result.Outcome != ApplyPartialFailure || deletions != 2 || len(result.Operations) != 3 || result.Operations[1].Status != OperationCompleted || result.Operations[2].Status != OperationFailedUnknown {
		t.Fatal(result, err)
	}
}
