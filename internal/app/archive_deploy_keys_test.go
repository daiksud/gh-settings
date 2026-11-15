package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestArchivedDeployKeyMatching(t *testing.T) {
	repo := repository.Repository{Owner: "owner", Name: "current"}
	keyText := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEB"
	otherKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIC"
	for _, mode := range []string{"match", "mixed case target", "unknown enabled", "false enabled", "true enabled", "unselected drift", "title", "key", "grant", "missing", "empty observation", "first denied", "fresh denied", "idless", "nil reader", "repo id", "repo name", "owner id", "archive", "final archive", "created", "url", "enabled changed", "prune", "mixed", "mixed differs"} {
		t.Run(mode, func(t *testing.T) {
			state := appDeployState(t, 7, keyText)
			if mode == "false enabled" || mode == "true enabled" || mode == "enabled changed" {
				v := mode == "true enabled"
				state.Enabled = &v
			}
			want := state.Declaration()
			if mode == "idless" {
				want.ID = nil
			}
			desired, err := settings.NewDeployKeyDeclarations([]settings.DeployKeyDeclaration{want})
			if err != nil {
				t.Fatal(err)
			}
			yes := true
			declaration := settings.Declaration{Repository: settings.RepositoryDeclaration{Archived: &yes}, DeployKeys: &desired}
			if mode == "mixed" || mode == "mixed differs" {
				declaration.Topics.Names = &[]string{"one"}
			}
			reads, archives, writes, bindings := 0, 0, 0, 0
			w := Writer{
				ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
					archives++
					id := int64(1)
					if mode == "final archive" && archives == 2 {
						id = 9
					}
					name := "owner/current"
					if mode == "mixed case target" {
						name = "Owner/Current"
					}
					return settings.ArchiveState{ID: id, FullName: name, Archived: true}, nil
				},
				ReadTopics: func(repository.Repository) ([]string, error) {
					if mode == "mixed differs" {
						return []string{"different"}, nil
					}
					return []string{"one"}, nil
				},
				ReadDeployKeys: func(repository.Repository) (settings.DeployKeyObservation, error) {
					reads++
					if mode == "first denied" || mode == "fresh denied" && reads == 2 {
						return settings.DeployKeyObservation{}, errors.New(keyText)
					}
					if mode == "empty observation" {
						return settings.DeployKeyObservation{}, nil
					}
					target := settings.DeployKeyTarget{ID: 1, OwnerID: 2, FullName: "owner/current", Archived: true}
					row := state
					rows := []settings.DeployKeyState{row}
					if reads == 2 {
						switch mode {
						case "title":
							row.Title = "different"
						case "key":
							row.Key, _ = settings.ParseDeployPublicKey(otherKey)
						case "grant":
							row.ReadOnly = !row.ReadOnly
						case "missing":
							rows = nil
						case "repo id":
							target.ID = 9
						case "repo name":
							target.FullName = "owner/other"
						case "owner id":
							target.OwnerID = 9
						case "archive":
							target.Archived = false
						case "created":
							row.CreatedAt = "2026-11-15T00:00:00Z"
						case "url":
							row.APIURL = "https://api.github.com/repos/owner/current/keys/8"
						case "enabled changed":
							v := true
							row.Enabled = &v
						case "unselected drift":
							rows = append(rows, appDeployState(t, 8, otherKey))
						}
						if mode != "missing" {
							rows[0] = row
						}
					}
					return settings.NewDeployKeyObservation(target, rows)
				},
				CreateDeployKey: func(repository.Repository, settings.DeployKeyObservation, settings.DeployKeyDeclaration) (settings.DeployKeyWriteResult, error) {
					writes++
					return settings.DeployKeyWriteResult{}, nil
				},
				DeleteDeployKey: func(repository.Repository, settings.DeployKeyObservation, int64) (settings.DeployKeyObservation, error) {
					writes++
					return settings.DeployKeyObservation{}, nil
				},
				CreatedID: func(string, string, int64) error { bindings++; return nil },
				Archive:   func(repository.Repository, settings.ArchiveState, bool) error { writes++; return nil },
			}
			if mode == "nil reader" {
				w.ReadDeployKeys = nil
			}
			var prune settings.PruneScopes
			if mode == "prune" {
				prune, _ = settings.ParsePruneScopes("deploy-keys")
			}
			result, err := Apply(func() (settings.Declaration, error) { return declaration, nil }, func() (repository.Repository, error) { return repo, nil }, w, prune)
			success := mode == "match" || mode == "mixed case target" || mode == "unknown enabled" || mode == "false enabled" || mode == "true enabled" || mode == "unselected drift" || mode == "mixed"
			if (err == nil) != success || writes != 0 || bindings != 0 || len(result.CreatedIDs) != 0 {
				t.Fatalf("err=%v writes=%d bindings=%d ids=%v", err, writes, bindings, result.CreatedIDs)
			}
			if err != nil && strings.Contains(err.Error(), keyText) {
				t.Fatal("raw public key leaked in error")
			}
		})
	}
}

func TestArchivedDeployKeysEmptyIsUnmanaged(t *testing.T) {
	empty, err := settings.NewDeployKeyDeclarations(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range []*settings.DeployKeyDeclarations{nil, &empty} {
		if err := verifyArchivedDeployKeys(repository.Repository{}, d, settings.ArchiveState{}, nil); err != nil {
			t.Fatal(err)
		}
	}
}
