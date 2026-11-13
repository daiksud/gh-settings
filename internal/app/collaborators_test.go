package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestCollaboratorApplyBindingAndRemoteProgress(t *testing.T) {
	for _, mode := range []string{"pending binding", "invite", "no writeback", "binding failure", "later failure", "earlier write", "prewrite failure", "uncertain failure", "verified invitation failure"} {
		t.Run(mode, func(t *testing.T) {
			person := settings.CollaboratorIdentity{ID: 3, Username: "person"}
			desired, _ := settings.NewCollaboratorDeclarations([]settings.CollaboratorDeclaration{{Username: "person"}})
			rows := []settings.CollaboratorState{{Person: person, Status: "pending", InvitationID: 7}}
			if mode == "invite" {
				rows = nil
			}
			observed, _ := settings.NewCollaboratorObservation(settings.CollaboratorTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, rows)
			bound, later, written := 0, 0, 0
			declaration := settings.Declaration{Collaborators: &desired}
			if mode == "later failure" {
				declaration.CopilotInstructions = new(bool)
			}
			if mode == "earlier write" {
				declaration.Repository.HasIssues = new(bool)
			}
			writer := Writer{Repository: func(repository.Repository, settings.RepositoryDeclaration) (bool, error) { return true, nil }, ReadCollaborators: func(repository.Repository) (settings.CollaboratorObservation, error) { return observed, nil }, ResolveCollaborator: func(repository.Repository, settings.CollaboratorDeclaration) (settings.CollaboratorIdentity, error) {
				return person, nil
			}, Collaborator: func(_ repository.Repository, selected settings.CollaboratorSelection) (settings.CollaboratorWriteResult, error) {
				written++
				result := settings.CollaboratorWriteResult{Person: person, Status: "pending", InvitationID: 7}
				if mode == "invite" {
					result.Attempted, result.Invited = true, true
				}
				if mode == "prewrite failure" {
					return settings.CollaboratorWriteResult{Person: person}, &collaboratorTestPrerequisite{errors.New("read failed")}
				}
				if mode == "uncertain failure" {
					return settings.CollaboratorWriteResult{Person: person, Attempted: true, InvitationID: 7}, errors.New("unknown")
				}
				if mode == "verified invitation failure" {
					return settings.CollaboratorWriteResult{Person: person, Attempted: true, Invited: true, InvitationID: 7}, errors.New("readback failed")
				}
				return result, nil
			}, BindCollaboratorID: func(username string, id int64) error {
				bound++
				if username != "person" || id != 3 {
					t.Fatal(username, id)
				}
				if mode == "binding failure" || mode == "earlier write" {
					return errors.New("concurrent edit")
				}
				return nil
			}, CopilotInstructions: func(repository.Repository, bool) (bool, error) {
				later++
				return false, &collaboratorTestPrerequisite{errors.New("later read")}
			}}
			if mode == "no writeback" {
				writer.BindCollaboratorID = nil
			}
			result, err := Apply(func() (settings.Declaration, error) { return declaration, nil }, func() (repository.Repository, error) {
				return repository.Repository{Owner: "owner", Name: "current"}, nil
			}, writer)
			failure := mode == "binding failure" || mode == "later failure" || mode == "earlier write" || mode == "prewrite failure" || mode == "uncertain failure" || mode == "verified invitation failure"
			if (err != nil) != failure || written != 1 || len(result.Collaborators) != 1 {
				t.Fatal(result, err, written)
			}
			if failure {
				want := ApplyFailed
				if mode == "earlier write" || mode == "verified invitation failure" {
					want = ApplyPartialFailure
				}
				if result.Outcome != want {
					t.Fatal("local binding confused with remote write", result)
				}
			}
			if mode == "prewrite failure" || mode == "uncertain failure" || mode == "verified invitation failure" {
				if bound != 0 || result.Collaborators[0].BindingVerified {
					t.Fatal("bound failed request")
				}
			} else if !result.Collaborators[0].BindingVerified {
				t.Fatal("lost verified binding")
			}
			if mode == "later failure" && (later != 1 || !result.Collaborators[0].BindingRecorded) {
				t.Fatal("lost earlier local progress", result)
			}
		})
	}
}

type collaboratorTestPrerequisite struct{ error }

func (*collaboratorTestPrerequisite) ReadFailedBeforeWrite() {}

