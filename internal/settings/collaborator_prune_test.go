package settings

import (
	"strings"
	"testing"
)

func collaboratorPruneModel(t *testing.T, desired []CollaboratorDeclaration, resolved []CollaboratorIdentity, inviter *CollaboratorInviter) (CollaboratorPruneObservation, error) {
	t.Helper()
	rows := []CollaboratorState{{Person: CollaboratorIdentity{ID: 3, Username: "active"}, Status: "active"}, {Person: CollaboratorIdentity{ID: 4, Username: "pending"}, Status: "pending", InvitationID: 7}}
	observed, err := NewCollaboratorObservation(CollaboratorTarget{ID: 1, OwnerID: 2, FullName: "owner/current"}, rows)
	if err != nil {
		t.Fatal(err)
	}
	entries := []CollaboratorPruneEntry{{State: rows[0]}, {State: rows[1], CreatedAt: "2026-11-14T00:00:00Z", Inviter: inviter}}
	snapshot, err := NewCollaboratorPruneObservation(observed, entries, CollaboratorOperator{Kind: "owner", User: CollaboratorIdentity{ID: 2, Username: "owner"}})
	if err != nil {
		t.Fatal(err)
	}
	declaration, err := NewCollaboratorDeclarations(desired)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot.Prepare(declaration, resolved)
}
func TestCollaboratorPruneRetainedUnionAndDependencies(t *testing.T) {
	owner := &CollaboratorInviter{ID: 2, Login: "owner", Type: "User"}
	active := &CollaboratorInviter{ID: 3, Login: "active", Type: "User"}
	id := int64(3)
	for _, test := range []struct {
		name      string
		desired   []CollaboratorDeclaration
		resolved  []CollaboratorIdentity
		inviter   *CollaboratorInviter
		deletions int
		invalid   bool
	}{
		{name: "empty cancels before remove", inviter: active, deletions: 2},
		{name: "username retained", desired: []CollaboratorDeclaration{{Username: "pending"}}, resolved: []CollaboratorIdentity{{ID: 4, Username: "pending"}}, inviter: owner, deletions: 1},
		{name: "bound and unbound union", desired: []CollaboratorDeclaration{{Username: "active", ID: &id}, {Username: "pending"}}, resolved: []CollaboratorIdentity{{ID: 3, Username: "active"}, {ID: 4, Username: "pending"}}, inviter: active},
		{name: "initial add retained", desired: []CollaboratorDeclaration{{Username: "new"}}, resolved: []CollaboratorIdentity{{ID: 5, Username: "new"}}, inviter: owner, deletions: 2},
		{name: "retained dependency", desired: []CollaboratorDeclaration{{Username: "pending"}}, resolved: []CollaboratorIdentity{{ID: 4, Username: "pending"}}, inviter: active, invalid: true},
		{name: "unknown retained inviter", desired: []CollaboratorDeclaration{{Username: "pending"}}, resolved: []CollaboratorIdentity{{ID: 4, Username: "pending"}}, invalid: true},
		{name: "unknown cancelled inviter", deletions: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := collaboratorPruneModel(t, test.desired, test.resolved, test.inviter)
			if (err != nil) != test.invalid {
				t.Fatal(err)
			}
			if err != nil {
				return
			}
			d := got.Deletions()
			if len(d) != test.deletions {
				t.Fatal(d)
			}
			if len(d) == 2 && (d[0].Operation() != "cancel-invitation" || d[1].Operation() != "remove-collaborator") {
				t.Fatal(d)
			}
			if len(d) > 0 && got.BeforeDelete(d[0]) != nil {
				t.Fatal("fixed first candidate rejected")
			}
		})
	}
}
func TestCollaboratorPruneSelectionPresence(t *testing.T) {
	empty, _ := NewCollaboratorDeclarations(nil)
	for _, scope := range []string{"collaborators", "all", "labels,collaborators,environments"} {
		p, err := ParsePruneScopes(scope)
		if err != nil || !p.Has("collaborators") || p.Manages(Declaration{}) || !p.Manages(Declaration{Collaborators: &empty}) {
			t.Fatal(p, err)
		}
	}
	for _, scope := range []string{"collaborators,collaborators", "all,collaborators", "collaborators,"} {
		if _, err := ParsePruneScopes(scope); err == nil {
			t.Fatal(scope)
		}
	}
}
func TestCollaboratorPruneClosedSnapshotsAndFixedCandidates(t *testing.T) {
	initial, err := collaboratorPruneModel(t, nil, nil, &CollaboratorInviter{ID: 2, Login: "owner", Type: "User"})
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"accepted", "vanished", "replacement", "renamed", "operator", "owner", "created", "expiry", "permission", "late"} {
		t.Run(change, func(t *testing.T) {
			fresh := initial
			fresh.entries = make([]CollaboratorPruneEntry, len(initial.entries))
			for i, row := range initial.entries {
				fresh.entries[i] = cloneCollaboratorPruneEntry(row)
			}
			switch change {
			case "accepted":
				fresh.entries[1].State.Status = "active"
				fresh.entries[1].State.InvitationID = 0
			case "vanished":
				fresh.entries = fresh.entries[:1]
			case "replacement":
				fresh.entries[1].State.InvitationID = 8
			case "renamed":
				fresh.entries[0].State.Person.Username = "renamed"
			case "operator":
				fresh.operator.Kind = "installation"
				fresh.operator.User = CollaboratorIdentity{}
			case "owner":
				fresh.observed.target.OwnerID = 99
			case "created":
				fresh.entries[1].CreatedAt = "2026-11-13T00:00:00Z"
			case "expiry":
				value := false
				fresh.entries[1].Expired = &value
			case "permission":
				fresh.entries[0].Permissions = map[string]bool{"pull": true, "push": true, "admin": false}
			case "late":
				fresh.entries = append(fresh.entries, CollaboratorPruneEntry{State: CollaboratorState{Person: CollaboratorIdentity{ID: 5, Username: "late"}, Status: "active"}})
			}
			if _, err := initial.Refresh(fresh, nil); err == nil {
				t.Fatal("drift adopted")
			}
		})
	}
	d := initial.Deletions()[0]
	after := initial
	after.entries = append([]CollaboratorPruneEntry(nil), initial.entries[:1]...)
	next, err := initial.AfterDelete(after, d)
	if err != nil || len(next.Deletions()) != 1 {
		t.Fatal(err)
	}
	if err := next.BeforeDelete(next.Deletions()[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := next.AfterDelete(after, d); err == nil {
		t.Fatal("reused deletion adopted")
	}
	if strings.Contains(d.Operation(), "private") {
		t.Fatal(d)
	}
}
