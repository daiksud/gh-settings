package app

import (
	"errors"
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func appCollaboratorPrune(t *testing.T, desired settings.CollaboratorDeclarations, rows []settings.CollaboratorState, prepare bool) settings.CollaboratorPruneObservation {
	t.Helper()
	target := settings.CollaboratorTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}
	observed, err := settings.NewCollaboratorObservation(target, rows)
	if err != nil {
		t.Fatal(err)
	}
	entries := []settings.CollaboratorPruneEntry{}
	for _, row := range rows {
		entry := settings.CollaboratorPruneEntry{State: row}
		if row.Status == "pending" {
			entry.CreatedAt = "2026-11-14T00:00:00Z"
			entry.Inviter = &settings.CollaboratorInviter{ID: 2, Login: "owner", Type: "User"}
		}
		entries = append(entries, entry)
	}
	current, err := settings.NewCollaboratorPruneObservation(observed, entries, settings.CollaboratorOperator{Kind: "owner", User: settings.CollaboratorIdentity{ID: 2, Username: "owner"}})
	if err != nil {
		t.Fatal(err)
	}
	if prepare {
		resolved := []settings.CollaboratorIdentity{}
		for _, d := range desired.Values() {
			resolved = append(resolved, settings.CollaboratorIdentity{ID: 5, Username: d.Username})
		}
		current, err = current.Prepare(desired, resolved)
		if err != nil {
			t.Fatal(err)
		}
	}
	return current
}
func TestCollaboratorPruneGlobalPreflight(t *testing.T) {
	empty, _ := settings.NewCollaboratorDeclarations(nil)
	prune, _ := settings.ParsePruneScopes("collaborators")
	value := false
	title := "changed"
	writes := 0
	_, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Collaborators: &empty, Repository: settings.RepositoryDeclaration{Archived: &value, Description: &title}}, nil
	}, func() (repository.Repository, error) { return repository.Repository{}, nil }, Writer{
		ReadCollaboratorPrune: func(repository.Repository, settings.CollaboratorDeclarations) (settings.CollaboratorPruneObservation, error) {
			return settings.CollaboratorPruneObservation{}, errors.New("unsafe retained invitation")
		},
		Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) { writes++; return true, nil },
		ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
			t.Fatal("archive observed after failed global preflight")
			return settings.ArchiveState{}, nil
		},
	}, prune)
	if err == nil || writes != 0 {
		t.Fatal(err, writes)
	}
}
func TestCollaboratorPruneOrderingFrozenSelectionAndProgress(t *testing.T) {
	for _, mode := range []string{"success", "binding failure", "first delete prewrite", "first delete uncertain", "first delete acknowledged", "second delete", "prior write"} {
		t.Run(mode, func(t *testing.T) {
			desired, _ := settings.NewCollaboratorDeclarations([]settings.CollaboratorDeclaration{{Username: "kept"}})
			rows := []settings.CollaboratorState{{Person: settings.CollaboratorIdentity{ID: 5, Username: "kept"}, Status: "active"}, {Person: settings.CollaboratorIdentity{ID: 3, Username: "active"}, Status: "active"}, {Person: settings.CollaboratorIdentity{ID: 4, Username: "pending"}, Status: "pending", InvitationID: 7}}
			initial := appCollaboratorPrune(t, desired, rows, true)
			current := initial
			events := []string{}
			deletes := 0
			declaration := settings.Declaration{Collaborators: &desired}
			title := "change"
			if mode == "prior write" {
				declaration.Repository.Description = &title
			}
			write := Writer{
				ReadCollaboratorPrune: func(repository.Repository, settings.CollaboratorDeclarations) (settings.CollaboratorPruneObservation, error) {
					events = append(events, "prepare")
					return initial, nil
				},
				RefreshCollaboratorPrune: func(_ repository.Repository, old settings.CollaboratorPruneObservation, reconciled *settings.CollaboratorWriteResult) (settings.CollaboratorPruneObservation, error) {
					return old.Refresh(appCollaboratorPrune(t, desired, rows, false), reconciled)
				},
				Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) {
					events = append(events, "patch")
					return true, nil
				},
				ResolveCollaborator: func(repository.Repository, settings.CollaboratorDeclaration) (settings.CollaboratorIdentity, error) {
					t.Fatal("resolved a new retained identity after preflight")
					return settings.CollaboratorIdentity{}, nil
				},
				Collaborator: func(_ repository.Repository, s settings.CollaboratorSelection) (settings.CollaboratorWriteResult, error) {
					events = append(events, "reconcile")
					if s.Person.ID != 5 {
						t.Fatal(s)
					}
					return settings.CollaboratorWriteResult{Person: s.Person, Status: "active"}, nil
				},
				BindCollaboratorID: func(_ string, id int64) error {
					events = append(events, "bind")
					if id != 5 {
						t.Fatal(id)
					}
					if mode == "binding failure" {
						return errors.New("file changed")
					}
					return nil
				},
				DeleteCollaborator: func(_ repository.Repository, old settings.CollaboratorPruneObservation, d settings.CollaboratorDeletion) (settings.CollaboratorPruneObservation, settings.CollaboratorDeleteResult, error) {
					events = append(events, d.Operation())
					deletes++
					r := settings.CollaboratorDeleteResult{Deletion: d}
					if mode == "first delete prewrite" || mode == "prior write" {
						return old, r, &collaboratorTestPrerequisite{errors.New("drift")}
					}
					r.Attempted = true
					if mode == "first delete uncertain" || mode == "second delete" && deletes == 2 {
						return old, r, errors.New("unknown")
					}
					r.Acknowledged = true
					if mode == "first delete acknowledged" {
						return old, r, errors.New("readback denied")
					}
					next := []settings.CollaboratorState{}
					for _, row := range rows {
						if row.Person.ID != d.Person.ID {
							next = append(next, row)
						}
					}
					rows = next
					var err error
					current, err = old.AfterDelete(appCollaboratorPrune(t, desired, rows, false), d)
					r.Verified = err == nil
					return current, r, err
				},
			}
			prune, _ := settings.ParsePruneScopes("collaborators")
			result, err := Apply(func() (settings.Declaration, error) { return declaration, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, write, prune)
			if (err == nil) != (mode == "success") {
				t.Fatal(result, err)
			}
			if mode == "success" && fmt.Sprint(events) != "[prepare reconcile bind cancel-invitation remove-collaborator]" {
				t.Fatal(events)
			}
			if mode == "second delete" || mode == "prior write" {
				if result.Outcome != ApplyPartialFailure {
					t.Fatal(result)
				}
			} else if err != nil && result.Outcome != ApplyFailed {
				t.Fatal("local binding or unverified DELETE claimed prior remote completion", result)
			}
			if mode == "binding failure" && deletes != 0 {
				t.Fatal(events)
			}
			if mode == "first delete acknowledged" && (!result.CollaboratorDeletions[0].Acknowledged || result.CollaboratorDeletions[0].Verified) {
				t.Fatal(result)
			}
		})
	}
}