func TestCollaboratorsObserveAfterOwnUnarchive(t *testing.T) {
	archived, desiredArchive := true, false
	person := settings.CollaboratorIdentity{ID: 3, Username: "person"}
	desired, _ := settings.NewCollaboratorDeclarations([]settings.CollaboratorDeclaration{{Username: "person"}})
	result, err := Apply(func() (settings.Declaration, error) {
		return settings.Declaration{Collaborators: &desired, Repository: settings.RepositoryDeclaration{Archived: &desiredArchive}}, nil
	}, func() (repository.Repository, error) {
		return repository.Repository{Owner: "owner", Name: "current"}, nil
	}, Writer{
		ReadArchive: func(repository.Repository) (settings.ArchiveState, error) {
			return settings.ArchiveState{ID: 1, FullName: "owner/current", Archived: archived}, nil
		},
		Archive: func(_ repository.Repository, _ settings.ArchiveState, after bool) error { archived = after; return nil },
		ReadCollaborators: func(repository.Repository) (settings.CollaboratorObservation, error) {
			if archived {
				t.Fatal("collaborators observed before own unarchive")
			}
			return settings.NewCollaboratorObservation(settings.CollaboratorTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, nil)
		},
		ResolveCollaborator: func(repository.Repository, settings.CollaboratorDeclaration) (settings.CollaboratorIdentity, error) {
			return person, nil
		},
		Collaborator: func(_ repository.Repository, selected settings.CollaboratorSelection) (settings.CollaboratorWriteResult, error) {
			if selected.Target.Archived {
				t.Fatal("stale archived target")
			}
			return settings.CollaboratorWriteResult{Person: person, Status: "pending", InvitationID: 7, Attempted: true, Invited: true}, nil
		},
	})
	if err != nil || !result.Completed() || archived {
		t.Fatal(result, err)
	}
}

func TestCollaboratorsPrepareAllAndEmptyNoop(t *testing.T) {
	for _, empty := range []bool{false, true} {
		desired, _ := settings.NewCollaboratorDeclarations([]settings.CollaboratorDeclaration{{Username: "first"}, {Username: "second"}})
		if empty {
			desired, _ = settings.NewCollaboratorDeclarations(nil)
		}
		observed, _ := settings.NewCollaboratorObservation(settings.CollaboratorTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, nil)
		reads, writes := 0, 0
		result, err := Apply(func() (settings.Declaration, error) { return settings.Declaration{Collaborators: &desired}, nil }, func() (repository.Repository, error) { reads++; return repository.Repository{}, nil }, Writer{ReadCollaborators: func(repository.Repository) (settings.CollaboratorObservation, error) { return observed, nil }, ResolveCollaborator: func(_ repository.Repository, d settings.CollaboratorDeclaration) (settings.CollaboratorIdentity, error) {
			if d.Username == "second" {
				return settings.CollaboratorIdentity{}, errors.New("unresolvable")
			}
			return settings.CollaboratorIdentity{ID: 3, Username: d.Username}, nil
		}, Collaborator: func(repository.Repository, settings.CollaboratorSelection) (settings.CollaboratorWriteResult, error) {
			writes++
			return settings.CollaboratorWriteResult{}, nil
		}})
		if writes != 0 || empty && (err != nil || reads != 0 || result.Outcome != ApplyUnmanaged) || !empty && err == nil {
			t.Fatal(result, err, reads, writes)
		}
	}
}
func TestCollaboratorsExportWholeFamily(t *testing.T) {
	id := int64(8)
	prior, _ := settings.NewCollaboratorDeclarations([]settings.CollaboratorDeclaration{{Username: "retained", ID: &id}})
	for _, available := range []bool{false, true} {
		observed, _ := settings.NewCollaboratorObservation(settings.CollaboratorTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, []settings.CollaboratorState{{Person: settings.CollaboratorIdentity{ID: 3, Username: "active"}, Status: "active"}, {Person: settings.CollaboratorIdentity{ID: 4, Username: "pending"}, Status: "pending", InvitationID: 7}})
		var got settings.Declaration
		err := Export(ExportCheckout{Existing: settings.Declaration{Collaborators: &prior}, Repository: func() (repository.Repository, error) { return repository.Repository{}, nil }, PrepareSave: func() (func(settings.Declaration) error, error) {
			return func(d settings.Declaration) error { got = d; return nil }, nil
		}}, Reader{Collaborators: func(repository.Repository) (settings.CollaboratorObservation, error) {
			if !available {
				return settings.CollaboratorObservation{}, errors.New("invitations unavailable")
			}
			return observed, nil
		}})
		if err != nil || got.Collaborators == nil {
			t.Fatal(err)
		}
		values := got.Collaborators.Values()
		if available && (len(values) != 2 || *values[0].ID != 3 || *values[1].ID != 4) || !available && (len(values) != 1 || values[0].Username != "retained") {
			t.Fatal(values)
		}
	}
}
