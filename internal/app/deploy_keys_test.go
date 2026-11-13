package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

const appDeployOne = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB"
const appDeployTwo = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC"

func appDeployState(t *testing.T, id int64, material string) settings.DeployKeyState {
	t.Helper()
	key, e := settings.ParseDeployPublicKey(material)
	if e != nil {
		t.Fatal(e)
	}
	return settings.DeployKeyState{ID: id, Title: "same", Key: key, ReadOnly: true, CreatedAt: "2026-11-14T00:00:00Z", APIURL: fmt.Sprintf("https://api.github.com/repos/owner/current/keys/%d", id)}
}
func appDeployObservation(t *testing.T, archived bool, rows ...settings.DeployKeyState) settings.DeployKeyObservation {
	t.Helper()
	o, e := settings.NewDeployKeyObservation(settings.DeployKeyTarget{ID: 1, OwnerID: 2, FullName: "owner/current", Archived: archived}, rows)
	if e != nil {
		t.Fatal(e)
	}
	return o
}

type deployPreError struct{}

func (deployPreError) Error() string          { return "unavailable before mutation" }
func (deployPreError) ReadFailedBeforeWrite() {}
func TestDeployKeyGlobalPreflight(t *testing.T) {
	empty, _ := settings.NewDeployKeyDeclarations(nil)
	prune, _ := settings.ParsePruneScopes("deploy-keys")
	title := "changed"
	active := false
	writes := 0
	_, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{DeployKeys: &empty, Repository: settings.RepositoryDeclaration{Description: &title, Archived: &active}}, nil
	}, func() (repository.Repository, error) { return repository.Repository{}, nil }, Writer{
		ReadDeployKeys: func(repository.Repository) (settings.DeployKeyObservation, error) {
			return settings.DeployKeyObservation{}, errors.New("incomplete final candidate")
		},
		Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) { writes++; return true, nil },
		ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
			t.Fatal("unarchive before complete prune acquisition")
			return settings.ArchiveState{}, nil
		},
	}, prune)
	if err == nil || writes != 0 {
		t.Fatal(err, writes)
	}
}
func TestDeployKeyApplicationProgressAndFixedCandidates(t *testing.T) {
	for _, mode := range []string{"success", "create prewrite", "create uncertain", "recovery only", "binding failure", "delete prewrite", "delete uncertain", "wrong create inventory", "fresh drift"} {
		t.Run(mode, func(t *testing.T) {
			old := appDeployState(t, 1, appDeployOne)
			newKey := appDeployState(t, 2, appDeployTwo)
			initial := appDeployObservation(t, false, old)
			current := initial
			declaration := newKey.Declaration()
			declaration.ID = nil
			desired, _ := settings.NewDeployKeyDeclarations([]settings.DeployKeyDeclaration{declaration})
			events := []string{}
			result := ApplyResult{Outcome: ApplyFailed}
			writer := Writer{
				ReadDeployKeys: func(repository.Repository) (settings.DeployKeyObservation, error) {
					if mode == "fresh drift" {
						return appDeployObservation(t, false), nil
					}
					return current, nil
				},
				CreateDeployKey: func(_ repository.Repository, expected settings.DeployKeyObservation, d settings.DeployKeyDeclaration) (settings.DeployKeyWriteResult, error) {
					events = append(events, "create")
					if !settings.DeployKeyInventoryEqual(initial, expected) {
						t.Fatal("lost frozen baseline")
					}
					if mode == "create prewrite" {
						return settings.DeployKeyWriteResult{}, deployPreError{}
					}
					if mode == "create uncertain" {
						return settings.DeployKeyWriteResult{}, errors.New("uncertain")
					}
					if mode == "recovery only" {
						return settings.DeployKeyWriteResult{ID: 2}, errors.New("unverified")
					}
					current, _ = expected.WithCreated(newKey)
					if mode == "wrong create inventory" {
						current = appDeployObservation(t, false, newKey)
					}
					return settings.DeployKeyWriteResult{ID: 2, Observation: current}, nil
				},
				CreatedID: func(collection, identity string, id int64) error {
					events = append(events, "bind")
					if collection != "deploy_keys" || identity != newKey.Key.Fingerprint() || id != 2 {
						t.Fatal("binding by title")
					}
					if mode == "binding failure" {
						return errors.New("changed source")
					}
					return nil
				},
				DeleteDeployKey: func(_ repository.Repository, expected settings.DeployKeyObservation, id int64) (settings.DeployKeyObservation, error) {
					events = append(events, "delete")
					if id != 1 || !expected.HasID(2) || !settings.DeployKeyInventoryEqual(current, expected) {
						t.Fatal("widened candidates or lost own creation")
					}
					if mode == "delete prewrite" {
						return settings.DeployKeyObservation{}, deployPreError{}
					}
					if mode == "delete uncertain" {
						return settings.DeployKeyObservation{}, errors.New("uncertain")
					}
					current, _ = expected.Without(id)
					return current, nil
				},
			}
			wrote, err := applyDeployKeys(&result, repository.Repository{}, &desired, &initial, true, false, writer)
			if (err == nil) != (mode == "success") {
				t.Fatal(err)
			}
			if mode == "success" {
				if fmt.Sprint(events) != "[create bind delete]" || !wrote || current.HasID(1) || !current.HasID(2) {
					t.Fatal(events, wrote)
				}
				return
			}
			if mode == "binding failure" {
				if fmt.Sprint(events) != "[create bind]" || result.Outcome != ApplyPartialFailure || !wrote {
					t.Fatal(events, result, wrote)
				}
				return
			}
			if mode == "fresh drift" {
				if len(events) != 0 || result.Outcome != ApplyFailed {
					t.Fatal(events, result)
				}
				return
			}
			op := result.Operations[len(result.Operations)-1]
			pre := mode == "create prewrite" || mode == "delete prewrite"
			if (op.Status == OperationFailedRead) != pre {
				t.Fatal("incorrect certainty", op)
			}
			prior := mode == "delete prewrite" || mode == "delete uncertain"
			if wrote != prior || (result.Outcome == ApplyPartialFailure) != prior {
				t.Fatal("incorrect progress", wrote, result)
			}
			if mode == "recovery only" && (len(result.CreatedIDs) != 1 || fmt.Sprint(events) != "[create]") {
				t.Fatal("recovery confused with verified binding", result, events)
			}
		})
	}
}
func TestDeployKeyKnownUnarchiveAndNoOp(t *testing.T) {
	state := appDeployState(t, 1, appDeployOne)
	frozen := appDeployObservation(t, true, state)
	active := appDeployObservation(t, false, state)
	desired, _ := settings.NewDeployKeyDeclarations([]settings.DeployKeyDeclaration{state.Declaration()})
	for _, unarchived := range []bool{false, true} {
		result := ApplyResult{Outcome: ApplyFailed}
		wrote, err := applyDeployKeys(&result, repository.Repository{}, &desired, &frozen, true, unarchived, Writer{ReadDeployKeys: func(repository.Repository) (settings.DeployKeyObservation, error) { return active, nil }, CreateDeployKey: func(repository.Repository, settings.DeployKeyObservation, settings.DeployKeyDeclaration) (settings.DeployKeyWriteResult, error) {
			t.Fatal("matching key mutation")
			return settings.DeployKeyWriteResult{}, nil
		}})
		if wrote || (err == nil) != unarchived {
			t.Fatal(wrote, err)
		}
	}
}