func TestCollaboratorDeleteExplicitAttemptEvidence(t *testing.T) {
	for _, earlierWrite := range []bool{false, true} {
		for _, attempted := range []bool{false, true} {
			for _, invalidResult := range []bool{false, true} {
				t.Run(fmt.Sprintf("earlier=%t/attempted=%t/invalid=%t", earlierWrite, attempted, invalidResult), func(t *testing.T) {
					empty, _ := settings.NewCollaboratorDeclarations(nil)
					rows := []settings.CollaboratorState{{Person: settings.CollaboratorIdentity{ID: 3, Username: "active"}, Status: "active"}}
					initial := appCollaboratorPrune(t, empty, rows, true)
					result := ApplyResult{}
					if earlierWrite {
						result.Operations = append(result.Operations, ApplyOperation{Kind: RepositoryUpdate, Status: OperationCompleted})
					}
					failure := errors.New("configure client failed before request")
					// Even a wrapped prerequisite marker must not override explicit evidence
					// that the deletion request was attempted.
					if attempted {
						failure = &collaboratorTestPrerequisite{errors.New("request effect unknown")}
					}
					if invalidResult {
						failure = nil
					}
					_, err := applyCollaboratorPrune(&result, repository.Repository{Owner: "owner", Name: "current"}, initial, Writer{
						RefreshCollaboratorPrune: func(repository.Repository, settings.CollaboratorPruneObservation, *settings.CollaboratorWriteResult) (settings.CollaboratorPruneObservation, error) {
							return initial, nil
						},
						DeleteCollaborator: func(_ repository.Repository, old settings.CollaboratorPruneObservation, d settings.CollaboratorDeletion) (settings.CollaboratorPruneObservation, settings.CollaboratorDeleteResult, error) {
							return old, settings.CollaboratorDeleteResult{Deletion: d, Attempted: attempted}, failure
						},
					})
					want := OperationFailedRead
					if attempted {
						want = OperationFailedUnknown
					}
					if err == nil || result.Operations[len(result.Operations)-1].Status != want {
						t.Fatal("incorrect attempt classification", result, err)
					}
					outcome := ApplyFailed
					if earlierWrite {
						outcome = ApplyPartialFailure
					}
					if result.Outcome != outcome {
						t.Fatal("earlier verified write classification changed", result)
					}
				})
			}
		}
	}
}