func TestDeployKeyExportIndependentRetention(t *testing.T) {
	for _, mode := range []string{"known empty", "unavailable first", "unavailable retained", "readable"} {
		t.Run(mode, func(t *testing.T) {
			state := appDeployState(t, 1, appDeployOne)
			prior, _ := settings.NewDeployKeyDeclarations([]settings.DeployKeyDeclaration{state.Declaration()})
			var retained settings.Declaration
			if mode == "unavailable retained" {
				retained.DeployKeys = &prior
			}
			var saved settings.Declaration
			infos := 0
			checkout := ExportCheckout{Repository: func() (repository.Repository, error) { return repository.Repository{}, nil }, Existing: retained, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(d settings.Declaration) error { saved = d; return nil }, nil
			}, Info: func(path string, _ error) error {
				if path == settings.DeployKeysPath {
					infos++
				}
				return nil
			}}
			reader := Reader{DeployKeys: func(repository.Repository) (settings.DeployKeyObservation, error) {
				if mode == "unavailable first" || mode == "unavailable retained" {
					return settings.DeployKeyObservation{}, errors.New("private raw response")
				}
				if mode == "known empty" {
					return appDeployObservation(t, false), nil
				}
				return appDeployObservation(t, false, state), nil
			}}
			if err := Export(checkout, reader); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "known empty":
				if saved.DeployKeys == nil || saved.DeployKeys.Len() != 0 {
					t.Fatal("known empty lost")
				}
			case "unavailable first":
				if saved.DeployKeys != nil || infos != 1 {
					t.Fatal("unknown guessed empty", infos)
				}
			case "unavailable retained":
				if saved.DeployKeys == nil || saved.DeployKeys.Len() != 1 || infos != 1 {
					t.Fatal("retained intent lost", infos)
				}
			case "readable":
				if saved.DeployKeys == nil || saved.DeployKeys.Len() != 1 {
					t.Fatal("projection lost")
				}
			}
		})
	}
}
